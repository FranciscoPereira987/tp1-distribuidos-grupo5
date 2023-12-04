package state

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

const StateFileName = "state.json"

var (
	ErrNotFound = errors.New("key not found")
	ErrNotMap   = errors.New("object is not a map")
	ErrNaN      = errors.New("object is not a number")
)

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

func (sw *StateManager) NewMap(keys ...string) {
	m := sw.State
	for _, key := range keys[:len(keys)-1] {
		m = m[key].(map[string]any)
	}
	m[keys[len(keys)-1]] = make(map[string]any)
}

func (sw *StateManager) Add(value any, keys ...string) {
	m := sw.State
	for _, key := range keys[:len(keys)-1] {
		m = m[key].(map[string]any)
	}
	m[keys[len(keys)-1]] = value
}

func (sw *StateManager) Remove(keys ...string) {
	m := sw.State
	for _, key := range keys[:len(keys)-1] {
		m = m[key].(map[string]any)
	}
	delete(m, keys[len(keys)-1])
}

func getJsonMap(m map[string]any, keys ...string) (map[string]any, error) {
	for i, key := range keys {
		if v, ok := m[key]; !ok {
			key = strings.Join(keys[:i+1], ".")
			return nil, fmt.Errorf("%w: state[%s]", ErrNotFound, key)
		} else if m, ok = v.(map[string]any); !ok {
			key = strings.Join(keys[:i+1], ".")
			return nil, fmt.Errorf("%w: state[%s]=%v", ErrNotMap, key, v)
		}
	}
	return m, nil
}

func (sw *StateManager) getNumber(keys ...string) (json.Number, error) {
	m, err := getJsonMap(sw.State, keys[:len(keys)-1]...)
	if err != nil {
		return "", err
	}

	v, ok := m[keys[len(keys)-1]]
	if !ok {
		key := strings.Join(keys, ".")
		return "", fmt.Errorf("%w: state[%s]", ErrNotFound, key)
	}
	num, ok := v.(json.Number)
	if !ok {
		key := strings.Join(keys, ".")
		return "", fmt.Errorf("%w: state[%s]=%v", ErrNaN, key, v)
	}
	return num, nil
}

func (sw *StateManager) GetInt64(keys ...string) (int64, error) {
	num, err := sw.getNumber(keys...)
	if err != nil {
		return 0, err
	}
	return num.Int64()
}

func (sw *StateManager) GetInt(keys ...string) (int, error) {
	n, err := sw.GetInt64(keys...)
	return int(n), err
}

func (sw *StateManager) GetFloat(keys ...string) (float64, error) {
	num, err := sw.getNumber(keys...)
	if err != nil {
		return 0, err
	}
	return num.Float64()
}

func (sw *StateManager) GetMapStringInt64(keys ...string) (map[string]int64, error) {
	m, err := getJsonMap(sw.State, keys...)
	if err != nil {
		return nil, err
	}

	ret := make(map[string]int64)
	for key, value := range m {
		v, ok := value.(json.Number)
		if !ok {
			key := strings.Join(keys, ".")
			return nil, fmt.Errorf("%w: state[%s]=%v", ErrNaN, key, v)
		}
		ret[key], err = v.Int64()
		if err != nil {
			return nil, err
		}
	}
	return ret, nil
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
		dec.UseNumber()
		if err = dec.Decode(&sw.State); err == nil {
			log.Infof("recovered state: %v", sw.State)
		}
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
	dir, file := filepath.Split(filename)
	tmpDir := filepath.Join(dir, "tmp")
	if err := os.Mkdir(tmpDir, 0755); err != nil && !os.IsExist(err) {
		return nil, err
	}
	return os.Create(filepath.Join(tmpDir, file))
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

// Atomically removes the working directory and all files in it
// so as to not leave the system in an inconsistent state.
func RemoveWorkdir(workdir string) error {
	dir, file := filepath.Split(workdir)
	tmpdir := filepath.Join(dir, "tmp")
	tmpWorkdir := filepath.Join(tmpdir, file)
	if err := os.Mkdir(tmpdir, 0755); err != nil && !os.IsExist(err) {
		log.Warnf("os.Mkdir(%q) failed, removing non-atomically", tmpdir)
		tmpWorkdir = workdir
	} else if err := os.Rename(workdir, tmpWorkdir); err != nil {
		log.Warnf("os.Rename(%q, %q) failed, removing non-atomically", workdir, tmpWorkdir)
		tmpWorkdir = workdir
	}
	return os.RemoveAll(tmpWorkdir)
}
