package inmem

import (
	"context"
	"fmt"
	"sync"

	"github.com/operationspark/shorty/shortlink"
)

// NewInMemoryShortyStore creates an empty Shorty store.
func NewInMemoryShortyStore() *InMemoryShortyStore {
	return &InMemoryShortyStore{
		map[string]ShortLink{},
		sync.RWMutex{},
	}
}

// InMemoryShortyStore stores the short links in memory.
type InMemoryShortyStore struct {
	store map[string]ShortLink
	// A mutex is used to synchronize read/write access to the map
	lock sync.RWMutex
}

func (i *InMemoryShortyStore) BaseURL() string {
	return "https://ospk.org"
}

func (i *InMemoryShortyStore) CreateLink(ctx context.Context, newLink shortlink.ShortLink) (shortlink.ShortLink, error) {
	i.lock.Lock()
	defer i.lock.Unlock()

	code := CreateCode()
	s := ShortLink{
		Code:        code,
		CustomCode:  code,
		OriginalUrl: newLink.OriginalUrl,
		ShortURL:    fmt.Sprintf("%s/%s", i.BaseURL(), code),
	}
	i.store[code] = s
	return s, nil
}

func (i *InMemoryShortyStore) GetLink(ctx context.Context, code string) (shortlink.ShortLink, error) {
	i.lock.RLock()
	defer i.lock.RUnlock()
	panic("GetLink not implemented")
}

func (i *InMemoryShortyStore) GetLinks(ctx context.Context) ([]shortlink.ShortLink, error) {
	i.lock.RLock()
	defer i.lock.RUnlock()
	panic("GetLinks not implemented")
}

func (i *InMemoryShortyStore) UpdateLink(ctx context.Context, code string) (shortlink.ShortLink, error) {
	i.lock.Lock()
	defer i.lock.Unlock()
	panic("UpdateLink not implemented")
}

func (i *InMemoryShortyStore) DeleteLink(ctx context.Context, code string) (int, error) {
	i.lock.Lock()
	defer i.lock.Unlock()
	panic("DeleteLink not implemented")
}
