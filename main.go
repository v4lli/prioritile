package main

import (
	"flag"
	"fmt"
	"github.com/schollz/progressbar/v3"
	"image"
	"image/draw"
	"image/png"
	"io"
	"log"
	"os"
	"strconv"
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
				log.Fatalf("Zoom level mismatch between target %s and %s", tilesets[0].Backend.GetBasePath(),
					tilesets[idx].Backend.GetBasePath())
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
		zPart := fmt.Sprintf("/%d/", z)
		zBasePath := source.Backend.GetBasePath() + zPart
		//log.Printf("Entering z=%s\n", zBasePath)
		xDirs, err := source.Backend.GetDirectories(zBasePath)
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
			xBasePath := source.Backend.GetBasePath() + xPart
			//log.Printf("Entering x=%s\n", xBasePath)
			yFiles, err := source.Backend.GetFiles(xBasePath)
			if err != nil {
				panic(err)
			}
			if !*quiet {
				bar.ChangeMax(bar.GetMax() + len(yFiles))
			}
			for _, y := range yFiles {
				if err := source.Backend.MkdirAll(dest.Backend.GetBasePath() + xPart); err != nil {
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
	f, err := source.Backend.GetFileReader(relTilePath)
	if err != nil {
		return
	}

	img, _, err := image.Decode(f)
	if err != nil {
		return
	}

	skip, hasAlphaPixel := analyzeAlpha(img)
	if skip {
		return nil
	}

	// if the front image has at least one transparent pixel (and exists), merge front and back
	if hasAlphaPixel && dest.Backend.FileExists(relTilePath) {
		destF, err := dest.Backend.GetFileReader(relTilePath)
		if err != nil {
			return err
		}

		destImg, _, err := image.Decode(destF)
		if err != nil {
			return err
		}
		merged := image.NewRGBA(image.Rect(0, 0, destImg.Bounds().Max.X, destImg.Bounds().Max.Y))
		draw.Draw(merged, destImg.Bounds(), destImg, image.Point{0, 0}, draw.Over)
		draw.Draw(merged, img.Bounds(), img, image.Point{0, 0}, draw.Over)

		output, err := dest.Backend.GetFileWriter(relTilePath)
		if err != nil {
			return err
		}
		err = png.Encode(output, merged)
		if err != nil {
			return err
		}
	} else {
		// if the front tile completely occludes the back tile, just replace it
		output, err := dest.Backend.GetFileWriter(relTilePath)
		if err != nil {
			return err
		}
		_, err = io.Copy(output, f)
		if err != nil {
			return err
		}
	}

	return nil
}
