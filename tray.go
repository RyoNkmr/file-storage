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
	path      string
	cache     []byte
	mut       sync.RWMutex
	expiredAt *time.Time
}

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
		return false, ErrLiveForever
	}
	return s.expiredAt.Before(time.Now()), nil
}

func (s *tray) clear() (err error) {
	s.mut.Lock()
	defer s.mut.Unlock()

	s.cache = nil
	if _, err = os.Stat(s.path); err == nil {
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
		return ErrExpired
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
