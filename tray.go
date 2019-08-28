package filestorage

import (
	"errors"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"sync"
	"time"

	"github.com/vmihailenco/msgpack"
)

type tray struct {
	Path      string
	ExpiredAt *time.Time
	Key       string

	cache []byte
	mut   sync.RWMutex
	index *indexFile
}

func newTray(dirpath, key string, expiredAt *time.Time, index *indexFile) *tray {
	path := path.Join(dirpath, key)
	return &tray{
		Path:      path,
		Key:       key,
		ExpiredAt: expiredAt,
		index:     index,
		mut:       sync.RWMutex{},
	}
}

func newOrLoadTray(dirpath, key string, expiredAt *time.Time, index *indexFile) (*tray, error) {
	t := newTray(dirpath, key, expiredAt, index)
	if _, err := os.Stat(t.Path); err != nil {
		return t, nil
	}
	err := t.loadCacheFromFile()
	return t, err
}

func (s *tray) loadCacheFromFile() (err error) {
	s.mut.RLock()
	defer s.mut.RUnlock()

	if isExpired, _ := s.isExpired(); isExpired {
		return nil
	}

	if _, err = os.Stat(s.Path); err != nil {
		return nil
	}

	file, err := os.Open(s.Path)
	if err != nil {
		return err
	}

	defer file.Close()

	b, err := ioutil.ReadAll(file)
	s.cache = b

	return err
}

func (s *tray) isExpired() (bool, error) {
	if s.index != nil {
		return s.index.isExpired(s.Key)
	}
	return false, ErrIndexNotFound
}

func (s *tray) clear() (err error) {
	s.mut.Lock()
	defer s.mut.Unlock()

	s.cache = nil
	if _, err = os.Stat(s.Path); err != nil {
		return err
	}

	if err = os.Remove(s.Path); err != nil {
		return err
	}

	if s.index != nil {
		return s.index.delete(s.Key)
	}
	return nil
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
		return ErrExpired
	}

	if useCache && s.cache != nil {
		err = msgpack.Unmarshal(s.cache, dest)
		return err
	}

	s.mut.RLock()
	defer s.mut.RUnlock()

	if _, err = os.Stat(s.Path); err != nil {
		return ErrNoData
	}

	file, err := os.Open(s.Path)
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
	file, err := os.OpenFile(s.Path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
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
	if s.index != nil {
		return s.index.update(s)
	}
	return nil
}
