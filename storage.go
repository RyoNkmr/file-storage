package storage

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"sync"
	"time"

	"github.com/vmihailenco/msgpack"
)

type FileStorage interface {
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

var ErrNoData = errors.New("fileStorage: no data")

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

type tray struct {
	path      string
	cache     []byte
	mut       sync.RWMutex
	expiredAt *time.Time
}

var ErrNeverExpire = errors.New("filetray: expiredAt is not set")

func newTray(dirpath, filename string, expiredAt *time.Time) *tray {
	path := path.Join(dirpath, filename)
	return &tray{
		path:      path,
		mut:       sync.RWMutex{},
		expiredAt: expiredAt,
	}
}

func (s *tray) isExpired() (isExpired bool, err error) {
	if s.expiredAt == nil {
		return false, ErrNeverExpire
	}
	return s.expiredAt.Before(time.Now()), nil
}

func (s *tray) clear() (err error) {
	s.mut.Lock()
	defer s.mut.Unlock()

	s.cache = nil
	if _, err = os.Stat(s.path); err == nil {
		fmt.Println("file exist, removed")
		return os.Remove(s.path)
	}
	return err
}

func (s *tray) get(dest interface{}, useCache bool) (err error) {
	rv := reflect.ValueOf(dest)
	if rv.Kind() != reflect.Ptr {
		return errors.New("must pass a pointer")
	}

	if isExpired, _ := s.isExpired(); isExpired {
		if err = s.clear(); err != nil {
			return err
		}
		fmt.Println("expired")
		return ErrNoData
	}

	if useCache && s.cache != nil {
		err = msgpack.Unmarshal(s.cache, dest)
		return err
	}

	s.mut.RLock()
	defer s.mut.RUnlock()

	if _, err = os.Stat(s.path); err != nil {
		return ErrNoData
	}

	file, err := os.Open(s.path)
	if err != nil {
		return err
	}

	defer file.Close()
	b, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	err = msgpack.Unmarshal(b, dest)
	return err
}

func (s *tray) set(value interface{}) (err error) {
	s.mut.Lock()
	defer s.mut.Unlock()
	file, err := os.OpenFile(s.path, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer file.Close()
	var b []byte
	if b, err = msgpack.Marshal(value); err != nil {
		return err
	}
	if _, err := file.Write(b); err != nil {
		return err
	}
	s.cache = b
	return nil
}
