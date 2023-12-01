package state

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
)

const StateFileName = "state.json"

/*
It should only be stored in state, objects that implement either:

1. json.Marshaller
2. encoding.TextMarshaller
*/
type StateManager struct {
	Filename string
	State    map[string]any
	tmp      string
}

func NewStateManager(workdir string) *StateManager {
	return &StateManager{
		Filename: filepath.Join(workdir, StateFileName),
		State:    make(map[string]any),
	}
}

func (sw *StateManager) AddToState(key string, value any) {
	sw.State[key] = value
}

func (sw *StateManager) Prepare() error {
	buf, err := json.Marshal(sw.State)
	if err != nil {
		return err
	}
	name, err := WriteTmp(sw.Filename, buf)
	if err == nil {
		sw.tmp = name
	}
	return err
}

func (sw StateManager) Commit() error {
	return LinkTmp(sw.tmp, sw.Filename)
}

func (sw *StateManager) DumpState() error {
	err := sw.Prepare()
	if err == nil {
		err = sw.Commit()
	}

	return err
}

func (sw *StateManager) RecoverState() (err error) {
	var file *os.File
	file, err = os.Open(sw.Filename)

	if err == nil {
		defer file.Close()
		dec := json.NewDecoder(file)
		err = dec.Decode(&sw.State)
	}

	return
}

type recovered struct {
	Id      string
	Workdir string
	State   *StateManager
}

// State files are stored in a subdirectory of the worker's working directory.
// This subdirectory is named using the associated client's id.
// Example (with workdir := "/clients"):
//
//	$ tree /clients
//	/clients
//	├── 0841bcc1
//	│   └── state.json
//	├── 94519ae2
//	│   └── state.json
//	└── a9e48a18
//	    └── state.json
func RecoverStateFiles(workdir string) []recovered {
	subdirs, _ := os.ReadDir(workdir)
	rec := make([]recovered, 0, len(subdirs))

	for _, dir := range subdirs {
		if !dir.IsDir() {
			continue
		}
		id, err := hex.DecodeString(dir.Name())
		if err != nil {
			continue
		}
		dirName := filepath.Join(workdir, dir.Name())
		if _, err := os.Stat(filepath.Join(dirName, StateFileName)); os.IsNotExist(err) {
			continue
		}
		state := NewStateManager(dirName)
		if err := state.RecoverState(); err == nil {
			rec = append(rec, recovered{string(id), dirName, state})
		}
	}
	return rec
}

// given a /path/to/file, create and return a temporary file
// in /path/to/tmp/file to be renamed later using LinkTmp()
func CreateTmp(filename string) (*os.File, error) {
	tmpDir := filepath.Join(filepath.Dir(filename), "tmp")
	if err := os.Mkdir(tmpDir, 0755); err != nil && !os.IsExist(err) {
		return nil, err
	}
	return os.Create(filepath.Join(tmpDir, filepath.Base(filename)))
}

// given a /path/to/file (`filename'), renames the temporary
// file /path/to/tmp/file to /path/to/file. removing the tmp/ component from
// the path
func LinkTmp(oldpath, newpath string) error {
	err := os.Rename(oldpath, newpath)
	if err != nil {
		os.Remove(oldpath)
	}
	return err
}

// writes a temporary file to link it later with LinkTmp()
func WriteTmp(filename string, p []byte) (string, error) {
	f, err := CreateTmp(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err = f.Write(p); err != nil {
		return "", err
	}
	return f.Name(), f.Sync()
}

// convenience function to write the temporary file and link it at once
func WriteFile(filename string, p []byte) error {
	tmp, err := WriteTmp(filename, p)
	if err != nil {
		return err
	}

	return LinkTmp(tmp, filename)
}
