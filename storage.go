package filestorage

import (
	"errors"
	"path"
	"time"
)

var (
	ErrNoData      = errors.New("fileStorage: no data")           // file accessing methods returns this error when the given key has no data
	ErrNeverExpire = errors.New("filetray: expiredAt is not set") // IsExpired method on fileStorage returns this error when expiredAt is nil for the key
)

// A storage with cache implementation
type Storage interface {
	Get(key string, dest interface{}) error
	NoCacheGet(key string, dest interface{}) error
	IsExpired(key string) (isExpired bool, err error)
	Set(key string, value interface{}, expiredAt *time.Time) error
	Delete(key string) error
}

type fileStorage struct {
	dirpath string
	trays   map[string]*tray
}

func NewFileStorage(dirpath string) *fileStorage {
	return &fileStorage{
		dirpath: path.Dir(dirpath),
		trays:   map[string]*tray{},
	}
}

func (f *fileStorage) get(key string, dest interface{}, useCache bool) (err error) {
	if _, ok := f.trays[key]; !ok {
		return ErrNoData
	}
	return f.trays[key].get(dest, useCache)
}

func (f *fileStorage) Get(key string, dest interface{}) (err error) {
	return f.get(key, dest, true)
}

func (f *fileStorage) NoCacheGet(key string, dest interface{}) (err error) {
	return f.get(key, dest, false)
}

func (f *fileStorage) IsExpired(key string) (isExpired bool, err error) {
	if _, ok := f.trays[key]; !ok {
		return false, ErrNoData
	}
	return f.trays[key].isExpired()
}

func (f *fileStorage) Delete(key string) (err error) {
	if _, ok := f.trays[key]; !ok {
		return ErrNoData
	}
	return f.trays[key].clear()
}

func (f *fileStorage) Set(key string, value interface{}, expiredAt *time.Time) error {
	if _, ok := f.trays[key]; !ok {
		f.trays[key] = newTray(f.dirpath, key, expiredAt)
	}
	return f.trays[key].set(value)
}
