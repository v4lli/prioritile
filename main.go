package main

import (
	"flag"
	"fmt"
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

	//dest := flag.Args()[0]
	sources := flag.Args()[1:]

	//log.Println(input)
	//log.Println(sources)

	//destDesc := discoverTileset(dest)
	sourcesDesc := make([]TilesetDescriptor, len(sources))
	for idx, source := range sources {
		sourcesDesc[idx] = discoverTileset(source)
	}

	source := sourcesDesc[0]
	for z := source.minZ; z < source.maxZ; z++ {
		zBasePath := fmt.Sprintf("%s/%d/", source.basePath, z)
		log.Printf("Entering z=%s\n", zBasePath)
		xDirs, err := ioutil.ReadDir(zBasePath)
		if err != nil {
			log.Fatal(err)
			return
		}
		for _, x := range xDirs {
			if x.IsDir() {
				xNum, err := strconv.Atoi(x.Name())
				if err != nil {
					fmt.Println(err)
					return
				}
				xBasePath := fmt.Sprintf("%s%d/", zBasePath, xNum)
				log.Printf("Entering x=%s\n", xBasePath)
				yFiles, err := ioutil.ReadDir(xBasePath)
				if err != nil {
					log.Fatal(err)
					break
				}
				for _, y := range yFiles {
					processInputTile(xBasePath + y.Name())
				}
			}
		}
	}
}

func processInputTile(relTilePath string) {
	// XXX read tile, check if it has at least one pixel with alpha
	// if no, replace target with source and be done.
	// if yes, read target, do alpha blending and then write back target
}

type TilesetDescriptor struct {
	maxZ     int
	minZ     int
	basePath string
}

// Read
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
