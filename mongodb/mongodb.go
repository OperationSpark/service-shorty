package mongodb

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/operationspark/shorty/shortlink"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type (
	// InMemoryShortyStore stores the short links in memory.
	MongoShortyStore struct {
		Client            *mongo.Client
		DBName            string
		URLCollectionName string
	}

	StoreOpts struct {
		URI string
	}
)

// NewStore creates an empty Shorty store.
func NewStore(o StoreOpts) (*MongoShortyStore, error) {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(o.URI))
	if err != nil {
		return &MongoShortyStore{}, fmt.Errorf("connect: %v", err)
	}

	// Ping the primary
	if err := client.Ping(context.TODO(), readpref.Primary()); err != nil {
		panic(err)
	}
	fmt.Println("Successfully connected and pinged.")

	connectionURI, err := url.Parse(o.URI)
	if err != nil {
		panic(err)
	}

	dbName := strings.TrimPrefix(connectionURI.Path, "/")
	fmt.Println(dbName)
	return &MongoShortyStore{
		Client:            client,
		DBName:            dbName,
		URLCollectionName: "urls",
	}, nil
}

func (i *MongoShortyStore) BaseURL() string {
	return "https://ospk.org"
}

func (i *MongoShortyStore) CreateLink(ctx context.Context, newLink shortlink.ShortLink) (shortlink.ShortLink, error) {

	newLink.GenCode(i.BaseURL())

	coll := i.Client.Database(i.DBName).Collection(i.URLCollectionName)

	_, err := coll.InsertOne(ctx, newLink)
	if err != nil {
		return shortlink.ShortLink{}, fmt.Errorf("insertOne: %v", err)
	}
	return newLink, nil
}

func (i *MongoShortyStore) GetLink(ctx context.Context, code string) (shortlink.ShortLink, error) {
	panic("GetLink not implemented")
}

func (i *MongoShortyStore) GetLinks(ctx context.Context) ([]shortlink.ShortLink, error) {
	panic("GetLinks not implemented")
}

func (i *MongoShortyStore) UpdateLink(ctx context.Context, code string) (shortlink.ShortLink, error) {
	panic("UpdateLink not implemented")
}

func (i *MongoShortyStore) DeleteLink(ctx context.Context, code string) (int, error) {
	panic("DeleteLink not implemented")
}
