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
	if got := NormalizeURL("https://hanime1.me/watch?list=890459&v=408216&sort=latest#player"); got != "https://hanime1.me/watch?v=408216" {
		t.Fatal(got)
	}
	if a, b := NormalizeURL("https://hanime1.me/watch?v=408216"), NormalizeURL("https://hanime1.me/watch?v=408207"); a == b {
		t.Fatalf("distinct Hanime1 videos collapsed to %q", a)
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
