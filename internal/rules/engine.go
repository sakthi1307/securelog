package rules

import (
	"context"
	"net"
	"time"
	"log/slog"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sakthi1307/securelog/internal/store"
)

type RuleEvalMsg struct {
	EventType string
	SrcIP     net.IP
	Ts        time.Time
}

type Engine struct {
	DB     *pgxpool.Pool
	Alerts *store.AlertStore
	Queue  <-chan RuleEvalMsg
}

const (
	ruleName          = "login_failed_spike"
	threshold         = 5
	window            = 2 * time.Minute
	alertSeverity     = "high"
	targetEventType   = "login_failed"
)

func (e *Engine) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-e.Queue:
			if !ok {
				return
			}
			e.handle(ctx, msg)
		}
	}
}

func (e *Engine) handle(ctx context.Context, msg RuleEvalMsg) {
	if msg.EventType != targetEventType || msg.SrcIP == nil {
		return
	}

	// Count events in last 2 minutes for this IP
	start := msg.Ts.Add(-window)
	var c int
	qctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	err := e.DB.QueryRow(qctx, `
		SELECT count(*)
		FROM events
		WHERE type = $1
		  AND src_ip = $2::inet
		  AND ts >= $3
		  AND ts <= $4
	`, targetEventType, msg.SrcIP.String(), start, msg.Ts).Scan(&c)
	if err != nil {
		slog.Error("rule_count_query_failed", "err", err)
		return
	}

	if c < threshold {
		return
	}

	fingerprint := ruleName + ":" + msg.SrcIP.String()
	details := map[string]any{
		"src_ip": msg.SrcIP.String(),
		"type":   targetEventType,
		"count":  c,
		"window_seconds": int(window.Seconds()),
	}

	id, err := e.Alerts.UpsertOpen(qctx, ruleName, fingerprint, msg.SrcIP, alertSeverity, start, msg.Ts, c, details)
	if err != nil {
		slog.Error("alert_upsert_failed", "err", err)
		return
	}
	slog.Info("alert_upsert_ok", "id", id, "fingerprint", fingerprint)
}
