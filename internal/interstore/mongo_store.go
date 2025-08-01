package interstore

import (
	"context"
	"errors"
	"fmt"

	"github.com/IsaacDSC/webhook/internal/intersvc"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const dbName = "webhook_service"
const collectionInternalEvent = "internal_events"
const collectionExternalEvent = "external_events"

type MongoStore struct {
	collectionInternalEvent *mongo.Collection
	collectionExternalEvent *mongo.Collection
}

func NewMongoStore(client *mongo.Client) *MongoStore {
	db := client.Database(dbName)
	return &MongoStore{
		collectionInternalEvent: db.Collection(collectionInternalEvent),
		collectionExternalEvent: db.Collection(collectionExternalEvent),
	}
}

func (r MongoStore) GetInternalEvent(ctx context.Context, eventName string) (output intersvc.InternalEvent, err error) {
	filter := bson.D{{Key: "name", Value: eventName}}
	err = r.collectionInternalEvent.FindOne(ctx, filter).Decode(&output)
	if err != nil {
		if errors.Is(mongo.ErrNoDocuments, err) {
			fmt.Println("No documents found: ", eventName)
			return output, nil
		}
	}

	return
}

func (r MongoStore) CreateInternalEvent(ctx context.Context, event intersvc.InternalEvent) error {
	if _, err := r.collectionInternalEvent.InsertOne(ctx, event); err != nil {
		fmt.Println("Error on create internal event:", err)
		return err
	}

	return nil
}

func (r MongoStore) SaveInternalEvent(ctx context.Context, event intersvc.InternalEvent) error {
	filter := bson.D{{Key: "name", Value: bson.D{{Key: "$eq", Value: event.Name}}}}
	update := bson.D{{Key: "$set", Value: event}}

	if _, err := r.collectionInternalEvent.UpdateOne(ctx, filter, update); err != nil {
		fmt.Println("Error updating internal event:", err)
		return err
	}

	return nil
}
