package main

import (
	"fmt"
	"sort"
	"strconv"
)

type TilesetDescriptor struct {
	MaxZ    int
	MinZ    int
	Backend StorageBackend
}

func (t TilesetDescriptor) String() string {
	return fmt.Sprintf("%d-%d", t.MaxZ, t.MinZ)
}

func discoverTilesets(paths []string) ([]TilesetDescriptor, error) {
	tilesets := make([]TilesetDescriptor, len(paths))
	for i, path := range paths {
		backend, err := stringToBackend(path)
		if err != nil {
			return nil, err
		}

		tilesets[i], err = discoverTileset(backend)
		if err != nil {
			return nil, fmt.Errorf("could not discover tileset: %v", err)
		}

		if i > 0 && (tilesets[0].MaxZ != tilesets[i].MaxZ || tilesets[0].MinZ != tilesets[i].MinZ) {
			return nil, fmt.Errorf("Zoom level mismatch for target %s", path)
		}
	}
	return tilesets, nil
}

func discoverTileset(backend StorageBackend) (TilesetDescriptor, error) {
	files, err := backend.GetDirectories("")
	if err != nil {
		return TilesetDescriptor{}, err
	}

	var z []int
	for _, f := range files {
		i, err := strconv.Atoi(f)
		if err != nil {
			return TilesetDescriptor{}, err
		}
		z = append(z, i)
	}
	if z == nil {
		return TilesetDescriptor{}, fmt.Errorf("invalid or empty tileset")
	}
	sort.Ints(z)

	return TilesetDescriptor{
		MinZ:    z[0],
		MaxZ:    z[len(z)-1],
		Backend: backend,
	}, nil
}
