package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/schollz/progressbar/v3"
	"github.com/v4lli/prioritile/FsBackend"
	"github.com/v4lli/prioritile/S3Backend"
	"image"
	"image/draw"
	"image/png"
	"log"
	"os"
	"strings"
	"sync"
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
	sources []*TilesetDescriptor
	dest    TilesetDescriptor
	tile    TileDescriptor
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
	tilesDb := make(map[string][]*TilesetDescriptor)
	// composite-key hashmap; could be replaced with some fancy tree in the future, if necessary
	if !*quiet {
		log.Println("Indexing source directories and creating target structure...")
	}
	for idx, tileset := range sources {
		tiles, err := discoverTiles(tileset)
		if err != nil {
			log.Fatal(err)
		}
		for _, tile := range tiles {
			if have, ok := tilesDb[tile.String()]; ok {
				tilesDb[tile.String()] = append(have, &sources[idx])
			} else {
				tilesDb[tile.String()] = []*TilesetDescriptor{&sources[idx]}
			}
			if err := dest.Backend.MkdirAll(fmt.Sprintf("%d/%d/", tile.Z, tile.X)); err != nil {
				log.Fatal(err)
			}
		}
	}

	// XXX check if input and output are both RGBA
	// XXX check all tiles resolutions to match
	var bar *progressbar.ProgressBar

	var wg sync.WaitGroup
	jobChan := make(chan Job, 128)
	for i := 0; i < *numWorkers; i++ {
		wg.Add(1)
		go func(jobChan <-chan Job) {
			defer wg.Done()
			for job := range jobChan {
				if !*quiet {
					bar.Add(1)
				}

				// iterate sources backwards until fully opaque tile has been found, then merge all up to that one
				var toMerge []*image.Image
				opaque := false
				for i := range job.sources {
					backend := job.sources[i].Backend
					f, err := backend.GetFile(job.tile.String())
					if err != nil {
						log.Fatal(err)
					}
					img, _, err := image.Decode(bytes.NewBuffer(f))
					if err != nil {
						log.Fatal(err)
					}

					skip, _ := analyzeAlpha(img)
					if skip {
						continue
					}
					toMerge = append(toMerge, &img)
					// XXX optimize to stop iterating once a opaque tile has been found
					//if !hasAlphaPixel {
					//	//opaque = true
					//	break
					//}
				}

				if !opaque {
					destF, err := dest.Backend.GetFile(job.tile.String())
					if err == nil {
						img, _, err := image.Decode(bytes.NewBuffer(destF))
						if err != nil {
							log.Fatal(err)
						}
						toMerge = append([]*image.Image{&img}, toMerge...)
					}
				}
				if len(toMerge) < 1 {
					continue
				}

				merged := image.NewRGBA(image.Rect(0, 0, (*toMerge[0]).Bounds().Max.X, (*toMerge[0]).Bounds().Max.Y))
				for _, img := range toMerge {
					canvas := image.NewRGBA(image.Rect(0, 0, (*merged).Bounds().Max.X, (*merged).Bounds().Max.Y))
					draw.Draw(canvas, (*merged).Bounds(), merged, image.Point{0, 0}, draw.Over)
					draw.Draw(canvas, (*img).Bounds(), *img, image.Point{0, 0}, draw.Over)
					merged = canvas
				}

				buf := new(bytes.Buffer)
				if err = png.Encode(buf, merged); err != nil {
					log.Fatal(err)
				}
				if err = dest.Backend.PutFile(job.tile.String(), buf); err != nil {
					log.Fatal(err)
				}
			}
		}(jobChan)
	}

	if !*quiet {
		bar = progressbar.Default(int64(len(tilesDb)))
	}

	for key, value := range tilesDb {
		tile, err := Str2Tile(key)
		if err != nil {
			log.Fatal(err)
		}
		jobChan <- Job{
			sources: value,
			dest:    dest,
			tile:    *tile,
		}
	}

	close(jobChan)
	wg.Wait()
}

func stringToBackend(pathSpec string) (StorageBackend, error) {
	if strings.HasPrefix(pathSpec, "s3://") {
		backend, err := S3Backend.NewS3Backend(pathSpec)
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
	return &FsBackend.FsBackend{BasePath: pathSpec}, nil
}
