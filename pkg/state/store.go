package state

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
)

type StateManager struct {
	filename string
	state    map[string][]byte
}

func NewStateManager(filename string) *StateManager {
	//TODO: Recover state from file
	return &StateManager{
		filename: filename,
		state:    make(map[string][]byte),
	}
}

func (sw *StateManager) AddToState(key string, value []byte) {
	sw.state[key] = value
}

func (sw *StateManager) DumpState() (err error) {
	buffer := bytes.NewBuffer(nil)
	for key, value := range sw.state {
		err = encodeField(key, value, buffer)
		if err != nil {
			return
		}
	}
	err = WriteFile(sw.filename, buffer.Bytes())
	return
}

func (sw *StateManager) RecoverState() (err error) {
	var file *os.File
	file, err = os.Open(sw.filename)

	if err == nil {
		reader := bufio.NewReader(file)
		var parseErr error
		var key string
		var value []byte
		for parseErr == nil {
			key, value, parseErr = decodeField(reader)
			if parseErr == nil {
				sw.state[key] = value
			}
		}
		if !errors.Is(parseErr, io.EOF) {
			err = parseErr
		}
	}

	return
}

func encodeField(key string, field []byte, buffer *bytes.Buffer) (err error) {
	err = typing.WriteString(buffer, key)
	if err == nil {
		err = typing.WriteString(buffer, string(field))
	}

	return
}

func decodeField(reader *bufio.Reader) (key string, value []byte, err error) {
	key, err = typing.ReadString(reader)

	if err == nil {
		stringValue, errValue := typing.ReadString(reader)
		value = []byte(stringValue)
		err = errValue
	}

	return
}

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
	f, err := os.CreateTemp(filepath.Dir(filename), "tmp.")
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err = f.Write(p); err != nil {
		return err
	}

	return LinkTmp(f, filename)
}

func IsTmp(filename string) bool {
	return strings.HasPrefix(filename, "tmp.")
}
