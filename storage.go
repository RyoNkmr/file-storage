package filestorage

import (
	"errors"
	"path"
	"time"
)

var (
	ErrNoData      = errors.New("FileStorage: no data")               // file accessing methods returns this error when the given key has no data
	ErrExpired     = errors.New("filetray: data has already expired") // file accessing methods returns this error when the data is expired
	ErrLiveForever = errors.New("filetray: expiredAt is not set")     // IsExpired method on FileStorage returns this error when expiredAt is nil for the key
)

// A storage with cache implementation
type Storage interface {
	Get(key string, dest interface{}) error
	NoCacheGet(key string, dest interface{}) error
	IsExpired(key string) (isExpired bool, err error)
	Set(key string, value interface{}, expiredAt *time.Time) error
	Delete(key string) error
}

type FileStorage struct {
	dirpath string
	trays   map[string]*tray
}

// FileStorage create file each key at "dirpath/key"
func NewFileStorage(dirpath string) *FileStorage {
	return &FileStorage{
		dirpath: path.Dir(dirpath),
		trays:   map[string]*tray{},
	}
}

func (f *FileStorage) get(key string, dest interface{}, useCache bool) (err error) {
	if _, ok := f.trays[key]; !ok {
		return ErrNoData
	}
	return f.trays[key].get(dest, useCache)
}

// Get scans the stored value from FileStorage into dest. if the data is not stored, then returns ErrNoData. if the has already expired, this returns ErrExpired.
func (f *FileStorage) Get(key string, dest interface{}) (err error) {
	return f.get(key, dest, true)
}

// Get scans the stored value from FileStorage into dest without cache. if the data is not stored, then returns ErrNoData. if the has already expired, this returns ErrExpired.
func (f *FileStorage) NoCacheGet(key string, dest interface{}) (err error) {
	return f.get(key, dest, false)
}

// IsExpired checks whether the stored data is expired. if the data does not have expiredAt, then this returns ErrLiveForever as the second return value.
func (f *FileStorage) IsExpired(key string) (isExpired bool, err error) {
	if _, ok := f.trays[key]; !ok {
		return false, ErrNoData
	}
	return f.trays[key].isExpired()
}

// Delete removes the file and the data from the cache
func (f *FileStorage) Delete(key string) (err error) {
	if _, ok := f.trays[key]; !ok {
		return ErrNoData
	}
	return f.trays[key].clear()
}

// Set stores value into FileStorage.
// FileStorage creates file each key at "dirpath/key"
func (f *FileStorage) Set(key string, value interface{}, expiredAt *time.Time) error {
	if _, ok := f.trays[key]; !ok {
		f.trays[key] = newTray(f.dirpath, key, expiredAt)
	}
	return f.trays[key].set(value)
}
