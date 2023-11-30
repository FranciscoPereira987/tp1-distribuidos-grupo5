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

func (sw *StateManager) Get(key string) any {
	return sw.State[key]
}

func (sw *StateManager) GetString(key string) string {
	s, _ := sw.State[key].(string)
	return s
}

func (sw *StateManager) GetInt(key string) int {
	i, _ := sw.State[key].(int)
	return i
}

func (sw *StateManager) DumpState() error {
	buf, err := json.Marshal(sw.State)

	if err == nil {
		err = WriteFile(sw.Filename, buf)
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
func LinkTmp(filename string) error {
	tmp_name := filepath.Join(filepath.Dir(filename), "tmp", filepath.Base(filename))

	err := os.Rename(tmp_name, filename)
	if err != nil {
		os.Remove(tmp_name)
	}
	return err
}

// writes a temporary file to link it later with LinkTmp()
func WriteTmp(filename string, p []byte) error {
	f, err := CreateTmp(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err = f.Write(p); err != nil {
		return err
	}
	return f.Sync()
}

// convenience function to write the temporary file and link it at once
func WriteFile(filename string, p []byte) error {
	if err := WriteTmp(filename, p); err != nil {
		return err
	}

	return LinkTmp(filename)
}
