package state

import (
	"os"
	"path/filepath"
	"strconv"

	"golang.org/x/sys/unix"
)

func Store(file string, p []byte) error {
	fd, err := unix.Open(filepath.Dir(file), unix.O_TMPFILE|unix.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	f := os.NewFile(uintptr(fd), "/proc/self/fd/"+strconv.Itoa(fd))
	defer f.Close()

	if _, err = f.Write(p); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}

	return unix.Linkat(unix.AT_FDCWD, f.Name(), unix.AT_FDCWD, file, unix.AT_SYMLINK_FOLLOW)
}
