package main

import (
	"image"
	"image/color"
	"testing"
)

func TestAnalyzeAlpha(t *testing.T) {
	// Define list of rectangles and if they should be contained within a
	var testCases = []struct {
		Image                [][]color.RGBA
		ShouldHaveAlphaPixel bool
		ShouldSkip           bool
	}{
		{[][]color.RGBA{
			{{R: 0xff, G: 0xff, B: 0xff, A: 0xff}, {R: 0xff, G: 0xff, B: 0xff, A: 0xff}},
			{{R: 0xff, G: 0xff, B: 0xff, A: 0xff}, {R: 0xff, G: 0xff, B: 0xff, A: 0xff}},
		}, false, false},
		{[][]color.RGBA{
			{{R: 0xff, G: 0xff, B: 0xff, A: 0}, {R: 0xff, G: 0xff, B: 0xff, A: 0}},
			{{R: 0xff, G: 0xff, B: 0xff, A: 0}, {R: 0xff, G: 0xff, B: 0xff, A: 0}},
		}, true, true},
		{[][]color.RGBA{
			{{R: 0xff, G: 0xff, B: 0xff, A: 0xff}, {R: 0xff, G: 0xff, B: 0xff, A: 0}},
			{{R: 0xff, G: 0xff, B: 0xff, A: 0}, {R: 0xff, G: 0xff, B: 0xff, A: 0}},
		}, true, false},
	}
	for idx, tc := range testCases {
		img := image.NewRGBA(image.Rect(0, 0, 2, 2))
		for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
			for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
				img.SetRGBA(x, y, tc.Image[x][y])
			}
		}
		skip, hasAlpha := analyzeAlpha(img)
		t.Logf("Test Case %d => hasAlphaPixel=%t (expecting %t) skip=%t (expecting %t)\n", idx+1,
			hasAlpha, tc.ShouldHaveAlphaPixel, skip, tc.ShouldSkip)
		if tc.ShouldHaveAlphaPixel != hasAlpha || tc.ShouldSkip != skip {
			t.Fail()
		}
	}
}
