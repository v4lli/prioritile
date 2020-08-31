package main

import (
	"sort"
	"strconv"
)

type TilesetDescriptor struct {
	MaxZ    int
	MinZ    int
	Backend StorageBackend
}

func discoverTileset(backend StorageBackend) TilesetDescriptor {
	files, err := backend.GetDirectories("")
	if err != nil {
		// XXX inconsistent, should return err
		panic(err)
	}

	var z []int
	for _, f := range files {
		if i, err := strconv.Atoi(f); err == nil {
			z = append(z, i)
		}
	}
	if z == nil {
		panic("Invalid or empty tileset")
	}
	sort.Ints(z)

	return TilesetDescriptor{
		MinZ:    z[0],
		MaxZ:    z[len(z)-1],
		Backend: backend,
	}
}
