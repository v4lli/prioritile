package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type TilesetDescriptor struct {
	MaxZ    int
	MinZ    int
	Backend StorageBackend
	Tiles   map[int][]TileDescriptor // <zoom, []tiles> mapping
}

func (t TilesetDescriptor) GetTiles() []TileDescriptor {
	// Copy and sort keys of map
	keys := make([]int, len(t.Tiles))
	i := 0
	for k := range t.Tiles {
		keys[i] = k
		i += 1
	}
	sort.Ints(keys)
	// Extract all tiles to a sorted slice
	var result []TileDescriptor
	for k := range keys {
		tileset := t.Tiles[k]
		for _, s := range tileset {
			result = append(result, s)
		}
	}
	return result
}

func (t TilesetDescriptor) String() string {
	return fmt.Sprintf("%d-%d", t.MaxZ, t.MinZ)
}

func discoverTilesets(paths []string, target TilesetDescriptor, bestEffort bool, timeout int) ([]TilesetDescriptor, []error) {
	var tilesets []TilesetDescriptor
	var errors []error

	for _, path := range paths {
		backend, err := stringToBackend(path, true, timeout)
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
	files, err := backend.GetFilesRecursive("")
	if err != nil {
		return TilesetDescriptor{}, err
	}

	result := TilesetDescriptor{
		MinZ:    minZ,
		MaxZ:    maxZ,
		Backend: backend,
	}
	err = buildTilesetStructure(files, &result)
	if err != nil {
		return TilesetDescriptor{}, fmt.Errorf("invalid or empty tileset: %w", err)
	}
	return result, nil
}

// Assumes the passed list of files is already sorted alphabetically.
// Returns the respective Z/X/Y.png structure.
func buildTilesetStructure(files []string, tileset *TilesetDescriptor) error {
	var currentZoomLevel string
	var tiles []TileDescriptor
	tileset.Tiles = map[int][]TileDescriptor{}
	for _, f := range files {
		pathParts := strings.Split(f, "/")
		if len(pathParts) != 3 {
			return fmt.Errorf("invalid file path %s, expected format {z}/{x}/{y}.<ext>", f)
		}
		z := pathParts[0]
		zNum, err := strconv.Atoi(z)
		if err != nil {
			return err
		}
		x := pathParts[1]
		xNum, _ := strconv.Atoi(x)
		if err != nil {
			return err
		}
		y := pathParts[2]
		// Get format
		fileParts := strings.Split(y, ".")
		if len(fileParts) != 2 {
			return fmt.Errorf("invalid file path %s, expected format {z}/{x}/{y}.<ext>", f)
		}
		yNum, _ := strconv.Atoi(fileParts[0])
		if err != nil {
			return err
		}
		format := fileParts[1]
		// Check zoom level
		if z != currentZoomLevel {
			// New zoom level
			if zNum < tileset.MinZ || (tileset.MaxZ > 0 && zNum > tileset.MaxZ) {
				// Filter out file based on specified zoom level boundaries
				continue
			}
			currentZoomLevel = z
			tiles = []TileDescriptor{}
		}
		// Add Z/X/Y (file)
		tiles = append(tiles, TileDescriptor{
			X:       xNum,
			Y:       yNum,
			Z:       zNum,
			Format:  format,
			TileSet: tileset,
		})
		tileset.Tiles[zNum] = tiles
	}
	return nil
}
