package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"r34watch/internal/dedupe"
	"r34watch/internal/mediainfo"
	"r34watch/internal/store"
)

// errSourceChanged means the on-disk source no longer matches the SHA-256 the
// caller based its work on; HWatch refuses to replace it.
var errSourceChanged = errors.New("source_changed")

const transcodeApplyReplace = "replace_source"

func isHEVCCodec(codec string) bool {
	c := strings.ToUpper(codec)
	return strings.Contains(c, "265") || strings.Contains(c, "HEVC")
}

// validateTranscodeOutput enforces the conservative "hevc-preserve" contract:
// same resolution, same framing rate, close duration, no bit-depth regression by
// default, and stable color signaling. HWatch probes both files itself.
func validateTranscodeOutput(source, output mediainfo.Info) error {
	if output.DurationSeconds <= 0 || output.Width <= 0 || output.Height <= 0 {
		return errors.New("转码产物缺少有效的视频信息")
	}
	if !isHEVCCodec(output.VideoCodec) {
		return fmt.Errorf("转码产物编码不是 HEVC: %s", output.VideoCodec)
	}
	if source.Width > 0 && output.Width != source.Width {
		return fmt.Errorf("分辨率宽度变化: %d -> %d", source.Width, output.Width)
	}
	if source.Height > 0 && output.Height != source.Height {
		return fmt.Errorf("分辨率高度变化: %d -> %d", source.Height, output.Height)
	}
	if source.FrameRate > 0 && output.FrameRate > 0 {
		tolerance := math.Max(0.5, source.FrameRate*0.01)
		if math.Abs(output.FrameRate-source.FrameRate) > tolerance {
			return fmt.Errorf("帧率变化: %.3f -> %.3f", source.FrameRate, output.FrameRate)
		}
	}
	if source.DurationSeconds > 0 {
		tolerance := math.Max(2, source.DurationSeconds*0.01)
		if math.Abs(output.DurationSeconds-source.DurationSeconds) > tolerance {
			return fmt.Errorf("时长差异过大: %.2f 秒", math.Abs(output.DurationSeconds-source.DurationSeconds))
		}
	}
	if source.BitDepth > 0 && output.BitDepth > 0 && output.BitDepth < source.BitDepth {
		return fmt.Errorf("位深被降低: %d -> %d", source.BitDepth, output.BitDepth)
	}
	if source.ColorPrimaries != "" && output.ColorPrimaries != "" && source.ColorPrimaries != output.ColorPrimaries {
		return fmt.Errorf("色彩原色变化: %s -> %s", source.ColorPrimaries, output.ColorPrimaries)
	}
	if source.ColorTransfer != "" && output.ColorTransfer != "" && source.ColorTransfer != output.ColorTransfer {
		return fmt.Errorf("色彩传递特性变化: %s -> %s", source.ColorTransfer, output.ColorTransfer)
	}
	return nil
}

// moveFile renames src over dst, falling back to a same-directory copy when the
// two are on different filesystems (EXDEV).
func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".hwatch-apply-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	in, err := os.Open(src)
	if err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	_, copyErr := io.Copy(tmp, in)
	in.Close()
	if copyErr == nil {
		copyErr = tmp.Sync()
	}
	if closeErr := tmp.Close(); copyErr == nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		os.Remove(tmpName)
		return copyErr
	}
	if err := os.Rename(tmpName, dst); err != nil {
		os.Remove(tmpName)
		return err
	}
	_ = os.Remove(src)
	return nil
}

// applyTranscodeResult validates the staged output and atomically replaces the
// source media in place. The caller guarantees outputAbs is inside the staging
// root under the media root. On any failure the original source is restored.
func (a *App) applyTranscodeResult(v store.Video, outputAbs, sourceSHA string) error {
	source := mediaPathOf(v)
	if source == "" {
		return errors.New("媒体路径为空")
	}
	if sourceSHA != "" {
		if _, currentSHA, _ := a.store.MediaFingerprint(v.ID); currentSHA != "" && currentSHA != sourceSHA {
			return errSourceChanged
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	sourceInfo, err := mediainfo.Probe(ctx, source)
	if err != nil {
		return fmt.Errorf("源文件校验失败: %w", err)
	}
	outputInfo, err := mediainfo.Probe(ctx, outputAbs)
	if err != nil {
		return fmt.Errorf("转码产物校验失败: %w", err)
	}
	if err := validateTranscodeOutput(sourceInfo, outputInfo); err != nil {
		return err
	}
	backup := source + ".hwatch-pre-transcode"
	_ = os.Remove(backup)
	if err := os.Rename(source, backup); err != nil {
		return fmt.Errorf("暂存原文件失败: %w", err)
	}
	if err := moveFile(outputAbs, source); err != nil {
		_ = os.Rename(backup, source)
		return fmt.Errorf("替换源文件失败: %w", err)
	}
	// DB metadata update uses HWatch's own ffprobe result, never the callback.
	if err := a.store.UpdateMediaInfo(v.ID, outputInfo.VideoCodec, outputInfo.AudioCodec, outputInfo.Container, outputInfo.Width, outputInfo.Height, outputInfo.FrameRate, outputInfo.BitRate, outputInfo.BitDepth); err != nil {
		_ = moveFile(source, outputAbs)
		_ = os.Rename(backup, source)
		return err
	}
	_ = a.store.SetVideoDuration(v.ID, int64(math.Round(outputInfo.DurationSeconds)))
	if size, quick, sha, hashErr := dedupe.FileHashes(source); hashErr == nil {
		_ = a.store.SaveFileHashes(v.ID, size, quick, sha)
	}
	_ = os.Remove(backup)
	return nil
}
