package filestorage

import (
	"time"
)

type entry struct {
	Key       string
	ExpiredAt *time.Time
	UpdatedAt time.Time
}

type indexFile struct {
	data map[string]*entry
	tray *tray
}

const indexKey = ".index"

func NewOrLoadIndexFile(dirpath string) (*indexFile, error) {
	t, err := newOrLoadTray(dirpath, indexKey, nil, nil)
	if err != nil {
		return nil, err
	}
	var data map[string]*entry
	if err = t.get(&data, true); err != nil {
		return nil, err
	}
	return &indexFile{data, t}, nil
}

func (i *indexFile) save() error {
	return i.tray.set(i.data)
}

func (i *indexFile) isExpired(key string) (isExpired bool, err error) {
	e, ok := i.data[key]
	if !ok {
		return false, ErrNoData
	}

	if e.ExpiredAt == nil {
		return false, ErrLiveForever
	}

	return e.ExpiredAt.Before(time.Now()), nil
}

func (i *indexFile) update(tray *tray) error {
	v, ok := i.data[tray.Key]

	if ok {
		v.Key = tray.Key
		v.ExpiredAt = tray.ExpiredAt
		v.UpdatedAt = time.Now()
		return i.save()
	}

	i.data[tray.Key] = &entry{
		Key:       tray.Key,
		ExpiredAt: tray.ExpiredAt,
		UpdatedAt: time.Now(),
	}
	return i.save()
}

func (i *indexFile) delete(key string) error {
	delete(i.data, key)
	return i.save()
}

func (i *indexFile) getAliveTrayEntries() (entries []*entry) {
	entries = make([]*entry, 0, len(i.data))
	for _, entry := range i.data {
		if isExpired, _ := i.isExpired(entry.Key); !isExpired {
			entries = append(entries, entry)
		}
	}
	return entries
}
