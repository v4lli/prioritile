package main

import (
	"fmt"
	"log"
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

func discoverTilesets(paths []string, target TilesetDescriptor, bestEffort bool) ([]TilesetDescriptor, []error) {
	var tilesets []TilesetDescriptor
	var errors []error

	for _, path := range paths {
		backend, err := stringToBackend(path)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		tileset, err := discoverTileset(backend, target.MinZ, target.MaxZ)

		if err != nil {
			errors = append(errors, fmt.Errorf("could not discover tileset: %v in %s", err, path))
			continue
		}

		if len(tilesets) > 0 && (target.MaxZ != tileset.MaxZ || target.MinZ != tileset.MinZ) {
			errors = append(errors, fmt.Errorf("zoom level mismatch for target and source %s", path))
			if !bestEffort {
				continue
			}
		}
		tilesets = append(tilesets, tileset)
	}
	return tilesets, errors
}

func discoverTileset(backend StorageBackend, minZ int, maxZ int) (TilesetDescriptor, error) {
	files, err := backend.GetDirectories("")
	if err != nil {
		return TilesetDescriptor{}, err
	}

	var z []int
	for _, f := range files {
		i, err := strconv.Atoi(f)
		if i < minZ || (i > maxZ && maxZ != -1) {
			continue
		}
		if err == nil {
			z = append(z, i)
		} else {
			log.Printf("Invalid file '%s'", f)
		}
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
