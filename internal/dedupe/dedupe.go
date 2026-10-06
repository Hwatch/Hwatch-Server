package dedupe

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math/bits"
	"net/url"
	"os"
	"strconv"
	"strings"
)

func NormalizeURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return strings.TrimSpace(raw)
	}
	u.Fragment = ""
	u.Host = strings.ToLower(u.Host)
	u.Path = strings.TrimRight(u.Path, "/")
	// Most query strings seen here are expiring signatures or tracking data and
	// must not become identity. Hanime1 is an exception: /watch uses the v query
	// parameter as the stable video ID, so dropping it collapses every video into
	// the same URL.
	if strings.EqualFold(u.Hostname(), "hanime1.me") && u.Path == "/watch" {
		videoID := u.Query().Get("v")
		u.RawQuery = ""
		if videoID != "" {
			q := url.Values{}
			q.Set("v", videoID)
			u.RawQuery = q.Encode()
		}
	} else {
		u.RawQuery = ""
	}
	return u.String()
}
func ImageHash(r io.Reader) (string, error) {
	raw, err := io.ReadAll(io.LimitReader(r, (20<<20)+1))
	if err != nil {
		return "", err
	}
	if len(raw) > 20<<20 {
		return "", fmt.Errorf("image too large")
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	if cfg.Width <= 1 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > 50_000_000 {
		return "", fmt.Errorf("invalid image dimensions")
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	b := img.Bounds()
	if b.Dx() < 2 || b.Dy() < 1 {
		return "", fmt.Errorf("image too small")
	}
	var hash uint64
	bit := uint(0)
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			a := luma(img.At(b.Min.X+x*b.Dx()/9, b.Min.Y+y*b.Dy()/8))
			c := luma(img.At(b.Min.X+(x+1)*b.Dx()/9, b.Min.Y+y*b.Dy()/8))
			if a > c {
				hash |= 1 << bit
			}
			bit++
		}
	}
	return fmt.Sprintf("%016x", hash), nil
}
func luma(c interface {
	RGBA() (uint32, uint32, uint32, uint32)
}) uint32 {
	r, g, b, _ := c.RGBA()
	return (299*r + 587*g + 114*b) / 1000
}
func Similarity(a, b string) float64 {
	x, e1 := strconv.ParseUint(a, 16, 64)
	y, e2 := strconv.ParseUint(b, 16, 64)
	if e1 != nil || e2 != nil {
		return 0
	}
	return 1 - float64(bits.OnesCount64(x^y))/64
}
func FileHashes(path string) (size int64, quick, full string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, "", "", err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return 0, "", "", err
	}
	size = info.Size()
	q := sha256.New()
	_, err = io.CopyN(q, f, min(size, 1<<20))
	if err != nil && err != io.EOF {
		return 0, "", "", err
	}
	if size > 1<<20 {
		_, _ = f.Seek(max(0, size-(1<<20)), io.SeekStart)
		_, err = io.Copy(q, f)
		if err != nil {
			return 0, "", "", err
		}
	}
	quick = hex.EncodeToString(q.Sum(nil))
	_, _ = f.Seek(0, io.SeekStart)
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return 0, "", "", err
	}
	return size, quick, hex.EncodeToString(h.Sum(nil)), nil
}
