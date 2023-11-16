package state

import (
	"os"
	"strconv"

	"golang.org/x/sys/unix"
)

func OpenTmp() (*os.File, error) {
	fd, err := unix.Open(os.TempDir(), unix.O_TMPFILE|unix.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	return os.NewFile(uintptr(fd), "/proc/self/fd/"+strconv.Itoa(fd)), nil
}

func LinkTmp(f *os.File, name string) error {
	if err := f.Sync(); err != nil {
		return err
	}

	return unix.Linkat(unix.AT_FDCWD, f.Name(), unix.AT_FDCWD, name, unix.AT_SYMLINK_FOLLOW)
}

func Store(filename string, p []byte) error {
	f, err := OpenTmp()
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err = f.Write(p); err != nil {
		return err
	}

	return LinkTmp(f, filename)
}
