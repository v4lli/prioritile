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
	atomic "sync/atomic"
	"time"
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

func atomicAverage(target *int64, channel *chan time.Duration) {
	// This is more fun than a locked section to take care about atomic stuff
	for i := range *channel {
		*target = (*target + i.Nanoseconds()) / 2
	}
}

func main() {
	numWorkers := flag.Int("parallel", 1, "Number of parallel threads to use for processing")
	quiet := flag.Bool("quiet", false, "Don't output progress information")
	debug := flag.Bool("debug", false, "Enable debugging (tracing and some perf counters)")
	report := flag.Bool("report", false, "Enable periodic reports (every min)")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: prioritile [-debug] [-report] [-parallel=4] /tiles/target/ /tiles/source1/ [/tiles/source2/ [...]]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "prioritile applies a painter-type algorithm to the first tiles location specified")
		fmt.Fprintln(os.Stderr, "on the commandline in an efficient way by leveraging the XYZ (and WMTS) directory ")
		fmt.Fprintln(os.Stderr, "structure. All trailing tile source directives will be used by the algorithm, in the")
		fmt.Fprintln(os.Stderr, "z-order specified. At least two (one base tileset + one overlay) source directives")
		fmt.Fprintln(os.Stderr, "are required. The zoom levels of all files must be the same.")
		fmt.Fprintln(os.Stderr, "Some assumptions about the source directories:")
		fmt.Fprintln(os.Stderr, "- Tiles are RGBA PNGs")
		fmt.Fprintln(os.Stderr, "- NODATA is represented by 100% alpha")
		fmt.Fprintln(os.Stderr, "- Resolution of corresponding tiles matches")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "S3 disk backends are supported as source and target by prefixing the tile")
		fmt.Fprintln(os.Stderr, "directories with 's3://', e.g. 's3://example.com/source/'.")
		fmt.Fprintln(os.Stderr, "S3 authentication information is read from environment variables:")
		fmt.Fprintln(os.Stderr, "AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY")
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

	// Performance measurement setup
	counterBackwardsIteration := make(chan time.Duration, 1024)
	var counterBackwardsIterationDurationNS int64
	go atomicAverage(&counterBackwardsIterationDurationNS, &counterBackwardsIteration)
	counterOpaquenessCheck := make(chan time.Duration, 1024)
	var counterOpaquenessCheckNS int64
	go atomicAverage(&counterOpaquenessCheckNS, &counterOpaquenessCheck)
	counterAlphaCheck := make(chan time.Duration, 1024)
	var counterAlphaCheckNS int64
	go atomicAverage(&counterAlphaCheckNS, &counterAlphaCheck)
	counterDraw := make(chan time.Duration, 1024)
	var counterDrawNS int64
	go atomicAverage(&counterDrawNS, &counterDraw)
	counterEncode := make(chan time.Duration, 1024)
	var counterEncodeNS int64
	go atomicAverage(&counterEncodeNS, &counterEncode)

	var wg sync.WaitGroup
	jobChan := make(chan Job, 128)
	var iterationCounter int32
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
				startBackwardsIteration := time.Now()
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

					counterAlphaCheckStart := time.Now()
					skip, _ := analyzeAlpha(img)
					counterAlphaCheck <- time.Since(counterAlphaCheckStart)
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
				counterBackwardsIteration <- time.Since(startBackwardsIteration)

				counterOpaquenessCheckStart := time.Now()
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
				counterOpaquenessCheck <- time.Since(counterOpaquenessCheckStart)
				if len(toMerge) < 1 {
					continue
				}

				counterDrawStart := time.Now()
				merged := image.NewRGBA(image.Rect(0, 0, (*toMerge[0]).Bounds().Max.X, (*toMerge[0]).Bounds().Max.Y))
				for _, img := range toMerge {
					canvas := image.NewRGBA(image.Rect(0, 0, (*merged).Bounds().Max.X, (*merged).Bounds().Max.Y))
					draw.Draw(canvas, (*merged).Bounds(), merged, image.Point{0, 0}, draw.Over)
					draw.Draw(canvas, (*img).Bounds(), *img, image.Point{0, 0}, draw.Over)
					merged = canvas
				}
				counterDraw <- time.Since(counterDrawStart)

				counterEncodeStart := time.Now()
				buf := new(bytes.Buffer)
				if err = png.Encode(buf, merged); err != nil {
					log.Fatal(err)
				}
				if err = dest.Backend.PutFile(job.tile.String(), buf); err != nil {
					log.Fatal(err)
				}
				counterEncode <- time.Since(counterEncodeStart)
				atomic.AddInt32(&iterationCounter, 1)
			}
		}(jobChan)
	}

	if !*quiet {
		bar = progressbar.Default(int64(len(tilesDb)))
	}

	if *report {
		go func() {
			log.Printf("Progress: %d of %d total\n", iterationCounter, len(tilesDb))
			time.Sleep(60 * time.Second)
		}()
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
	if *debug {
		fmt.Printf("Average Backwards Iteration: %s\n", time.Duration(counterBackwardsIterationDurationNS/1000/1000))
		fmt.Printf("Average Opaqueness Check: %s\n", time.Duration(counterOpaquenessCheckNS/1000/1000))
		fmt.Printf("\\_Average Alpa Check: %s\n", time.Duration(counterAlphaCheckNS/1000/1000))
		fmt.Printf("Average Draw: %s\n", time.Duration(counterDrawNS/1000/1000))
		fmt.Printf("Average Encode: %s\n", time.Duration(counterEncodeNS/1000/1000))
	}
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
