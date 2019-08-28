package filestorage

import (
	"errors"
	"os"
	"path"
	"time"
)

var (
	ErrInvalidDir    = errors.New("FileStorage: invalid dirpath given")
	ErrNoData        = errors.New("FileStorage: no data")              // file accessing methods returns this error when the given key has no data
	ErrExpired       = errors.New("FileStorage: expired")              // file accessing methods returns this error when the data is expired
	ErrLiveForever   = errors.New("FileStorage: expiredAt is not set") // IsExpired method on FileStorage returns this error when expiredAt is nil for the key
	ErrIndexNotFound = errors.New("FileStorage: indexfile is not found")
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
	index   *indexFile
}

// FileStorage creates the storage directory at dirpath to place the file into.
// NewFileStorage returns an error if it fails to prepare directory for the one.
func NewFileStorage(dirpath string) (storage *FileStorage, err error) {
	fileInfo, err := os.Stat(dirpath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}

		err = os.MkdirAll(dirpath, 0755)
		if err != nil {
			return nil, err
		}
		return NewFileStorage(dirpath)
	}

	if !fileInfo.IsDir() {
		return nil, ErrInvalidDir
	}

	index, err := NewOrLoadIndexFile(dirpath)
	if err != nil {
		return nil, err
	}

	trays := make(map[string]*tray)
	for _, entry := range index.getAliveTrayEntries() {
		t, err := newOrLoadTray(dirpath, entry.Key, entry.ExpiredAt, index)
		if err != nil {
			return nil, err
		}
		trays[entry.Key] = t
	}

	return &FileStorage{
		dirpath: path.Dir(dirpath),
		trays:   trays,
		index:   index,
	}, nil
}

func (f *FileStorage) get(key string, dest interface{}, useCache bool) (err error) {
	if _, ok := f.trays[key]; !ok {
		return ErrNoData
	}

	t := f.trays[key]
	return t.get(dest, useCache)
}

// Get scans the stored value from FileStorage into the destination pointer. An error is returned if the data is not stored or has already expired.
func (f *FileStorage) Get(key string, dest interface{}) (err error) {
	return f.get(key, dest, true)
}

// Get scans the stored value from the stored file into the destination pointer. An error is returned if the data is not stored or has already expired.
func (f *FileStorage) NoCacheGet(key string, dest interface{}) (err error) {
	return f.get(key, dest, false)
}

// IsExpired checks whether the stored data is expired. ErrLiveForever is returns if expiredAt is not set.
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
func (f *FileStorage) Set(key string, value interface{}, expiredAt *time.Time) (err error) {
	if _, ok := f.trays[key]; !ok {
		f.trays[key] = newTray(f.dirpath, key, expiredAt, f.index)
	}
	t := f.trays[key]
	return t.set(value)
}
