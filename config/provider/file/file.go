package file

import (
	"encoding/json"
	"os"

	"github.com/hysios/mx/config"
)

type FileProvider struct {
	// contains filtered or unexported fields
	vals config.Map
}

func NewFileProvider(path string) (*FileProvider, error) {
	// open file
	// read file
	f, err := os.OpenFile(path, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	// json unmarshal to map[string]interface{}
	var vals = make(map[string]interface{})

	if err = json.NewDecoder(f).Decode(vals); err != nil {
		return nil, err
	}

	return &FileProvider{vals: config.NewMap(vals)}, nil
}

// MustFileProvider returns a new FileProvider or panic.
func MustFileProvider(path string) *FileProvider {
	f, err := NewFileProvider(path)
	if err != nil {
		panic(err)
	}
	return f
}

func (f *FileProvider) LookupPath(selector string) (val *config.Value, ok bool) {
	val = f.vals.Get(selector)
	ok = !val.IsNil()

	return
}

func (f *FileProvider) Set(selector string, val interface{}) (interface{}, error) {
	old := f.vals.Get(selector)
	f.vals.Set(selector, val)
	return old.Data(), nil
}

func (f *FileProvider) Update(vals map[string]interface{}) config.Map {
	return f.vals.MergeHere(vals)
}
