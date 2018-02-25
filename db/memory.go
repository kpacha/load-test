package db

import (
	"bytes"
	"io"
	"sync"
)

func NewInMemory() DB {
	return &memory{&sync.Map{}}
}

type memory struct {
	m *sync.Map
}

func (db *memory) Get(key string) (io.Reader, error) {
	v, ok := db.m.Load(key)
	if !ok {
		return nil, ErrNotFound
	}
	data, ok := v.([]byte)
	if !ok {
		return nil, ErrNotFound
	}
	return bytes.NewBuffer(data), nil
}

func (db *memory) Keys() ([]string, error) {
	res := []string{}
	db.m.Range(func(key interface{}, _ interface{}) bool {
		res = append(res, key.(string))
		return true
	})
	return res, nil
}

func (db *memory) Set(key string, r io.Reader) (int, error) {
	buf := bytes.Buffer{}
	if _, err := buf.ReadFrom(r); err != nil {
		return 0, err
	}
	db.m.Store(key, buf.Bytes())
	return buf.Len(), nil
}
