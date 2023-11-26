package state

import (
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

func (sw *StateManager) GetString(key string) (stringVal string) {
	value, ok := sw.GetFromState(key)
	if asString, casted := value.(string); ok && casted {
		stringVal = asString
	}
	return
}

func (sw *StateManager) GetInt(key string) (intVal int) {
	value, ok := sw.GetFromState(key)
	if asInt, casted := value.(int); ok && casted {
		intVal = asInt
	}
	return
}

func (sw *StateManager) DumpState() error {
	buf, err := json.Marshal(sw.state)

	if err == nil {
		err = WriteFile(sw.Filename, buf)
	}

	return err
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

/*
Filters files that have .state in their names
and are not directories
*/
func filterStateFiles(dir string) (files []string) {
	unfiltered, err := os.ReadDir(dir)
	if err == nil {
		for _, entry := range unfiltered {
			if strings.Contains(entry.Name(), ".state") {
				files = append(files, filepath.Join(dir, entry.Name()))
			}
		}
	}
	return
}

func RecoverStateFiles(workdir string) (states []*StateManager) {
	files := filterStateFiles(workdir)
	for _, stateFile := range files {
		state := NewStateManager(stateFile)
		if err := state.RecoverState(); err == nil {
			states = append(states, state)
		}
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
