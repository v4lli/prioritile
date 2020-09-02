package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/schollz/progressbar/v3"
)

type StorageBackend interface {
	GetDirectories(dirname string) ([]string, error)
	GetFiles(dirname string) ([]string, error)
	MkdirAll(dirname string) error
	GetFile(filename string) ([]byte, error)
	PutFile(filename string, content *bytes.Buffer) error
	FileExists(filename string) bool
}

type Job struct {
	source      TilesetDescriptor
	dest        TilesetDescriptor
	relTilePath string
}

func main() {
	numWorkers := flag.Int("parallel", 1, "Number of parallel threads to use for processing")
	quiet := flag.Bool("quiet", false, "Don't output progress information")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: prioritile [-parallel=4] /tiles/target/ /tiles/source1/ [/tiles/source2/ [...]]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "prioritile applies a painter-type algorithm to the first tiles location specified")
		fmt.Fprintln(os.Stderr, "on the commandline in an efficient way by leveraging the XYZ (and WMTS) directory ")
		fmt.Fprintln(os.Stderr, "structure. All trailing tile source directives will be used by the algorithm, in the")
		fmt.Fprintln(os.Stderr, "z-order specified. At least two (one base tileset + one overlay) source directives")
		fmt.Fprintln(os.Stderr, "are required. The zoom levels of all files must be the same.")
		fmt.Fprintln(os.Stderr, "Some assumptions about the source directories:")
		fmt.Fprintln(os.Stderr, "- Tiles are RGBA PNGs")
		fmt.Fprintln(os.Stderr, "- NODATA is represented by 100% alpha")
		fmt.Fprintln(os.Stderr, "")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() < 2 {
		flag.Usage()
		return
	}

	tilesets, err := discoverTilesets(flag.Args())
	if err != nil {
		log.Fatalf("could not discover tilesets: %v", err)
	}

	dest := tilesets[0]
	sources := tilesets[1:]

	// XXX check if input and output are both RGBA
	// XXX check all tiles resolutions to match
	// XXX actually support more than 1 input and 1 output file

	var wg sync.WaitGroup
	jobChan := make(chan Job, 1024)
	for i := 0; i < *numWorkers; i++ {
		wg.Add(1)
		go func(jobChan <-chan Job) {
			defer wg.Done()
			for job := range jobChan {
				if err := processInputTile(job.source, job.dest, job.relTilePath); err != nil {
					panic(err)
				}
			}
		}(jobChan)
	}

	source := sources[0]
	for z := source.MinZ; z <= source.MaxZ; z++ {
		zPart := fmt.Sprintf("%d/", z)
		xDirs, err := source.Backend.GetDirectories(zPart)
		if err != nil {
			panic(err)
		}
		var bar *progressbar.ProgressBar
		if !*quiet {
			log.Printf("Zoom level %d/%d", z, source.MaxZ)
			bar = progressbar.Default(int64(len(xDirs)))
		}
		for _, x := range xDirs {
			xNum, err := strconv.Atoi(x)
			if err != nil {
				panic(err)
			}
			xPart := fmt.Sprintf("%s%d/", zPart, xNum)
			yFiles, err := source.Backend.GetFiles(xPart)
			if err != nil {
				panic(err)
			}
			for _, y := range yFiles {
				if err := dest.Backend.MkdirAll(xPart); err != nil {
					log.Fatal(err)
				}
				jobChan <- Job{
					source:      source,
					dest:        dest,
					relTilePath: xPart + y,
				}
			}
			if !*quiet {
				bar.Add(1)
			}
		}
	}
	close(jobChan)
	wg.Wait()
}

func processInputTile(source, dest TilesetDescriptor, relTilePath string) (err error) {
	f, err := source.Backend.GetFile(relTilePath)
	if err != nil {
		return
	}
	img, _, err := image.Decode(bytes.NewBuffer(f))
	if err != nil {
		log.Println(err.Error())
		log.Println(f)
		return
	}

	skip, hasAlphaPixel := analyzeAlpha(img)
	if skip {
		return nil
	}

	// if the front image has at least one transparent pixel (and exists), merge front and back
	if hasAlphaPixel && dest.Backend.FileExists(relTilePath) {
		destF, err := dest.Backend.GetFile(relTilePath)
		if err != nil {
			return err
		}
		destImg, _, err := image.Decode(bytes.NewBuffer(destF))
		if err != nil {
			return err
		}
		merged := image.NewRGBA(image.Rect(0, 0, destImg.Bounds().Max.X, destImg.Bounds().Max.Y))
		draw.Draw(merged, destImg.Bounds(), destImg, image.Point{0, 0}, draw.Over)
		draw.Draw(merged, img.Bounds(), img, image.Point{0, 0}, draw.Over)

		buf := new(bytes.Buffer)
		if err = png.Encode(buf, merged); err != nil {
			return err
		}
		if err = dest.Backend.PutFile(relTilePath, buf); err != nil {
			return err
		}
	} else {
		// if the front tile completely occludes the back tile, just replace it
		if err = dest.Backend.PutFile(relTilePath, bytes.NewBuffer(f)); err != nil {
			return err
		}
	}

	return nil
}

func newS3Backend(path string) (*S3Backend, error) {
	pathComponents := strings.Split(path[5:], "/")

	minioClient, err := minio.New(pathComponents[0], &minio.Options{
		Creds:  credentials.NewStaticV4(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), ""),
		Secure: true,
	})

	if err != nil {
		return nil, err
	}

	return &S3Backend{
		Client:   minioClient,
		Bucket:   pathComponents[1],
		BasePath: strings.Join(pathComponents[2:], "/"),
	}, nil
}

func stringToBackend(pathSpec string) (StorageBackend, error) {
	// if pathSpec[len(pathSpec)-1] != '/' {
	// 	pathSpec = pathSpec + "/"
	// }

	if strings.HasPrefix(pathSpec, "s3://") {
		backend, err := newS3Backend(pathSpec)
		if err != nil {
			return nil, err
		}
		return backend, nil
	}

	// Default: local filesystem.
	_, err := os.Stat(pathSpec)
	if os.IsNotExist(err) {
		return nil, err
	}
	return &FsBackend{BasePath: pathSpec}, nil
}
