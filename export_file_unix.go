//go:build darwin || linux

package main

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

func writeExportFile(root string, data []byte) error {
	flags := unix.O_RDONLY | unix.O_DIRECTORY | unix.O_NOFOLLOW | unix.O_CLOEXEC
	rootFD, err := unix.Open(root, flags, 0)
	if err != nil {
		return fmt.Errorf("open checkout directory: %w", err)
	}
	defer unix.Close(rootFD)
	if err := unix.Mkdirat(rootFD, ".github", 0755); err != nil && err != unix.EEXIST {
		return fmt.Errorf("create .github: %w", err)
	}
	dirFD, err := unix.Openat(rootFD, ".github", flags, 0)
	if err != nil {
		return fmt.Errorf("open .github directory: %w", err)
	}
	defer unix.Close(dirFD)
	// Do not truncate until the opened file itself has been checked. Directory
	// descriptors and O_NOFOLLOW prevent path replacements from redirecting writes.
	fd, err := unix.Openat(dirFD, "settings.yml", unix.O_WRONLY|unix.O_CREAT|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0644)
	if err != nil {
		return err
	}
	file := os.NewFile(uintptr(fd), settingsPath)
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("destination must be a regular file")
	}
	if err := file.Truncate(0); err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		return err
	}
	return file.Close()
}
