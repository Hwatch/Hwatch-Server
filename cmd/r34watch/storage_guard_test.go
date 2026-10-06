package main

import "testing"

func TestMountedAtUsesLongestMount(t *testing.T) {
	raw := []byte("1 0 0:1 / / rw - ext4 /dev/root rw\n2 1 0:2 / /mnt/media rw - nfs 192.0.2.1:/media rw\n")
	mount, ok := mountedAt("/mnt/media/library/video", raw)
	if !ok || mount != "/mnt/media" {
		t.Fatalf("mount = %q, %v", mount, ok)
	}
}

func TestRequiresExternalMount(t *testing.T) {
	if !requiresExternalMount("/mnt/media/library") {
		t.Fatal("/mnt path must require an external mount")
	}
	if requiresExternalMount("/var/lib/hwatch") {
		t.Fatal("local application storage must remain supported")
	}
}
