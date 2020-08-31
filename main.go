package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/schollz/progressbar/v3"
	"image"
	"image/draw"
	"image/png"
	"log"
	"os"
	"strconv"
	"strings"
)

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

	if len(flag.Args()) < 2 {
		flag.Usage()
		return
	}

	log.Println(*numWorkers)

	tilesets := make([]TilesetDescriptor, len(flag.Args()))
	for idx, pathSpec := range flag.Args() {
		backend, err := stringToBackend(pathSpec)
		if err != nil {
			log.Fatal(err)
		}
		tilesets[idx] = discoverTileset(backend)

		if idx > 0 {
			if tilesets[0].MaxZ != tilesets[idx].MaxZ || tilesets[0].MinZ != tilesets[idx].MinZ {
				log.Fatalf("Zoom level mismatch for target %s\n", pathSpec)
			}
		}
	}

	dest := tilesets[0]
	sources := tilesets[1:]

	//log.Println(input)
	//log.Println(sources)

	// XXX check if input and output are both RGBA
	// XXX check all tiles resolutions to match
	// XXX actually support more than 1 input and 1 output file

	source := sources[0]
	var bar *progressbar.ProgressBar
	if !*quiet {
		bar = progressbar.Default(1)
	}
	for z := source.MinZ; z <= source.MaxZ; z++ {
		zPart := fmt.Sprintf("%d/", z)
		xDirs, err := source.Backend.GetDirectories(zPart)
		if err != nil {
			panic(err)
		}
		//log.Printf("Zoom level %d/%d", z, source.MaxZ)
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
			if !*quiet {
				bar.ChangeMax(bar.GetMax() + len(yFiles))
			}
			for _, y := range yFiles {
				if err := dest.Backend.MkdirAll(xPart); err != nil {
					log.Fatal(err)
				}
				if err := processInputTile(source, dest, xPart+y); err != nil {
					log.Fatal(err)
				}
				if !*quiet {
					bar.Add(1)
				}
			}
		}
	}
	if !*quiet {
		bar.Add(1)
	}
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

func stringToBackend(input string) (StorageBackend, error) {
	pathSpec := input
	if pathSpec[len(pathSpec)-1] != '/' {
		pathSpec = pathSpec + "/"
	}

	if pathSpec[0:5] == "s3://" {
		// Extract host & bucket
		pathComponents := strings.Split(pathSpec[5:], "/")

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

	// Default: local filesystem.
	_, err := os.Stat(pathSpec)
	if os.IsNotExist(err) {
		return nil, err
	}
	return &FsBackend{BasePath: pathSpec}, nil
}

type StorageBackend interface {
	GetDirectories(dirname string) ([]string, error)
	GetFiles(dirname string) ([]string, error)
	MkdirAll(dirname string) error
	GetFile(filename string) ([]byte, error)
	PutFile(filename string, content *bytes.Buffer) error
	FileExists(filename string) bool
}
