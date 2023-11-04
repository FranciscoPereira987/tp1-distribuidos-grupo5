package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

type ExactReader struct {
	R io.Reader
	N int64
}

func WriteFile(w io.Writer, f *os.File) error {
	stat, err := f.Stat()
	if err != nil {
		return err
	}
	size := stat.Size()

	// remove byte order mark if present
	var bom [len("\ufeff")]byte
	io.ReadFull(f, bom[:])
	if string(bom[:]) == "\ufeff" {
		size -= int64(len("\ufeff"))
	} else {
		if _, err = f.Seek(0, 0); err != nil {
			return err
		}
	}

	if err := binary.Write(w, binary.LittleEndian, size); err != nil {
		return err
	}
	_, err = io.Copy(w, f)
	return err
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
