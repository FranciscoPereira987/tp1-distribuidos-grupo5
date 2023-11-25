package state

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

/*
	It should only be stored in state, objects that implement either:

	1. json.Marshaller
	2. encoding.TextMarshaller
*/
type StateManager struct {
	Filename string
	state    map[string]any
}

func NewStateManager(filename string) *StateManager {
	return &StateManager{
		Filename: filename,
		state:    make(map[string]any),
	}
}

func (sw *StateManager) AddToState(key string, value any) {
	sw.state[key] = value
}

func (sw *StateManager) GetFromState(key string) (value any, ok bool) {
	value, ok = sw.state[key]
	return
}	

func (sw *StateManager) DumpState() (err error) {
	var marshalled []byte
	marshalled, err = json.Marshal(sw.state)

	if err == nil {
		buffer := bytes.NewBuffer(marshalled)
		err = WriteFile(sw.Filename, buffer.Bytes())
	}
	
	return
}

func (sw *StateManager) RecoverState() (err error) {
	var file *os.File
	file, err = os.Open(sw.Filename)

	if err == nil {
		dec := json.NewDecoder(file)
		err = dec.Decode(&sw.state)
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
