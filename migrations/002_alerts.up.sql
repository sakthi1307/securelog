CREATE TABLE IF NOT EXISTS alerts (
  id           uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
  rule_name    text NOT NULL,
  fingerprint  text NOT NULL, -- e.g., "login_failed_spike:1.2.3.4"
  src_ip       inet,
  severity     text NOT NULL,
  state        text NOT NULL DEFAULT 'open', -- open|ack|resolved
  first_seen   timestamptz NOT NULL,
  last_seen    timestamptz NOT NULL,
  count        integer NOT NULL DEFAULT 0,
  details      jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now()
);

-- Dedupe: only one OPEN alert per fingerprint
CREATE UNIQUE INDEX IF NOT EXISTS uniq_open_alert_per_fingerprint
ON alerts (fingerprint)
WHERE state = 'open';

CREATE INDEX IF NOT EXISTS idx_alerts_state_lastseen
ON alerts (state, last_seen DESC);
