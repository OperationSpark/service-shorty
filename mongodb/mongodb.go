package mongodb

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

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
	client, err := mongo.Connect(
		context.TODO(),
		options.Client().ApplyURI(o.URI),
	)
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

	return &s, nil
}

func (i *Store) SaveLink(ctx context.Context, newLink shorty.Link) (shorty.Link, error) {
	coll := i.Client.Database(i.DBName).Collection(i.LinksCollName)
	_, err := coll.InsertOne(ctx, newLink)
	if err != nil {
		return shorty.Link{}, fmt.Errorf("insertOne: %v", err)
	}
	return newLink, nil
}

func (i *Store) IncrementTotalClicks(ctx context.Context, code string) (int, error) {
	coll := i.Client.Database(i.DBName).Collection(i.LinksCollName)
	res, err := coll.UpdateOne(
		ctx,
		bson.D{{"code", code}},
		bson.D{
			{"$inc", bson.D{{"totalClicks", 1}}}},
	)
	if err != nil {
		return 0, fmt.Errorf("updateOne: %v", err)
	}
	if res.ModifiedCount == 0 {
		return 0, shorty.ErrLinkNotFound
	}
	return int(res.ModifiedCount), nil
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

func (i *Store) UpdateLink(ctx context.Context, code string, link shorty.Link) (shorty.Link, error) {
	coll := i.Client.Database(i.DBName).Collection(i.LinksCollName)

	updateDoc := bson.D{
		{"updatedAt", time.Now()},
	}
	if len(link.OriginalUrl) > 0 {
		updateDoc = append(updateDoc, bson.E{"originalUrl", link.OriginalUrl})
	}

	if len(link.CustomCode) > 0 {
		updateDoc = append(updateDoc,
			bson.E{"shortUrl", link.ShortURL},
			bson.E{"code", link.Code},
			bson.E{"customCode", link.CustomCode},
		)
	}
	res, err := coll.UpdateOne(
		ctx,
		bson.D{{"code", code}},
		bson.D{{"$set", updateDoc}},
	)

	if err != nil {
		return link, fmt.Errorf("replaceOne: %v", err)
	}
	if res.ModifiedCount == 0 {
		return link, shorty.ErrLinkNotFound
	}
	return link, nil
}

func (i *Store) DeleteLink(ctx context.Context, code string) (int, error) {
	coll := i.Client.Database(i.DBName).Collection(i.LinksCollName)
	res, err := coll.DeleteOne(ctx, bson.D{{"code", code}})
	if err != nil {
		return 0, fmt.Errorf("deleteOne: %v", err)
	}
	return int(res.DeletedCount), nil
}

// CheckCodeInUse returns false if the code is available for use, or true if the code is already in use.
func (i *Store) CheckCodeInUse(ctx context.Context, code string) (bool, error) {
	_, err := i.FindLink(ctx, code)
	if err != nil {
		if err == shorty.ErrLinkNotFound {
			return false, nil
		}
		// Default to true if there is an error
		return true, err
	}
	// Code found
	return true, nil
}
