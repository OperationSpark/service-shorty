package mongodb

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/operationspark/shorty/shorty"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type (
	// InMemoryShortyStore stores the short links in memory.
	Store struct {
		Client        *mongo.Client
		DBName        string
		LinksCollName string
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

	// Grab the DB Name from the connection URI or Env vars
	connectionURI, err := url.Parse(o.URI)
	if err != nil {
		panic(err)
	}
	dbName := strings.TrimPrefix(connectionURI.Path, "/")
	envDBName := os.Getenv("MONGO_DB_NAME")
	if len(envDBName) > 0 {
		dbName = envDBName
	}
	fmt.Println("Database name: " + dbName)

	s := Store{
		Client:        client,
		DBName:        dbName,
		LinksCollName: "urls",
	}

	err = s.CreateCodeIndex(context.Background())
	if err != nil {
		return &s, fmt.Errorf("createCodeIndex: %v", err)
	}
	return &s, nil
}

// CreateCodeIndex creates an index on the 'code' field in the urls collection
func (i *Store) CreateCodeIndex(ctx context.Context) error {
	indexModel := mongo.IndexModel{Keys: bson.D{{"code", 1}}}

	_, err := i.Client.
		Database(i.DBName).
		Collection(i.LinksCollName).
		Indexes().
		CreateOne(ctx, indexModel)
	if err != nil {
		return fmt.Errorf("createOne: %v", err)
	}
	return nil
}

func (i *Store) SaveLink(ctx context.Context, newLink shorty.Link) (shorty.Link, error) {
	coll := i.Client.Database(i.DBName).Collection(i.LinksCollName)
	// TODO: Maybe use upsert
	_, err := coll.InsertOne(ctx, newLink)
	if err != nil {
		return shorty.Link{}, fmt.Errorf("insertOne: %v", err)
	}
	return newLink, nil
}

func (i *Store) FindLink(ctx context.Context, code string) (shorty.Link, error) {
	var link shorty.Link
	coll := i.Client.Database(i.DBName).Collection(i.LinksCollName)

	res := coll.FindOne(ctx, bson.D{{"code", code}})
	if res.Err() != nil {
		if res.Err() == mongo.ErrNoDocuments {
			return link, shorty.ErrLinkNotFound
		}
		return link, fmt.Errorf("findOne: %v", res.Err())
	}

	err := res.Decode(&link)
	if err != nil {
		return link, fmt.Errorf("decode: %v", err)
	}
	return link, nil
}

func (i *Store) FindAllLinks(ctx context.Context) (shorty.Links, error) {
	coll := i.Client.Database(i.DBName).Collection(i.LinksCollName)
	cur, err := coll.Find(ctx, bson.D{{}})
	if err != nil {
		return shorty.Links{}, fmt.Errorf("find: %v", err)
	}
	defer cur.Close(ctx)

	links := make(shorty.Links, cur.RemainingBatchLength())
	cur.All(ctx, &links)
	return links, nil
}

func (i *Store) UpdateLink(ctx context.Context, code string) (shorty.Link, error) {
	panic("UpdateLink not implemented")
}

func (i *Store) DeleteLink(ctx context.Context, code string) (int, error) {
	panic("DeleteLink not implemented")
}
