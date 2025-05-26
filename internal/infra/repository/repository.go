package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/IsaacDSC/webhook/internal/structs"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const dbName = "webhook_service"
const collectionInternalEvent = "internal_events"
const collectionExternalEvent = "external_events"

var _ Repository = (*MongoRepo)(nil)

type MongoRepo struct {
	collectionInternalEvent *mongo.Collection
	collectionExternalEvent *mongo.Collection
}

func NewRepository(client *mongo.Client) *MongoRepo {
	db := client.Database(dbName)
	return &MongoRepo{
		collectionInternalEvent: db.Collection(collectionInternalEvent),
		collectionExternalEvent: db.Collection(collectionExternalEvent),
	}
}

func (r MongoRepo) CreateExternalEvent(ctx context.Context, event structs.ExternalEvent) error {
	if _, err := r.collectionExternalEvent.InsertOne(ctx, event); err != nil {
		return fmt.Errorf("error on create external event: %w", err)
	}
	return nil
}

func (r MongoRepo) SaveExternalEvent(ctx context.Context, event structs.ExternalEvent) error {
	filter := bson.D{{"id", bson.D{{"$eq", event.ID}}}}
	update := bson.D{{"$set", event}}
	if _, err := r.collectionExternalEvent.UpdateOne(ctx, filter, update); err != nil {
		return fmt.Errorf("error on create external event: %w", err)
	}
	return nil
}

func (r MongoRepo) CreateInternalEvent(ctx context.Context, event structs.InternalEvent) error {
	if _, err := r.collectionInternalEvent.InsertOne(ctx, event); err != nil {
		fmt.Println("Error on create internal event:", err)
		return err
	}

	return nil
}

func (r MongoRepo) SaveInternalEvent(ctx context.Context, event structs.InternalEvent) error {
	filter := bson.D{{"name", bson.D{{"$eq", event.Name}}}}
	update := bson.D{{"$set", event}}

	if _, err := r.collectionInternalEvent.UpdateOne(ctx, filter, update); err != nil {
		fmt.Println("Error updating internal event:", err)
		return err
	}

	return nil
}

func (r MongoRepo) GetInternalEvent(ctx context.Context, eventName string) (output structs.InternalEvent, err error) {
	filter := bson.D{{"name", eventName}}
	err = r.collectionInternalEvent.FindOne(ctx, filter).Decode(&output)
	if err != nil {
		if errors.Is(mongo.ErrNoDocuments, err) {
			fmt.Println("No documents found")
			return output, nil
		}
	}

	return
}
