package interstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/pkg/ctxlogger"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStoreFromDSN creates a new PostgresStore from a database DSN string
func NewPostgresStoreFromDSN(dsn string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresStore{db: db}, nil
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{
		db: db,
	}
}

func (r *PostgresStore) GetInternalEvent(ctx context.Context, eventName, serviceName string, eventType string, state string) ([]domain.Event, error) {
	l := ctxlogger.GetLogger(ctx)

	if eventName != "" {
		uniqueKey := r.getUniqueKey(eventName, serviceName, eventType, state)
		event, err := r.getEventByUniqueKey(ctx, uniqueKey)
		if err != nil {
			if errors.Is(err, domain.EventNotFound) {
				return nil, nil
			}
			l.Error("Error on get internal event by unique key", "error", err)
			return nil, fmt.Errorf("failed to get internal event by unique key: %w", err)
		}

		return []domain.Event{event}, nil
	}

	query := `
		SELECT name, service_name, repo_url, team_owner, triggers 
		FROM events 
		WHERE service_name = $1 AND deleted_at IS NULL
	`

	rows, err := r.db.Query(query, serviceName)
	if err != nil {
		l.Error("Error on get internal events", "error", err)
		return nil, fmt.Errorf("failed to get internal events: %w", err)
	}

	var event domain.Event
	var events []domain.Event
	var triggersJSON []byte
	for rows.Next() {
		if err := rows.Scan(&event.Name, &event.ServiceName, &event.RepoURL, &event.TeamOwner, &triggersJSON); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(triggersJSON, &event.Triggers); err != nil {
			return nil, fmt.Errorf("failed to unmarshal triggers: %w", err)
		}

		events = append(events, event)
	}

	return events, nil
}

func (r *PostgresStore) Save(ctx context.Context, event domain.Event) error {
	l := ctxlogger.GetLogger(ctx)

	triggersJSON, err := json.Marshal(event.Triggers)
	if err != nil {
		return fmt.Errorf("failed to marshal triggers: %w", err)
	}

	if event.State == "" {
		event.State = "active"
	}

	uniqueKey := r.getUniqueKey(event.Name, event.ServiceName, event.TypeEvent.String(), event.State)

	query := `
		INSERT INTO events (id, unique_key, name, service_name, repo_url, team_owner, type_event, state, triggers, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	now := time.Now()
	_, err = r.db.ExecContext(ctx, query,
		uuid.New(),
		uniqueKey,
		event.Name,
		event.ServiceName,
		event.RepoURL,
		event.TeamOwner,
		event.TypeEvent.String(),
		event.State,
		triggersJSON,
		now,
		now,
	)

	if err != nil {
		l.Error("Error on create internal event", "error", err)
		return fmt.Errorf("failed to create internal event: %w", err)
	}

	return nil
}

func (r *PostgresStore) getEventByUniqueKey(ctx context.Context, uniqueKey string) (domain.Event, error) {
	l := ctxlogger.GetLogger(ctx)

	query := `
		SELECT name, service_name, repo_url, team_owner, triggers 
		FROM events 
		WHERE unique_key = $1 AND deleted_at IS NULL
	`

	var event domain.Event
	var triggersJSON []byte

	err := r.db.QueryRowContext(ctx, query, uniqueKey).Scan(
		&event.Name,
		&event.ServiceName,
		&event.RepoURL,
		&event.TeamOwner,
		&triggersJSON,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			l.Warn("No documents found", "unique key", uniqueKey)
			return domain.Event{}, domain.EventNotFound
		}
		return domain.Event{}, fmt.Errorf("failed to get internal event: %w", err)
	}

	if err := json.Unmarshal(triggersJSON, &event.Triggers); err != nil {
		return domain.Event{}, fmt.Errorf("failed to unmarshal triggers: %w", err)
	}

	return event, nil
}

func (r *PostgresStore) getUniqueKey(eventName, serviceName, eventType, state string) string {
	return fmt.Sprintf("%s:%s:%s:%s", eventName, serviceName, eventType, state)
}
