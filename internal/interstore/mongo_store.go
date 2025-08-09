package interstore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/pkg/ctxlogger"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Event struct {
	ID uuid.UUID `bson:"id"`
	domain.Event
	CreatedAt time.Time `bson:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt"`
	DeletedAt time.Time `bson:"deletedAt"`
}

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

func (r MongoStore) GetInternalEvent(ctx context.Context, eventName string) (domain.Event, error) {
	l := ctxlogger.GetLogger(ctx)
	filter := bson.D{{Key: "event.name", Value: eventName}}
	var result Event

	if err := r.collectionInternalEvent.FindOne(ctx, filter).Decode(&result); err != nil {
		if errors.Is(mongo.ErrNoDocuments, err) {
			l.Warn("No documents found", "eventName", eventName)
			return domain.Event{}, domain.EventNotFound
		}

		return domain.Event{}, err
	}

	return result.Event, nil
}

func (r MongoStore) Save(ctx context.Context, event domain.Event) error {
	l := ctxlogger.GetLogger(ctx)
	if _, err := r.collectionInternalEvent.InsertOne(ctx, Event{
		ID:        uuid.New(),
		Event:     event,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}); err != nil {
		l.Error("Error on create internal event", "error", err)
		return fmt.Errorf("failed to create internal event: %w", err)
	}

	return nil
}

// func (r MongoStore) SaveInternalEvent(ctx context.Context, event intersvc.InternalEvent) error {
// 	filter := bson.D{{Key: "name", Value: bson.D{{Key: "$eq", Value: event.Name}}}}
// 	update := bson.D{{Key: "$set", Value: event}}

// 	if _, err := r.collectionInternalEvent.UpdateOne(ctx, filter, update); err != nil {
// 		fmt.Println("Error updating internal event:", err)
// 		return err
// 	}

// 	return nil
// }
