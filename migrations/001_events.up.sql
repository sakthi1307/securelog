CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS events (
  id          uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
  ts          timestamptz NOT NULL,
  type        text NOT NULL,
  severity    text NOT NULL,
  src_ip      inet,
  host        text,
  username    text,
  msg         text,
  raw         jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_events_ts ON events (ts DESC);
CREATE INDEX IF NOT EXISTS idx_events_type_ts ON events (type, ts DESC);
CREATE INDEX IF NOT EXISTS idx_events_srcip_ts ON events (src_ip, ts DESC);

