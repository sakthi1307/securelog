package store

import (
	"context"
	"encoding/json"
	"net"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sakthi1307/securelog/internal/models"
)

type EventStore struct {
	DB *pgxpool.Pool
}

type InsertedEvent struct {
	ID string
}

func NewEventStore(db *pgxpool.Pool) *EventStore {
	return &EventStore{DB: db}
}

func (s *EventStore) Insert(ctx context.Context, e models.EventIngestDTO, srcIP net.IP) (InsertedEvent, error) {
	raw := e.Raw
	if len(raw) == 0 {
		raw = json.RawMessage([]byte(`{}`))
	}

	var ip any = nil
	if srcIP != nil {
		ip = srcIP.String()
	}

	var id string
	err := s.DB.QueryRow(ctx, `
		INSERT INTO events (ts, type, severity, src_ip, host, username, msg, raw)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id
	`, e.Ts, e.Type, string(e.Severity), ip, e.Host, e.Username, e.Msg, raw).Scan(&id)

	return InsertedEvent{ID: id}, err
}

func withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, 2*time.Second)
}
