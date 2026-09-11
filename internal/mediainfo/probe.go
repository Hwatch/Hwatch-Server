package mediainfo

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type Info struct {
	VideoCodec, AudioCodec, Container string
	Width, Height, BitDepth           int
	FrameRate                         float64
	BitRate                           int64
}

type probeResult struct {
	Streams []struct {
		CodecType        string `json:"codec_type"`
		CodecName        string `json:"codec_name"`
		AvgFrameRate     string `json:"avg_frame_rate"`
		BitRate          string `json:"bit_rate"`
		Width            int    `json:"width"`
		Height           int    `json:"height"`
		BitsPerRawSample string `json:"bits_per_raw_sample"`
		PixelFormat      string `json:"pix_fmt"`
	} `json:"streams"`
	Format struct {
		FormatName string `json:"format_name"`
		BitRate    string `json:"bit_rate"`
	} `json:"format"`
}

func Probe(ctx context.Context, path string) (Info, error) {
	out, err := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-show_entries", "stream=codec_type,codec_name,width,height,avg_frame_rate,bit_rate,bits_per_raw_sample,pix_fmt:format=format_name,bit_rate", "-of", "json", path).Output()
	if err != nil {
		return Info{}, fmt.Errorf("ffprobe: %w", err)
	}
	return Parse(out)
}

func Parse(raw []byte) (Info, error) {
	var result probeResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return Info{}, err
	}
	info := Info{Container: first(result.Format.FormatName), BitRate: integer(result.Format.BitRate)}
	for _, stream := range result.Streams {
		switch stream.CodecType {
		case "video":
			if info.VideoCodec == "" {
				info.VideoCodec, info.Width, info.Height = normalizeVideo(stream.CodecName), stream.Width, stream.Height
				info.FrameRate, info.BitDepth = fraction(stream.AvgFrameRate), int(integer(stream.BitsPerRawSample))
				if info.BitDepth == 0 {
					if strings.Contains(stream.PixelFormat, "12") {
						info.BitDepth = 12
					} else if strings.Contains(stream.PixelFormat, "10") {
						info.BitDepth = 10
					} else if stream.PixelFormat != "" {
						info.BitDepth = 8
					}
				}
				if info.BitRate == 0 {
					info.BitRate = integer(stream.BitRate)
				}
			}
		case "audio":
			if info.AudioCodec == "" {
				info.AudioCodec = strings.ToUpper(stream.CodecName)
			}
		}
	}
	if info.VideoCodec == "" {
		return Info{}, fmt.Errorf("没有找到视频流")
	}
	return info, nil
}

func normalizeVideo(codec string) string {
	switch strings.ToLower(codec) {
	case "h264":
		return "H.264"
	case "hevc", "h265":
		return "H.265/HEVC"
	case "av1":
		return "AV1"
	case "vp9":
		return "VP9"
	default:
		return strings.ToUpper(codec)
	}
}
func first(value string) string {
	if strings.Contains(strings.ToLower(value), "mp4") {
		return "MP4"
	}
	if i := strings.IndexByte(value, ','); i >= 0 {
		value = value[:i]
	}
	return strings.ToUpper(value)
}
func integer(value string) int64 { n, _ := strconv.ParseInt(value, 10, 64); return n }
func fraction(value string) float64 {
	parts := strings.Split(value, "/")
	if len(parts) != 2 {
		n, _ := strconv.ParseFloat(value, 64)
		return n
	}
	a, _ := strconv.ParseFloat(parts[0], 64)
	b, _ := strconv.ParseFloat(parts[1], 64)
	if b == 0 {
		return 0
	}
	return a / b
}
