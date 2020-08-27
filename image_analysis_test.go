//package main
//
//import (
//	"image"
//	"os"
//	"testing"
//)
//import _ "image/png" //register JPEG decoder
//
//func CheckForAlphaPixel(file string) error {
//	f, err := os.Open(file)
//	if err != nil {
//		return err
//	}
//	defer f.Close()
//
//	img, fmtName, err := image.Decode(f)
//	if err != nil {
//		return err
//	}
//}
//
//func CheckForAlphaPixelTest(t *testing.T) {
//	if CheckForAlphaPixel("./test/arrow.png")
//
//}
