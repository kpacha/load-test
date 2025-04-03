package db

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
)

func NewFS(path string, s *session.Session, bucket string) (DB, error) {
	fs := fileSystem(path)
	s3, err := NewS3(s, &fs, bucket)
	if err != nil {
		return &fs, err
	}
	f := persistedFS{
		fs: &fs,
		s3: s3,
	}
	return f, nil
}

type persistedFS struct {
	fs *fileSystem
	s3 S3
}

func (f persistedFS) Get(key string) (io.Reader, error) {
	return f.fs.Get(key)
}

func (f persistedFS) Keys() ([]string, error) {
	return f.fs.Keys()
}

func (f persistedFS) Set(key string, r io.Reader) (int, error) {
	n, err := f.fs.Set(key, r)
	if err != nil {
		return n, err
	}
	go func() {
		if err := f.s3.Upload(key); err != nil {
			log.Printf("uploading '%s' to S3: %s", key, err)
		}
	}()

	return n, nil
}

const fsBDExtension = ".json"

type fileSystem string

func (f *fileSystem) Get(key string) (io.Reader, error) {
	data, err := os.ReadFile(string(*f) + "/" + key + fsBDExtension)
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
	data, err := io.ReadAll(r)
	if err != nil {
		return 0, err
	}
	err = ioutil.WriteFile(f.GetPath(key), data, 0644)
	if err != nil {
		return 0, err
	}
	return len(data), nil
}

func (f *fileSystem) GetPath(key string) string {
	return string(*f) + "/" + key + fsBDExtension
}
