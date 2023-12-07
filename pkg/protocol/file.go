package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

const bom = "\ufeff"

type BomSkipper struct {
	*os.File
	Size   int64
	HasBom bool
}

func NewBomSkipper(f *os.File) (BomSkipper, error) {
	bs := BomSkipper{File: f}
	stat, err := f.Stat()
	if err != nil {
		return bs, err
	}
	bs.Size = stat.Size()

	// skip byte order mark if present
	var buf [len(bom)]byte
	io.ReadFull(f, buf[:])
	bs.HasBom = string(buf[:]) == bom
	if bs.HasBom {
		bs.Size -= int64(len(bom))
	} else {
		_, err = f.Seek(0, 0)
	}

	return bs, err
}

func (bs *BomSkipper) Seek(offset int64, whence int) (ret int64, err error) {
	if bs.HasBom && whence == 0 {
		offset += int64(len(bom))
	}
	ret, err = bs.File.Seek(offset, whence)
	if err != nil {
		return ret, err
	}
	if bs.HasBom {
		ret -= int64(len(bom))
	}
	if ret < 0 {
		return 0, fmt.Errorf("seek %s: %w", bs.File.Name(), os.ErrInvalid)
	}
	return ret, err
}

func WriteFile(w io.Writer, f *os.File, offset int64) error {
	bs, err := NewBomSkipper(f)
	if err != nil {
		return fmt.Errorf("WriteFile - bom skipper: %w", err)
	}

	switch {
	case offset < 0:
		if err := binary.Write(w, binary.LittleEndian, bs.Size); err != nil {
			return fmt.Errorf("WriteFile - binary.Write: %w", err)
		}
	case offset > 0:
		if _, err := bs.Seek(offset, 0); err != nil {
			return fmt.Errorf("WriteFile - Seek: %w", err)
		}
	}

	if _, err = io.Copy(w, bs); err != nil {
		return fmt.Errorf("WriteFile - Copy: %w", err)
	}
	return nil
}

type ExactReader struct {
	R io.Reader
	N int64
}

func NewFileReader(r io.Reader) (ExactReader, error) {
	xr := ExactReader{R: r}
	err := binary.Read(r, binary.LittleEndian, &xr.N)
	return xr, err
}

func (r *ExactReader) Read(p []byte) (n int, err error) {
	if r.N == 0 {
		return 0, io.EOF
	}

	p = p[:min(len(p), int(r.N))]
	n, err = r.R.Read(p)
	r.N -= int64(n)

	if err == io.EOF && r.N > 0 {
		err = fmt.Errorf("%w: missing %d bytes", io.ErrUnexpectedEOF, r.N)
	}
	return n, err
}
