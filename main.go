package main

import (
	"flag"
	"fmt"
	"github.com/schollz/progressbar/v3"
	"image"
	"image/draw"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
)

func main() {
	numWorkers := flag.Int("parallel", 1, "Number of parallel threads to use for processing")
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

	log.Println(numWorkers)

	tilesets := make([]TilesetDescriptor, len(flag.Args()))
	for idx, pathSpec := range flag.Args() {
		backend, err := stringToBackend(pathSpec)
		if err != nil {
			log.Fatal(err)
		}
		tilesets[idx] = discoverTileset(backend)
	}

	dest := tilesets[0]
	sources := tilesets[1:]

	//log.Println(input)
	//log.Println(sources)

	// XXX check if input and output are both RGBA
	// XXX check all tiles resoltuions to match
	// XXX actually support more than 1 input and 1 output file

	source := sources[0]
	for z := source.MinZ; z <= source.MaxZ; z++ {
		zPart := fmt.Sprintf("/%d/", z)
		zBasePath := source.Backend.GetBasePath() + zPart
		//log.Printf("Entering z=%s\n", zBasePath)
		xDirs, err := ioutil.ReadDir(zBasePath)
		if err != nil {
			panic(err)
		}
		log.Printf("Zoom level %d/%d", z, source.MaxZ)
		bar := progressbar.Default(int64(len(xDirs)))
		for _, x := range xDirs {
			if x.IsDir() {
				xNum, err := strconv.Atoi(x.Name())
				if err != nil {
					panic(err)
				}
				xPart := fmt.Sprintf("%s%d/", zPart, xNum)
				xBasePath := source.Backend.GetBasePath() + xPart
				//log.Printf("Entering x=%s\n", xBasePath)
				yFiles, err := ioutil.ReadDir(xBasePath)
				if err != nil {
					panic(err)
				}
				for _, y := range yFiles {
					err := os.MkdirAll(dest.Backend.GetBasePath()+xPart, os.ModePerm)
					if err != nil {
						panic(err)
					}
					err = processInputTile(source, dest, xPart+y.Name())
					if err != nil {
						panic(err)
					}
				}
			}
			bar.Add(1)
		}
	}
}

func processInputTile(source, dest TilesetDescriptor, relTilePath string) (err error) {
	f, err := os.Open(source.Backend.GetBasePath() + relTilePath)
	if err != nil {
		return
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return
	}

	hasAlphaPixel, skip := analyzeAlpha(img)

	if skip {
		return nil
	}

	// if the front image has at least one transparent pixel (and exists), merge front and back
	_, err = os.Stat(dest.Backend.GetBasePath() + relTilePath)
	if hasAlphaPixel && err == nil {
		//log.Println(dest.BasePath + relTilePath)
		destF, err := os.Open(dest.Backend.GetBasePath() + relTilePath)
		if err != nil {
			return err
		}

		destImg, _, err := image.Decode(destF)
		if err != nil {
			return err
		}
		output := image.NewRGBA(image.Rect(0, 0, destImg.Bounds().Max.X, destImg.Bounds().Max.Y))
		draw.Draw(output, destImg.Bounds(), destImg, image.Point{0, 0}, draw.Over)
		draw.Draw(output, img.Bounds(), img, image.Point{0, 0}, draw.Over)
		destF.Close()

		destF, err = os.Create(dest.Backend.GetBasePath() + relTilePath)
		if err != nil {
			return err
		}
		err = png.Encode(destF, output)
		if err != nil {
			return err
		}
	} else {
		// if the front tile completely occludes the back tile, just replace it
		f.Seek(0, io.SeekStart)
		outfile, err := os.Create(dest.Backend.GetBasePath() + relTilePath)
		defer outfile.Close()
		if err != nil {
			return err
		}
		_, err = outfile.ReadFrom(f)
		if err != nil {
			return err
		}
	}

	return nil
}
