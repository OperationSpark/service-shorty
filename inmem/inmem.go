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
	Store map[string]shorty.Link
	// A mutex is used to synchronize read/write access to the map
	lock sync.RWMutex
}

func (i *Store) SaveLink(ctx context.Context, newLink shorty.Link) (shorty.Link, error) {
	i.lock.Lock()
	defer i.lock.Unlock()

	i.Store[newLink.Code] = newLink
	return newLink, nil
}

func (i *Store) FindLink(ctx context.Context, code string) (shorty.Link, error) {
	i.lock.RLock()
	defer i.lock.RUnlock()

	link, ok := i.Store[code]
	if !ok {
		return shorty.Link{}, shorty.ErrLinkNotFound
	}
	return link, nil
}

func (i *Store) FindAllLinks(ctx context.Context) (shorty.Links, error) {
	i.lock.RLock()
	defer i.lock.RUnlock()
	links := shorty.Links{}
	for _, l := range i.Store {
		links = append(links, &l)
	}
	return links, nil
}

func (i *Store) DeleteLink(ctx context.Context, code string) (int, error) {
	i.lock.Lock()
	defer i.lock.Unlock()
	panic("DeleteLink not implemented")
}
