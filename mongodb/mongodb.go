package mongodb

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/operationspark/shorty/shorty"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type (
	// InMemoryShortyStore stores the short links in memory.
	Store struct {
		Client            *mongo.Client
		DBName            string
		URLCollectionName string
	}

	StoreOpts struct {
		URI string
	}
)

// NewStore creates an empty Shorty store.
func NewStore(o StoreOpts) (*Store, error) {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(o.URI))
	if err != nil {
		return &Store{}, fmt.Errorf("connect: %v", err)
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
	return &Store{
		Client:            client,
		DBName:            dbName,
		URLCollectionName: "urls",
	}, nil
}

func (i *Store) BaseURL() string {
	return "https://ospk.org"
}

func (i *Store) CreateLink(ctx context.Context, newLink shorty.Link) (shorty.Link, error) {

	newLink.GenCode(i.BaseURL())

	coll := i.Client.Database(i.DBName).Collection(i.URLCollectionName)

	_, err := coll.InsertOne(ctx, newLink)
	if err != nil {
		return shorty.Link{}, fmt.Errorf("insertOne: %v", err)
	}
	return newLink, nil
}

func (i *Store) GetLink(ctx context.Context, code string) (shorty.Link, error) {
	panic("GetLink not implemented")
}

func (i *Store) GetLinks(ctx context.Context) ([]shorty.Link, error) {
	panic("GetLinks not implemented")
}

func (i *Store) UpdateLink(ctx context.Context, code string) (shorty.Link, error) {
	panic("UpdateLink not implemented")
}

func (i *Store) DeleteLink(ctx context.Context, code string) (int, error) {
	panic("DeleteLink not implemented")
}
