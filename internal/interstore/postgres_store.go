package interstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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

var _ Repository = (*PostgresStore)(nil)

func (r *PostgresStore) GetInternalEvent(ctx context.Context, eventName, serviceName string, state string) (domain.Event, error) {
	l := ctxlogger.GetLogger(ctx)

	uniqueKey := r.getUniqueKey(eventName, serviceName, state)

	query := `
			SELECT id, name, service_name, repo_url, team_owner, triggers
			FROM events
			WHERE unique_key = $1 AND deleted_at IS NULL
		`
	var event domain.Event
	var triggersJSON []byte
	err := r.db.QueryRowContext(ctx, query, uniqueKey).Scan(
		&event.ID,
		&event.Name,
		&event.ServiceName,
		&triggersJSON,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			l.Warn("No documents found", "unique key", uniqueKey)
			return domain.Event{}, domain.EventNotFound
		}
		l.Error("Error on get internal event by unique key", "error", err)
		return domain.Event{}, fmt.Errorf("failed to get internal event: %w", err)
	}

	if err := json.Unmarshal(triggersJSON, &event.Triggers); err != nil {
		return domain.Event{}, fmt.Errorf("failed to unmarshal triggers: %w", err)
	}

	return event, nil
}

func (r *PostgresStore) GetInternalEvents(ctx context.Context, filters domain.FilterEvents) ([]domain.Event, error) {
	var sqlFilter string

	if len(filters.State) > 0 {
		sqlFilter += fmt.Sprintf("state IN ('%s')", strings.Join(filters.State, "', '"))
	}

	if len(filters.TeamOwner) > 0 {
		sqlFilter += fmt.Sprintf("AND team_owner IN ('%s')", strings.Join(filters.TeamOwner, "', '"))
	}

	if len(filters.ServiceName) > 0 {
		sqlFilter += fmt.Sprintf("AND service_name IN ('%s')", strings.Join(filters.ServiceName, "', '"))
	}

	query := fmt.Sprintf(`SELECT %s FROM events WHERE %s LIMIT %d OFFSET %d`, modelEventFields, sqlFilter, filters.Limit, filters.Page-1)

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}

	defer rows.Close()

	events := make([]domain.Event, 0)
	for rows.Next() {
		var event ModelEvent
		if err := rows.Scan(
			&event.ID,
			&event.Name,
			&event.ServiceName,
			&event.State,
			&event.Triggers,
		); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		events = append(events, event.ToDomain())
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate over events: %w", err)
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

	uniqueKey := r.getUniqueKey(event.Name, event.ServiceName, event.State)

	query := `
		INSERT INTO events (id, unique_key, name, service_name, state, triggers, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	now := time.Now()
	_, err = r.db.ExecContext(ctx, query,
		uuid.New(),
		uniqueKey,
		event.Name,
		event.ServiceName,
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

const modelEventFields = `
	id,
	name,
	service_name,
	state,
	triggers
`

func (r *PostgresStore) GetEventByID(ctx context.Context, eventID uuid.UUID) (domain.Event, error) {
	l := ctxlogger.GetLogger(ctx)

	query := fmt.Sprintf(`SELECT %s FROM events WHERE id = $1 AND deleted_at IS NULL`, modelEventFields)

	var event ModelEvent
	err := r.db.QueryRowContext(ctx, query, eventID).Scan(
		&event.ID,
		&event.Name,
		&event.ServiceName,
		&event.State,
		&event.Triggers,
	)

	if errors.Is(err, sql.ErrNoRows) {
		l.Warn("Not found event", "tag", "PostgresStore.GetEventByID")
		return domain.Event{}, domain.EventNotFound
	}

	if err != nil {
		l.Error("Error on get event by id", "tag", "PostgresStore.GetEventByID", "error", err)
		return domain.Event{}, fmt.Errorf("failed to get event by id: %w", err)
	}

	return event.ToDomain(), nil
}

// State: archived | active
func (r *PostgresStore) GetAllSchedulers(ctx context.Context, state string) ([]domain.Event, error) {
	l := ctxlogger.GetLogger(ctx)

	query := fmt.Sprintf(`SELECT %s FROM events WHERE state = $1 AND deleted_at IS NULL`, modelEventFields)

	rows, err := r.db.QueryContext(ctx, query, state)
	if errors.Is(err, sql.ErrNoRows) {
		l.Warn("Not found schedulers", "tag", "PostgresStore.GetAllSchedulers")
		return nil, domain.EventNotFound
	}

	if err != nil {
		l.Error("Error on get all schedulers", "tag", "PostgresStore.GetAllSchedulers", "error", err)
		return nil, fmt.Errorf("failed to get all schedulers: %w", err)
	}

	defer rows.Close()

	var events []domain.Event
	for rows.Next() {
		var event ModelEvent
		if err := rows.Scan(
			&event.ID,
			&event.Name,
			&event.ServiceName,
			&event.State,
			&event.Triggers,
		); err != nil {
			l.Error("Error on scan row", "error", err)
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		events = append(events, event.ToDomain())
	}

	if err := rows.Err(); err != nil {
		l.Error("Error on get all schedulers", "error", err)
		return nil, fmt.Errorf("failed to get all schedulers: %w", err)
	}

	return events, nil
}

func (r *PostgresStore) DisabledEvent(ctx context.Context, eventID uuid.UUID) error {
	query := `UPDATE events SET state = 'disabled', unique_key = $2, deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL;`
	_, err := r.db.Exec(query, eventID, fmt.Sprintf("disabled.%s", uuid.New().String()))
	if err != nil {
		return fmt.Errorf("failed to disable event: %w", err)
	}

	return nil
}

func (r *PostgresStore) UpdateEvent(ctx context.Context, event domain.Event) error {
	query := `
	UPDATE
	events SET name = $2,
	service_name = $3,
	state = $7,
	triggers = $8
	WHERE id = $1 AND deleted_at IS NULL;`

	triggersJSON, err := json.Marshal(event.Triggers)
	if err != nil {
		return fmt.Errorf("failed to marshal triggers: %w", err)
	}

	if _, err := r.db.ExecContext(ctx, query, event.ID, event.Name, event.ServiceName, event.State, triggersJSON); err != nil {
		return fmt.Errorf("failed to update event: %w", err)
	}

	return nil
}

func (r *PostgresStore) getUniqueKey(eventName, serviceName, state string) string {
	return fmt.Sprintf("%s:%s:%s", eventName, serviceName, state)
}
