package inmem

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type (
	ShortLink struct {
		// Shortened URL result. Ex: https://ospk.org/bas12d21dc.
		ShortURL string `bson:"shortUrl"`
		// Short Code used as the path of the short URL. Ex: bas12d21dc.
		Code string `bson:"code"`
		// Optional custom short code passed when creating or updating the short URL.
		CustomCode string `bson:"customCode"`
		// The URL where the short URL redirects.
		OriginalUrl string `bson:"originalUrl"`
		// Count of times the short URL has been used.
		TotalClicks int `bson:"totalClicks"`
		// Identifier of the entity that created the short URL.
		CreatedBy string `bson:"createdBy"`
		// DateTime the URL was created.
		CreatedAt time.Time `bson:"createdAt"`
		// DateTime the URL was last updated.
		UpdatedAt time.Time `bson:"updatedAt"`
	}
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

func (i *InMemoryShortyStore) CreateLink(ctx context.Context, newLink ShortLink) (ShortLink, error) {
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
