package travellist

import (
	"context"
	"errors"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"log"
	"time"
)

type DBRepository struct {
	client *mongo.Client
	db     *mongo.Database
	col    *mongo.Collection
}

type Repository interface {
	ping() (string, error)
	findAll(ctx context.Context) (*Travels, error)
	findOne(ctx context.Context, id string) (*Travel, error)
	insertOne(ctx context.Context, travel *Travel) error
	updateOne(ctx context.Context, id string, travel *Travel) error
	updateField(ctx context.Context, id, field string, value interface{}) error
	deleteOne(ctx context.Context, id string) error
	Close()
}

func NewRepo(uri string) (Repository, error) {
	client, err := mongo.NewClient(options.Client().ApplyURI(uri))
	log.Println("db client created")
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	err = client.Connect(ctx)

	if err != nil {
		return nil, err
	}
	log.Println("db client connected")

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, err
	}
	log.Println("db client ping")

	dbName := viper.Get("DATABASE_NAME").(string)
	db := client.Database(dbName)
	col := db.Collection(viper.Get("TRAVEL_COLLECTION").(string))
	return &DBRepository{
		client: client,
		db:     db,
		col:    col,
	}, nil
}

func (d *DBRepository) ping() (string, error) {
	ctx := context.Background()
	err := d.client.Ping(ctx, readpref.Primary())
	if err != nil {
		return "", errors.New("connection error")
	}
	return "connection to database established", nil
}

func (d *DBRepository) findAll(ctx context.Context) (*Travels, error) {
	c, err := d.col.Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	var travels Travels

	for c.Next(ctx) {
		var travel Travel
		if err := c.Decode(&travel); err != nil {
			return nil, err
		}
		travels = append(travels, travel)
	}
	if err := c.Close(ctx); err != nil {
		return nil, err
	}
	return &travels, nil
}

func (d *DBRepository) findOne(ctx context.Context, id string) (*Travel, error) {
	objectId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	res := d.col.FindOne(ctx, bson.M{"_id": objectId})
	var travel Travel
	if err := res.Decode(&travel); err != nil {
		return nil, err
	}
	return &travel, nil
}

func (d *DBRepository) insertOne(ctx context.Context, travel *Travel) error {
	travel.ObjectID = primitive.NewObjectID()
	if _, err := d.col.InsertOne(ctx, travel); err != nil {
		return err
	}
	return nil
}

func (d *DBRepository) updateOne(ctx context.Context, id string, travel *Travel) error {
	travel.ObjectID, _ = primitive.ObjectIDFromHex(id)
	filter := bson.M{"_id": travel.ObjectID}
	if _, err := d.col.ReplaceOne(ctx, filter, travel); err != nil {
		return err
	}
	return nil
}

func (d *DBRepository) updateField(ctx context.Context, id, field string, value interface{}) error {
	objectID, _ := primitive.ObjectIDFromHex(id)
	filter := bson.M{"_id": objectID}
	update := bson.D{{
		"$set", bson.D{{
			field, value,
		}},
	}}
	if _, err := d.col.ReplaceOne(ctx, filter, update); err != nil {
		return err
	}
	return nil
}

func (d *DBRepository) deleteOne(ctx context.Context, id string) error {
	objectId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	if _, err := d.col.DeleteOne(ctx, bson.M{"_id": objectId}); err != nil {
		return err
	}
	return nil
}

func (d *DBRepository) Close() {
	if err := d.client.Disconnect(context.Background()); err != nil {
		log.Fatal(err)
	}
}
