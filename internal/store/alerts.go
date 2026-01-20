package store

import (
	"context"
	"encoding/json"
	"net"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AlertStore struct {
	DB *pgxpool.Pool
}

func NewAlertStore(db *pgxpool.Pool) *AlertStore {
	if db == nil {
		panic("AlertStore: db is nil")
	}
	return &AlertStore{DB: db}
}

type AlertRow struct {
	ID          string    `json:"id"`
	RuleName    string    `json:"rule_name"`
	Fingerprint string    `json:"fingerprint"`
	SrcIP       *string   `json:"src_ip,omitempty"`
	Severity    string    `json:"severity"`
	State       string    `json:"state"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
	Count       int       `json:"count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Details     any       `json:"details"`
}

// Upsert open alert by fingerprint (dedupes using partial unique index)
func (s *AlertStore) UpsertOpen(ctx context.Context, ruleName, fingerprint string, srcIP net.IP, severity string, firstSeen, lastSeen time.Time, count int, details any) (string, error) {
	var ip any = nil
	if srcIP != nil {
		ip = srcIP.String()
	}

	b, _ := json.Marshal(details)
	var id string
	err := s.DB.QueryRow(ctx, `
	INSERT INTO alerts (rule_name, fingerprint, src_ip, severity, state, first_seen, last_seen, count, details)
	VALUES ($1,$2,$3,$4,'open',$5,$6,$7,$8)
	ON CONFLICT (fingerprint) WHERE state = 'open'
	DO UPDATE SET
		last_seen = EXCLUDED.last_seen,
		count = EXCLUDED.count,
		updated_at = now(),
		details = EXCLUDED.details
	RETURNING id
	`, ruleName, fingerprint, ip, severity, firstSeen, lastSeen, count, b).Scan(&id)


	return id, err
}

func (s *AlertStore) ListOpen(ctx context.Context, limit int) ([]AlertRow, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.DB.Query(ctx, `
		SELECT id, rule_name, fingerprint, src_ip::text, severity, state,
		       first_seen, last_seen, count, created_at, updated_at, details
		FROM alerts
		WHERE state = 'open'
		ORDER BY last_seen DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]AlertRow, 0)
	for rows.Next() {
		var a AlertRow
		var srcIP *string
		var detailsBytes []byte
		if err := rows.Scan(&a.ID, &a.RuleName, &a.Fingerprint, &srcIP, &a.Severity, &a.State,
			&a.FirstSeen, &a.LastSeen, &a.Count, &a.CreatedAt, &a.UpdatedAt, &detailsBytes); err != nil {
			return nil, err
		}
		a.SrcIP = srcIP
		var details any
		_ = json.Unmarshal(detailsBytes, &details)
		a.Details = details
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *AlertStore) Ack(ctx context.Context, id string) error {
	_, err := s.DB.Exec(ctx, `
		UPDATE alerts SET state='ack', updated_at=now()
		WHERE id = $1 AND state='open'
	`, id)
	return err
}
