package main

import (
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
