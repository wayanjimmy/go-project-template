-- migrate:up
CREATE TABLE IF NOT EXISTS instances (
  id BIGSERIAL NOT NULL PRIMARY KEY,
  instance_id VARCHAR(128) NOT NULL,
  execution_id VARCHAR(128) NOT NULL,
  parent_instance_id VARCHAR(128) NULL,
  parent_execution_id VARCHAR(128) NULL,
  parent_schedule_event_id NUMERIC NULL,
  metadata BYTEA NULL,
  state INT NOT NULL,
  queue VARCHAR(128) DEFAULT '' NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  completed_at TIMESTAMPTZ NULL,
  locked_until TIMESTAMPTZ NULL,
  sticky_until TIMESTAMPTZ NULL,
  worker VARCHAR(64) NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_instances_instance_id_execution_id ON instances (instance_id, execution_id);
CREATE INDEX IF NOT EXISTS idx_instances_locked_until_completed_at_queue ON instances (completed_at, locked_until, sticky_until, worker, queue);
CREATE INDEX IF NOT EXISTS idx_instances_parent_instance_id_parent_execution_id ON instances (parent_instance_id, parent_execution_id);

CREATE TABLE IF NOT EXISTS pending_events (
  id BIGSERIAL NOT NULL PRIMARY KEY,
  event_id VARCHAR(128) NOT NULL,
  sequence_id BIGSERIAL NOT NULL,
  instance_id VARCHAR(128) NOT NULL,
  execution_id VARCHAR(128) NOT NULL,
  event_type INT NOT NULL,
  timestamp TIMESTAMPTZ NOT NULL,
  schedule_event_id BIGSERIAL NOT NULL,
  visible_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_pending_events_inid_exid ON pending_events (instance_id, execution_id);
CREATE INDEX IF NOT EXISTS idx_pending_events_inid_exid_visible_at_schedule_event_id ON pending_events (instance_id, execution_id, visible_at, schedule_event_id);

CREATE TABLE IF NOT EXISTS history (
  id BIGSERIAL NOT NULL PRIMARY KEY,
  event_id VARCHAR(128) NOT NULL,
  sequence_id BIGSERIAL NOT NULL,
  instance_id VARCHAR(128) NOT NULL,
  execution_id VARCHAR(128) NOT NULL,
  event_type INT NOT NULL,
  timestamp TIMESTAMPTZ NOT NULL,
  schedule_event_id BIGSERIAL NOT NULL,
  visible_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_history_instance_id_execution_id ON history (instance_id, execution_id);
CREATE INDEX IF NOT EXISTS idx_history_instance_id_execution_id_sequence_id ON history (instance_id, execution_id, sequence_id);

CREATE TABLE IF NOT EXISTS activities (
  id BIGSERIAL NOT NULL PRIMARY KEY,
  activity_id VARCHAR(128) NOT NULL,
  instance_id VARCHAR(128) NOT NULL,
  execution_id VARCHAR(128) NOT NULL,
  event_type INT NOT NULL,
  queue VARCHAR(128) DEFAULT '' NOT NULL,
  timestamp TIMESTAMPTZ NOT NULL,
  schedule_event_id BIGSERIAL NOT NULL,
  visible_at TIMESTAMPTZ NULL,
  locked_until TIMESTAMPTZ NULL,
  worker VARCHAR(64) NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_activities_instance_id_execution_id_activity_id_worker ON activities (instance_id, execution_id, activity_id, worker);
CREATE INDEX IF NOT EXISTS idx_activities_locked_until_queue ON activities (locked_until, queue);

CREATE TABLE IF NOT EXISTS attributes (
  id BIGSERIAL NOT NULL PRIMARY KEY,
  event_id VARCHAR(128) NOT NULL,
  instance_id VARCHAR(128) NOT NULL,
  execution_id VARCHAR(128) NOT NULL,
  data BYTEA NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_attributes_instance_id_execution_id_event_id ON attributes (instance_id, execution_id, event_id);
CREATE INDEX IF NOT EXISTS idx_attributes_event_id ON attributes (event_id);

-- migrate:down
DROP TABLE IF EXISTS attributes;
DROP TABLE IF EXISTS activities;
DROP TABLE IF EXISTS history;
DROP TABLE IF EXISTS pending_events;
DROP TABLE IF EXISTS instances;
