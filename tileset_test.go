package main

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBuildTilesetStructure(t *testing.T) {
	files := []string{
		"5/15/19.png",
		"5/16/19.png",
		"6/31/38.png",
		"6/32/38.png",
		"7/63/77.png",
		"7/64/77.png",
		"8/127/154.png",
		"8/127/155.png",
		"8/128/154.png",
		"8/128/155.png",
		"9/254/309.png",
		"9/254/310.png",
		"9/255/309.png",
		"9/255/310.png",
		"9/255/311.png",
		"9/256/309.png",
		"9/256/310.png",
		"9/256/311.png",
	}
	tileset := TilesetDescriptor{}
	err := buildTilesetStructure(files, &tileset)
	require.NoError(t, err)
	require.Len(t, tileset.Tiles, 5)
	// Zoom 5
	tiles := tileset.Tiles[5]
	assert.Len(t, tiles, 2)
	// Zoom 6
	tiles = tileset.Tiles[6]
	assert.Len(t, tiles, 2)
	// Zoom 7
	tiles = tileset.Tiles[7]
	assert.Len(t, tiles, 2)
	// Zoom 8
	tiles = tileset.Tiles[8]
	assert.Len(t, tiles, 4)
	// Zoom 9
	tiles = tileset.Tiles[9]
	assert.Len(t, tiles, 8)
}

func TestBuildTilesetStructureMinMax(t *testing.T) {
	files := []string{
		"5/15/19.png",
		"5/16/19.png",
		"6/31/38.png",
		"6/32/38.png",
		"7/63/77.png",
		"7/64/77.png",
		"8/127/154.png",
		"8/127/155.png",
		"8/128/154.png",
		"8/128/155.png",
		"9/254/309.png",
		"9/254/310.png",
		"9/255/309.png",
		"9/255/310.png",
		"9/255/311.png",
		"9/256/309.png",
		"9/256/310.png",
		"9/256/311.png",
	}
	tileset := TilesetDescriptor{
		MinZ: 7,
		MaxZ: 8,
	}
	err := buildTilesetStructure(files, &tileset)
	require.NoError(t, err)
	require.Len(t, tileset.Tiles, 2)
	// Zoom 7
	tiles := tileset.Tiles[7]
	assert.Len(t, tiles, 2)
	// Zoom 8
	tiles = tileset.Tiles[8]
	assert.Len(t, tiles, 4)
}

func TestBuildTilesetStructureInvalid(t *testing.T) {
	files := []string{
		"6/",
		"6/31/",
		"6/31/38.png",
		"6/32/",
		"6/32/38.png",
	}
	tileset := TilesetDescriptor{}
	err := buildTilesetStructure(files, &tileset)
	require.Error(t, err)
}
