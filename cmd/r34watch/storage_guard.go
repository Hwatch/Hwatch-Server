package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func decodeMountPath(value string) string {
	replacer := strings.NewReplacer(`\040`, " ", `\011`, "\t", `\012`, "\n", `\134`, `\`)
	return replacer.Replace(value)
}

func mountedAt(path string, mountInfo []byte) (string, bool) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}
	best := ""
	for _, line := range strings.Split(string(mountInfo), "\n") {
		parts := strings.Fields(line)
		if len(parts) < 5 {
			continue
		}
		mountPoint := filepath.Clean(decodeMountPath(parts[4]))
		if (abs == mountPoint || pathWithin(mountPoint, abs)) && len(mountPoint) > len(best) {
			best = mountPoint
		}
	}
	return best, best != ""
}

func requiresExternalMount(path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	for _, root := range []string{"/mnt", "/media", "/run/media"} {
		if abs == root || pathWithin(root, abs) {
			return true
		}
	}
	return false
}

func pathHasDistinctMount(path string) bool {
	mountInfo, err := os.ReadFile("/proc/self/mountinfo")
	if err != nil {
		return false
	}
	mountPoint, ok := mountedAt(path, mountInfo)
	return ok && mountPoint != "/"
}

func storagePathAvailable(path string, mountRequired bool) bool {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return false
	}
	if !mountRequired && !requiresExternalMount(path) {
		return true
	}
	return pathHasDistinctMount(path)
}

func ensureStorageDirectory(path string, mode os.FileMode, mountRequired bool) error {
	path = filepath.Clean(path)
	ancestor := path
	for {
		if info, err := os.Stat(ancestor); err == nil && info.IsDir() {
			break
		}
		parent := filepath.Dir(ancestor)
		if parent == ancestor {
			return errors.New("找不到可用的存储目录父级")
		}
		ancestor = parent
	}
	if !storagePathAvailable(ancestor, mountRequired) {
		return errors.New("存储目录未挂载，拒绝在本机文件系统创建目录")
	}
	return os.MkdirAll(path, mode)
}
