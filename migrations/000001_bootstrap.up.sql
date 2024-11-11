BEGIN;

CREATE TYPE notification_priority AS ENUM (
    'HIGH',
    'MEDIUM',
    'LOW'
);

CREATE TYPE notification_channel AS ENUM (
    'in-app',
    'e-mail',
    'sms',
    'push'
);

CREATE TYPE notification_status AS ENUM (
    'CREATED',
    'PUBLISHED',
    'PUBLISHED_FAILED',
    'PROCESSING',
    'PROCESSING_FAILED',
    'SENT'
);

CREATE TABLE IF NOT EXISTS distribution_lists (
    "name" VARCHAR PRIMARY KEY,
    num_recipients INT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS distribution_list_recipients (
    "name" VARCHAR,
    recipient VARCHAR NOT NULL,
    CONSTRAINT distribution_lists_pk
        PRIMARY KEY("name", recipient),
    CONSTRAINT distribution_list_fk
        FOREIGN KEY("name")
        REFERENCES distribution_lists("name")
        ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS distribution_list_recipients_idx
ON distribution_list_recipients("name", recipient);

CREATE TABLE IF NOT EXISTS notifications (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR NOT NULL,
    contents VARCHAR NOT NULL,
    image_url VARCHAR,
    topic VARCHAR NOT NULL,
    "priority" notification_priority DEFAULT 'LOW',
    distribution_list VARCHAR,
    created_at TIMESTAMPTZ NOT NULL,
    "status" notification_status NOT NULL,
    CONSTRAINT distribution_list_fk
        FOREIGN KEY (distribution_list)
        REFERENCES distribution_lists("name")
        ON DELETE SET NULL (distribution_list)
);

CREATE INDEX IF NOT EXISTS notifications_idx
ON notifications(id, created_at);

CREATE TABLE IF NOT EXISTS notification_status_log(
    notification_id uuid NOT NULL,
    status_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "status" notification_status NOT NULL,
    error_message VARCHAR,
    CONSTRAINT notification_id_fk
        FOREIGN KEY (notification_id)
        REFERENCES notifications(id),
    CONSTRAINT notification_status_log_pk
        PRIMARY KEY(notification_id, status_date)
);

CREATE INDEX IF NOT EXISTS notification_status_idx
ON notification_status_log(notification_id, status_date);

CREATE TABLE IF NOT EXISTS user_notifications (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR NOT NULL,
    title VARCHAR NOT NULL,
    contents VARCHAR NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    image_url VARCHAR,
    read_at TIMESTAMPTZ,
    topic VARCHAR
);

CREATE INDEX IF NOT EXISTS user_notifications_idx
ON user_notifications(id, user_id, created_at);

CREATE TABLE IF NOT EXISTS notification_recipients (
    notification_id uuid NOT NULL,
    recipient VARCHAR NOT NULL,
    CONSTRAINT notification_id_fk
        FOREIGN KEY (notification_id)
        REFERENCES notifications(id)
);

CREATE INDEX IF NOT EXISTS notification_recipients_idx
ON notification_recipients(notification_id, recipient);

CREATE TABLE IF NOT EXISTS notification_channels (
    notification_id uuid NOT NULL,
    channel notification_channel NOT NULL
);

CREATE TABLE IF NOT EXISTS user_config (
    user_id VARCHAR PRIMARY KEY,
    email_opt_in BOOLEAN NOT NULL DEFAULT TRUE,
    email_snooze_until TIMESTAMPTZ,
    sms_opt_in BOOLEAN NOT NULL DEFAULT TRUE,
    sms_snooze_until TIMESTAMPTZ,
    in_app_opt_in BOOLEAN NOT NULL DEFAULT TRUE,
    in_app_snooze_until TIMESTAMPTZ,
    push_opt_in BOOLEAN NOT NULL DEFAULT TRUE,
    push_snooze_until TIMESTAMPTZ
);

COMMIT;
