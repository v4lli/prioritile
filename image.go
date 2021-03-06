package main

import (
	"image"
)

func analyzeAlpha(img image.Image) (bool, bool) {
	skip := true
	hasAlphaPixel := false
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a == 65535 || a == 0xff {
				skip = false
				if hasAlphaPixel {
					return skip, hasAlphaPixel // return early
				}
			} else {
				hasAlphaPixel = true
				if !skip {
					return skip, hasAlphaPixel // return early
				}
			}
		}
	}
	return skip, hasAlphaPixel
}
