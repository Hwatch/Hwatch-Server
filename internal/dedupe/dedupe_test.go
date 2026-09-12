package dedupe

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestNormalizeURL(t *testing.T) {
	if got := NormalizeURL("HTTPS://Example.COM/video/1/?token=secret#x"); got != "https://example.com/video/1" {
		t.Fatal(got)
	}
}
func TestImageSimilarity(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 90, 80))
	for y := 0; y < 80; y++ {
		for x := 0; x < 90; x++ {
			img.SetGray(x, y, color.Gray{Y: uint8(x * 2)})
		}
	}
	var a, b bytes.Buffer
	_ = png.Encode(&a, img)
	_ = png.Encode(&b, img)
	ha, e := ImageHash(&a)
	if e != nil {
		t.Fatal(e)
	}
	hb, e := ImageHash(&b)
	if e != nil {
		t.Fatal(e)
	}
	if Similarity(ha, hb) != 1 {
		t.Fatal("identical images differ")
	}
}
