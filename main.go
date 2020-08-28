package main

import (
	"flag"
	"fmt"
	"github.com/schollz/progressbar"
	"image"
	"image/draw"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strconv"
)

func main() {
	storageBackend := flag.String("storage", "fs", "Storage backend to use, one of 'fs' (default) or 's3'. S3 backend requires authentication information to be available through environment variables")
	numWorkers := flag.Int("parallel", 1, "Number of parallel threads to use for processing")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: prioritile [-parallel=4] [-storage=fs] /tiles/target/ /tiles/source1/ [/tiles/source2/ [...]]\n")
		fmt.Fprintln(os.Stderr, "prioritile applies a painter-type algorithm to the first tiles location specified")
		fmt.Fprintln(os.Stderr, "on the commandline in an efficient way by leveraging the XYZ (and WMTS) directory ")
		fmt.Fprintln(os.Stderr, "structure. All trailing tile source directives will be used by the algorithm, in the")
		fmt.Fprintln(os.Stderr, "z-order specified. At least two (one base tileset + one overlay) source directives")
		fmt.Fprintln(os.Stderr, "are required. The zoom levels of all files must be the same.")
		fmt.Fprintln(os.Stderr, "Some assumptions about the source directories:")
		fmt.Fprintln(os.Stderr, "- Tiles are RGBA PNGs")
		fmt.Fprintln(os.Stderr, "- NODATA is represented by 100% alpha\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	log.Println(*storageBackend)
	log.Println(*numWorkers)

	if len(flag.Args()) < 2 {
		flag.Usage()
		return
	}

	dest := flag.Args()[0]
	sources := flag.Args()[1:]

	//log.Println(input)
	//log.Println(sources)

	// XXX check if input and output are both RGBA
	// XXX check all tiles resoltuions to match
	// XXX actually support more than 1 input and 1 output file

	destDesc := discoverTileset(dest)
	sourcesDesc := make([]TilesetDescriptor, len(sources))
	for idx, source := range sources {
		sourcesDesc[idx] = discoverTileset(source)
	}

	source := sourcesDesc[0]
	for z := source.minZ; z <= source.maxZ; z++ {
		zPart := fmt.Sprintf("/%d/", z)
		zBasePath := source.basePath + zPart
		//log.Printf("Entering z=%s\n", zBasePath)
		xDirs, err := ioutil.ReadDir(zBasePath)
		if err != nil {
			panic(err)
		}
		log.Printf("Zoom level %d/%d", z, source.maxZ)
		bar := progressbar.Default(int64(len(xDirs)))
		for _, x := range xDirs {
			if x.IsDir() {
				xNum, err := strconv.Atoi(x.Name())
				if err != nil {
					panic(err)
				}
				xPart := fmt.Sprintf("%s%d/", zPart, xNum)
				xBasePath := source.basePath + xPart
				//log.Printf("Entering x=%s\n", xBasePath)
				yFiles, err := ioutil.ReadDir(xBasePath)
				if err != nil {
					panic(err)
				}
				for _, y := range yFiles {
					err := os.MkdirAll(destDesc.basePath+xPart, os.ModePerm)
					if err != nil {
						panic(err)
					}
					err = processInputTile(source, destDesc, xPart+y.Name())
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
	f, err := os.Open(source.basePath + relTilePath)
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
	_, err = os.Stat(dest.basePath + relTilePath)
	if hasAlphaPixel && err == nil {
		//log.Println(dest.basePath + relTilePath)
		destF, err := os.Open(dest.basePath + relTilePath)
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

		destF, err = os.Create(dest.basePath + relTilePath)
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
		outfile, err := os.Create(dest.basePath + relTilePath)
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

func analyzeAlpha(img image.Image) (hasAlphaPixel, skip bool) {
	skip = true
	hasAlphaPixel = false
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a == 65535 {
				skip = false
			} else {
				hasAlphaPixel = true
			}
			// XXX could return early if one non-alpha pixel and one alpha pixel has been found.
		}
	}
	return hasAlphaPixel, skip
}

type TilesetDescriptor struct {
	maxZ     int
	minZ     int
	basePath string
}

func discoverTileset(path string) TilesetDescriptor {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		panic(err)
	}

	var z []int
	for _, f := range files {
		if f.IsDir() {
			if i, err := strconv.Atoi(f.Name()); err == nil {
				z = append(z, i)
			}
		}
	}
	if z == nil {
		panic("Invalid or empty tileset")
	}
	sort.Ints(z)

	basePath := path
	if path[len(path)-1] != '/' {
		basePath = basePath + "/"
	}

	return TilesetDescriptor{
		minZ:     z[0],
		maxZ:     z[len(z)-1],
		basePath: basePath,
	}
}
