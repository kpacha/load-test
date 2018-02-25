package db

import (
	"bytes"
	"io"
	"io/ioutil"
	"strings"
)

func NewFS(path string) DB {
	fs := fileSystem(path)
	return &fs
}

const fsBDExtension = ".json"

type fileSystem string

func (f *fileSystem) Get(key string) (io.Reader, error) {
	data, err := ioutil.ReadFile(string(*f) + "/" + key + fsBDExtension)
	if err != nil {
		return nil, ErrNotFound
	}
	return bytes.NewBuffer(data), nil
}

func (f *fileSystem) Keys() ([]string, error) {
	fs, err := ioutil.ReadDir(string(*f))
	if err != nil {
		return []string{}, ErrUnableToList
	}
	res := []string{}
	for _, file := range fs {
		if file.IsDir() || !strings.Contains(file.Name(), fsBDExtension) {
			continue
		}
		res = append(res, strings.TrimRight(file.Name(), fsBDExtension))
	}
	return res, nil
}

func (f *fileSystem) Set(key string, r io.Reader) (int, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return 0, err
	}
	err = ioutil.WriteFile(string(*f)+"/"+key+fsBDExtension, data, 0644)
	if err != nil {
		return 0, err
	}
	return len(data), nil
}
