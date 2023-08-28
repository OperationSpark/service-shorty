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
		TagsCollName  string
	}

	StoreOpts struct {
		URI string
	}

	Activity struct {
		// DateTime the tag was created.
		CreatedAt time.Time `json:"createdAt" bson:"createdAt"`

		// Short URL code used
		ShortCode string `json:"shortCode" bson:"shortCode"`
	}
)

// NewStore creates an empty Shorty store.
// MongoDB Connection to database "url-shortener"
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
		return &Store{}, fmt.Errorf("ping: %v", err)
	}

	// Grab the DB Name from the connection URI or Env vars
	connectionURI, err := url.Parse(o.URI)
	if err != nil {
		return &Store{}, fmt.Errorf("parse: %v", err)
	}

	dbName := strings.TrimPrefix(connectionURI.Path, "/")
	envDBName := os.Getenv("MONGO_DB_NAME")
	if len(envDBName) > 0 {
		dbName = envDBName
	}

	s := Store{
		Client:        client,
		DBName:        dbName,
		LinksCollName: "urls",
		TagsCollName:  "tags",
	}

	return &s, nil
}

// IncrementTotalClicks increments the "totalClicks" field and updates the database.
func (i *Store) AddTagActivity(ctx context.Context, codeData shorty.ShortCodeData) (int, error) {
	coll := i.Client.Database(i.DBName).Collection(i.TagsCollName)
	res, err := coll.UpdateOne(
		ctx,
		bson.D{{Key: "code", Value: codeData.Tag}},
		bson.D{
			{Key: "$push", Value: bson.D{{
				Key:   "activity",
				Value: bson.D{{Key: codeData.Code, Value: time.Now()}},
			}}},
		},
	)
	if err != nil {
		return 0, fmt.Errorf("Tags > updateOne: %v", err)
	}
	if res.ModifiedCount == 0 {
		return 0, shorty.ErrLinkNotFound
	}
	return int(res.ModifiedCount), nil
}

// SaveLink inserts a new Link into the database.
func (i *Store) SaveLink(ctx context.Context, newLink shorty.Link) (shorty.Link, error) {
	coll := i.Client.Database(i.DBName).Collection(i.LinksCollName)
	_, err := coll.InsertOne(ctx, newLink)
	if err != nil {
		return shorty.Link{}, fmt.Errorf("insertOne: %v", err)
	}
	return newLink, nil
}

// IncrementTotalClicks increments the "totalClicks" field and updates the database.
func (i *Store) IncrementTotalClicks(ctx context.Context, code string) (int, error) {
	coll := i.Client.Database(i.DBName).Collection(i.LinksCollName)
	res, err := coll.UpdateOne(
		ctx,
		bson.D{{Key: "code", Value: code}},
		bson.D{
			{Key: "$inc", Value: bson.D{{Key: "totalClicks", Value: 1}}},
			{Key: "$set", Value: bson.D{{Key: "updatedAt", Value: time.Now()}}},
		},
	)
	if err != nil {
		return 0, fmt.Errorf("updateOne: %v", err)
	}
	if res.ModifiedCount == 0 {
		return 0, shorty.ErrLinkNotFound
	}
	return int(res.ModifiedCount), nil
}

// FindLink finds the Link with the given code.
func (i *Store) FindLink(ctx context.Context, code string) (shorty.Link, error) {
	var link shorty.Link
	coll := i.Client.Database(i.DBName).Collection(i.LinksCollName)

	res := coll.FindOne(ctx, bson.D{{Key: "code", Value: code}})
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

// FindAllLinks returns all the links from the database.
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

// UpdateLink updates a links originalUrl if given. If a code is given, shortCode, code, and customCode are updated. The updatedAt is set to the current time.
func (i *Store) UpdateLink(ctx context.Context, code string, link shorty.Link) (shorty.Link, error) {
	coll := i.Client.Database(i.DBName).Collection(i.LinksCollName)

	updateDoc := bson.D{
		{Key: "updatedAt", Value: time.Now()},
	}
	if len(link.OriginalUrl) > 0 {
		updateDoc = append(updateDoc, bson.E{Key: "originalUrl", Value: link.OriginalUrl})
	}

	if len(link.CustomCode) > 0 {
		updateDoc = append(updateDoc,
			bson.E{Key: "shortUrl", Value: link.ShortURL},
			bson.E{Key: "code", Value: link.Code},
			bson.E{Key: "customCode", Value: link.CustomCode},
		)
	}
	res, err := coll.UpdateOne(
		ctx,
		bson.D{{Key: "code", Value: code}},
		bson.D{{Key: "$set", Value: updateDoc}},
	)

	if err != nil {
		return link, fmt.Errorf("replaceOne: %v", err)
	}
	if res.ModifiedCount == 0 {
		return link, shorty.ErrLinkNotFound
	}
	return link, nil
}

// DeleteLink deletes a link from the database.
func (i *Store) DeleteLink(ctx context.Context, code string) (int, error) {
	coll := i.Client.Database(i.DBName).Collection(i.LinksCollName)
	res, err := coll.DeleteOne(ctx, bson.D{{Key: "code", Value: code}})
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
