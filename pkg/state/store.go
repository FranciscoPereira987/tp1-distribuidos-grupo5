package state

import (
	"os"
	"path/filepath"
)

func LinkTmp(f *os.File, name string) (err error) {
	defer func() {
		if err != nil {
			os.Remove(f.Name())
		}
	}()

	if err := f.Sync(); err != nil {
		return err
	}

	return os.Rename(f.Name(), name)
}

func WriteFile(filename string, p []byte) error {
	f, err := os.CreateTemp("", filepath.Base(filename))
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err = f.Write(p); err != nil {
		return err
	}

	return LinkTmp(f, filename)
}
