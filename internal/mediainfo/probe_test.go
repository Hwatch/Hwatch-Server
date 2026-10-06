package mediainfo

import "testing"

func TestParse(t *testing.T) {
	raw := []byte(`{"streams":[{"codec_type":"video","codec_name":"hevc","width":3840,"height":2160,"avg_frame_rate":"60000/1001","bits_per_raw_sample":"10"},{"codec_type":"audio","codec_name":"aac"}],"format":{"format_name":"mov,mp4,m4a","bit_rate":"12000000","duration":"123.5"}}`)
	info, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if info.VideoCodec != "H.265/HEVC" || info.AudioCodec != "AAC" || info.Container != "MP4" || info.Width != 3840 || info.BitDepth != 10 || info.BitRate != 12000000 || info.FrameRate < 59.9 || info.DurationSeconds != 123.5 {
		t.Fatalf("unexpected info: %+v", info)
	}
}
