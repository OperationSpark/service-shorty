package shorty

import (
	"context"
	"sync"
)

// NewInMemoryShortyStore creates an empty Shorty store.
func NewInMemoryShortyStore() *InMemoryShortyStore {
	return &InMemoryShortyStore{
		map[string]int{},
		sync.RWMutex{},
	}
}

// InMemoryShortyStore stores the short links in memory.
type InMemoryShortyStore struct {
	store map[string]int
	// A mutex is used to synchronize read/write access to the map
	lock sync.RWMutex
}

func (i *InMemoryShortyStore) CreateLink(ctx context.Context, newLink ShortLink) (ShortLink, error) {
	i.lock.Lock()
	defer i.lock.Unlock()
	panic("CreateLinks not implemented")
}

func (i *InMemoryShortyStore) GetLink(ctx context.Context, code string) (ShortLink, error) {
	i.lock.RLock()
	defer i.lock.RUnlock()
	panic("GetLink not implemented")
}

func (i *InMemoryShortyStore) GetLinks(ctx context.Context) ([]ShortLink, error) {
	i.lock.RLock()
	defer i.lock.RUnlock()
	panic("GetLinks not implemented")
}

func (i *InMemoryShortyStore) UpdateLink(ctx context.Context, code string) (ShortLink, error) {
	i.lock.Lock()
	defer i.lock.Unlock()
	panic("UpdateLink not implemented")
}

func (i *InMemoryShortyStore) DeleteLink(ctx context.Context, code string) (int, error) {
	i.lock.Lock()
	defer i.lock.Unlock()
	panic("DeleteLink not implemented")
}
