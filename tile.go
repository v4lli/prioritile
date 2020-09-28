package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type TileDescriptor struct {
	X       int
	Y       int
	Z       int
	Format  string
	TileSet *TilesetDescriptor
}

func (p TileDescriptor) String() string {
	return fmt.Sprintf("%d/%d/%d.%s", p.Z, p.X, p.Y, p.Format)
}

func discoverTiles(tileset TilesetDescriptor) ([]TileDescriptor, error) {
	var result []TileDescriptor
	for z := tileset.MinZ; z <= tileset.MaxZ; z++ {
		zPart := fmt.Sprintf("%d/", z)
		xDirs, err := tileset.Backend.GetDirectories(zPart)
		if err != nil {
			return nil, err
		}
		for _, x := range xDirs {
			xNum, err := strconv.Atoi(x)
			if err != nil {
				return nil, err
			}
			xPart := fmt.Sprintf("%s%d/", zPart, xNum)
			yFiles, err := tileset.Backend.GetFiles(xPart)
			if err != nil {
				return nil, err
			}
			for _, y := range yFiles {
				ySplit := strings.Split(y, ".")
				if len(ySplit) != 2 {
					return nil, errors.New("unknown file in tile dir")
				}
				yNum, err := strconv.Atoi(ySplit[0])
				if err != nil {
					return nil, err
				}
				result = append(result, TileDescriptor{
					X:      xNum,
					Y:      yNum,
					Z:      z,
					Format: ySplit[1],
				})
			}
		}
	}
	return result, nil
}

func Str2Tile(tileSpec string) (*TileDescriptor, error) {
	parts := strings.Split(tileSpec, "/")
	yParts := strings.Split(parts[2], ".")

	z, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, err
	}
	x, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, err
	}
	y, err := strconv.Atoi(yParts[0])
	if err != nil {
		return nil, err
	}

	return &TileDescriptor{Z: z, X: x, Y: y, Format: yParts[1]}, nil
}
