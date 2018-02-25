package db

import (
	"errors"
	"io"
)

var (
	ErrNotFound     = errors.New("key not found")
	ErrUnableToList = errors.New("unable to list keys")
	ErrNotSet       = errors.New("unable to set the key-content pair")
)

type DB interface {
	Get(key string) (io.Reader, error)
	Keys() ([]string, error)
	Set(key string, r io.Reader) (int, error)
}
