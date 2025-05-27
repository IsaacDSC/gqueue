package repository

import (
	"context"
	"testing"
	"time"

	"github.com/IsaacDSC/webhook/internal/structs"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func setupMongoContainer(t *testing.T) (*mongo.Client, func()) {
	ctx := context.Background()
	mongoContainer, err := mongodb.RunContainer(ctx,
		testcontainers.WithImage("mongo:6.0"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("Waiting for connections").
				WithStartupTimeout(time.Second*60),
		),
	)
	if err != nil {
		t.Fatalf("Failed to start MongoDB container: %s", err)
	}

	connectionString, err := mongoContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("Failed to get MongoDB connection string: %s", err)
	}

	clientOpts := options.Client().ApplyURI(connectionString)
	client, err := mongo.Connect(clientOpts)
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %s", err)
	}

	// Ping the database to ensure the connection is established
	err = client.Ping(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to ping MongoDB: %s", err)
	}

	// Return the client and a cleanup function
	return client, func() {
		if err := client.Disconnect(ctx); err != nil {
			t.Fatalf("Failed to disconnect MongoDB client: %s", err)
		}
		if err := mongoContainer.Terminate(ctx); err != nil {
			t.Fatalf("Failed to terminate MongoDB container: %s", err)
		}
	}
}

func TestRepository_CreateExternalEvent(t *testing.T) {
	// Setup MongoDB container
	client, cleanup := setupMongoContainer(t)
	defer cleanup()

	// Create repository
	repo := NewRepository(client)

	// Create test event
	ctx := context.Background()
	id := uuid.New()
	event := structs.ExternalEvent{
		ID:   id,
		Name: "test-event",
		Data: structs.Data{
			"key": "value",
		},
		Triggers: []structs.Trigger{
			{
				ID:          uuid.New(),
				ServiceName: "test-service",
				Type:        structs.TriggerTypeFireForGet,
				BaseUrl:     "http://localhost",
				Path:        "/webhook",
				CreatedAt:   time.Now(),
			},
		},
	}

	// Test CreateExternalEvent
	err := repo.CreateExternalEvent(ctx, event)
	assert.NoError(t, err)

	// Verify the event was saved
	filter := bson.D{{"id", id}}
	var result structs.ExternalEvent
	err = client.Database(dbName).Collection(collectionExternalEvent).
		FindOne(ctx, filter).Decode(&result)

	assert.NoError(t, err)
	assert.Equal(t, event.ID, result.ID)
	assert.Equal(t, event.Name, result.Name)
	assert.Equal(t, "value", result.Data["key"])
}

func TestRepository_SaveExternalEvent(t *testing.T) {
	// Setup MongoDB container
	client, cleanup := setupMongoContainer(t)
	defer cleanup()

	// Create repository
	repo := NewRepository(client)

	// Create test event
	ctx := context.Background()
	id := uuid.New()
	event := structs.ExternalEvent{
		ID:   id,
		Name: "test-event",
		Data: structs.Data{
			"key": "value",
		},
	}

	// First create the event
	err := repo.CreateExternalEvent(ctx, event)
	assert.NoError(t, err)

	// Update the event
	event.Data["key"] = "updated-value"
	err = repo.SaveExternalEvent(ctx, event)
	assert.NoError(t, err)

	// Verify the event was updated
	filter := bson.D{{"id", id}}
	var result structs.ExternalEvent
	err = client.Database(dbName).Collection(collectionExternalEvent).
		FindOne(ctx, filter).Decode(&result)

	assert.NoError(t, err)
	assert.Equal(t, event.ID, result.ID)
	assert.Equal(t, "updated-value", result.Data["key"])
}

func TestRepository_CreateInternalEvent(t *testing.T) {
	// Setup MongoDB container
	client, cleanup := setupMongoContainer(t)
	defer cleanup()

	// Create repository
	repo := NewRepository(client)

	// Create test event
	ctx := context.Background()
	id := uuid.New()
	now := time.Now()
	event := structs.InternalEvent{
		ID:          id,
		Name:        "test-internal-event",
		ServiceName: "test-service",
		RepoUrl:     "https://github.com/test/test",
		TeamOwner:   "test-team",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Test CreateInternalEvent
	err := repo.CreateInternalEvent(ctx, event)
	assert.NoError(t, err)

	// Verify the event was saved
	filter := bson.D{{"id", id}}
	var result structs.InternalEvent
	err = client.Database(dbName).Collection(collectionInternalEvent).
		FindOne(ctx, filter).Decode(&result)

	assert.NoError(t, err)
	assert.Equal(t, event.ID, result.ID)
	assert.Equal(t, event.Name, result.Name)
	assert.Equal(t, event.ServiceName, result.ServiceName)
}

func TestRepository_GetInternalEvent(t *testing.T) {
	// Setup MongoDB container
	client, cleanup := setupMongoContainer(t)
	defer cleanup()

	// Create repository
	repo := NewRepository(client)

	// Create test event
	ctx := context.Background()
	id := uuid.New()
	eventName := "test-internal-event-for-get"
	now := time.Now()
	event := structs.InternalEvent{
		ID:          id,
		Name:        eventName,
		ServiceName: "test-service",
		RepoUrl:     "https://github.com/test/test",
		TeamOwner:   "test-team",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// First create the event
	err := repo.CreateInternalEvent(ctx, event)
	assert.NoError(t, err)

	// Test GetInternalEvent
	result, err := repo.GetInternalEvent(ctx, eventName)
	assert.NoError(t, err)

	// Verify the returned event
	assert.Equal(t, event.ID, result.ID)
	assert.Equal(t, event.Name, result.Name)
	assert.Equal(t, event.ServiceName, result.ServiceName)

	// Test getting non-existent event
	result, err = repo.GetInternalEvent(ctx, "non-existent-event")
	assert.NoError(t, err) // Should not return error, just empty result
	assert.Equal(t, uuid.Nil, result.ID)
}
