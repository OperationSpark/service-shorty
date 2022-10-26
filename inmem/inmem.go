package inmem

import (
	"context"
	"sync"

	"github.com/operationspark/shorty/shorty"
)

// NewStore creates an empty Shorty store.
func NewStore() *Store {
	return &Store{
		map[string]shorty.Link{},
		sync.RWMutex{},
	}
}

// Store stores the short links in memory.
type Store struct {
	store map[string]shorty.Link
	// A mutex is used to synchronize read/write access to the map
	lock sync.RWMutex
}

func (i *Store) BaseURL() string {
	return "https://ospk.org"
}

func (i *Store) CreateLink(ctx context.Context, newLink shorty.Link) (shorty.Link, error) {
	i.lock.Lock()
	defer i.lock.Unlock()

	newLink.GenCode(i.BaseURL())
	i.store[newLink.Code] = newLink
	return newLink, nil
}

func (i *Store) GetLink(ctx context.Context, code string) (shorty.Link, error) {
	i.lock.RLock()
	defer i.lock.RUnlock()
	panic("GetLink not implemented")
}

func (i *Store) GetLinks(ctx context.Context) ([]shorty.Link, error) {
	i.lock.RLock()
	defer i.lock.RUnlock()
	panic("GetLinks not implemented")
}

func (i *Store) UpdateLink(ctx context.Context, code string) (shorty.Link, error) {
	i.lock.Lock()
	defer i.lock.Unlock()
	panic("UpdateLink not implemented")
}

func (i *Store) DeleteLink(ctx context.Context, code string) (int, error) {
	i.lock.Lock()
	defer i.lock.Unlock()
	panic("DeleteLink not implemented")
}
