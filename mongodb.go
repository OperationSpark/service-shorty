package shorty

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type (
	// InMemoryShortyStore stores the short links in memory.
	MongoShortyStore struct {
		client            *mongo.Client
		dbName            string
		urlCollectionName string
	}

	MongoStoreOpts struct {
		uri string
	}

	ShortLinkDB struct {
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

// NewMongoShortyStore creates an empty Shorty store.
func NewMongoShortyStore(o MongoStoreOpts) (*MongoShortyStore, error) {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(o.uri))
	if err != nil {
		return &MongoShortyStore{}, fmt.Errorf("connect: %v", err)
	}

	// Ping the primary
	if err := client.Ping(context.TODO(), readpref.Primary()); err != nil {
		panic(err)
	}
	fmt.Println("Successfully connected and pinged.")

	connectionURI, err := url.Parse(o.uri)
	if err != nil {
		panic(err)
	}

	dbName := strings.TrimPrefix(connectionURI.Path, "/")
	fmt.Println(dbName)
	return &MongoShortyStore{
		client:            client,
		dbName:            dbName,
		urlCollectionName: "urls",
	}, nil
}

func (i *MongoShortyStore) BaseURL() string {
	return "https://ospk.org"
}

func (i *MongoShortyStore) CreateLink(ctx context.Context, newLink ShortLink) (ShortLink, error) {

	code := CreateCode()
	s := ShortLink{
		Code:        code,
		CustomCode:  code,
		OriginalUrl: newLink.OriginalUrl,
		ShortURL:    fmt.Sprintf("%s/%s", i.BaseURL(), code),
	}

	coll := i.client.Database(i.dbName).Collection(i.urlCollectionName)
	doc := ShortLinkDB(s)

	_, err := coll.InsertOne(ctx, doc)
	if err != nil {
		return ShortLink{}, fmt.Errorf("insertOne: %v", err)
	}
	return s, nil
}

func (i *MongoShortyStore) GetLink(ctx context.Context, code string) (ShortLink, error) {
	panic("GetLink not implemented")
}

func (i *MongoShortyStore) GetLinks(ctx context.Context) ([]ShortLink, error) {
	panic("GetLinks not implemented")
}

func (i *MongoShortyStore) UpdateLink(ctx context.Context, code string) (ShortLink, error) {
	panic("UpdateLink not implemented")
}

func (i *MongoShortyStore) DeleteLink(ctx context.Context, code string) (int, error) {
	panic("DeleteLink not implemented")
}
