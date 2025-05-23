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
    'QUEUED',
    'SENDING',
    'SENT',
    'FAILED',
    'CANCELED'
);

CREATE TYPE template_variable_type AS ENUM (
    'STRING',
    'NUMBER',
    'DATE',
    'DATETIME'
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

CREATE TABLE IF NOT EXISTS notification_templates (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    "name" VARCHAR NOT NULL,
    is_html BOOLEAN NOT NULL DEFAULT FALSE,
    title_template VARCHAR NOT NULL,
    contents_template VARCHAR NOT NULL,
    "description" VARCHAR NOT NULL,
    created_by VARCHAR NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (NOW() at time zone 'utc'),
    updated_by VARCHAR,
    updated_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS notification_template_variables (
    template_id uuid NOT NULL,
    "name" VARCHAR NOT NULL,
    "type" template_variable_type NOT NULL,
    "required" BOOLEAN NOT NULL,
    "validation" VARCHAR,
    CONSTRAINT template_variable_pk
        PRIMARY KEY(template_id, "name"),
    CONSTRAINT template_id_fk
        FOREIGN KEY (template_id)
        REFERENCES notification_templates(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS notifications (
    id uuid PRIMARY KEY,
    title VARCHAR,
    contents VARCHAR,
    template_id uuid,
    image_url VARCHAR,
    topic VARCHAR NOT NULL,
    "priority" notification_priority DEFAULT 'LOW',
    distribution_list VARCHAR,
    created_at TIMESTAMPTZ NOT NULL,
    created_by VARCHAR NOT NULL,
    "status" notification_status NOT NULL,
    CONSTRAINT distribution_list_fk
        FOREIGN KEY (distribution_list)
        REFERENCES distribution_lists("name")
        ON DELETE SET NULL (distribution_list),
    CONSTRAINT template_id_fk
        FOREIGN KEY (template_id)
        REFERENCES notification_templates(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS notifications_idx
ON notifications(id, created_at);

CREATE TABLE IF NOT EXISTS notification_template_variable_contents (
    notification_id uuid NOT NULL,
    "name" VARCHAR NOT NULL,
    "value" VARCHAR NOT NULL,
    CONSTRAINT notification_id_fk
        FOREIGN KEY (notification_id)
        REFERENCES notifications(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS notification_status_log(
    notification_id uuid NOT NULL,
    status_date TIMESTAMPTZ NOT NULL DEFAULT (NOW() at time zone 'utc'),
    "status" notification_status NOT NULL,
    error_message VARCHAR,
    CONSTRAINT notification_id_fk
        FOREIGN KEY (notification_id)
        REFERENCES notifications(id) ON DELETE CASCADE,
    CONSTRAINT notification_status_log_pk
        PRIMARY KEY(notification_id, status_date)
);

CREATE INDEX IF NOT EXISTS notification_status_idx
ON notification_status_log(notification_id, status_date);

CREATE TABLE IF NOT EXISTS recipient_notification_status_log(
    notification_id uuid NOT NULL,
    user_id VARCHAR NOT NULL,
    channel notification_channel NOT NULL,
    status_date TIMESTAMPTZ NOT NULL DEFAULT (NOW() at time zone 'utc'),
    "status" notification_status NOT NULL,
    error_message VARCHAR,
    CONSTRAINT notification_id_fk
        FOREIGN KEY (notification_id)
        REFERENCES notifications(id) ON DELETE CASCADE,
    CONSTRAINT notification_user_status_log_pk
        PRIMARY KEY(notification_id, user_id, channel, status_date)
);

CREATE INDEX IF NOT EXISTS recipient_notification_status_idx
ON recipient_notification_status_log(notification_id, user_id, channel, status_date);

CREATE TABLE IF NOT EXISTS user_notifications (
    id uuid PRIMARY KEY,
    user_id VARCHAR NOT NULL,
    title VARCHAR NOT NULL,
    contents VARCHAR NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    image_url VARCHAR,
    read_at TIMESTAMPTZ,
    topic VARCHAR
);

CREATE UNIQUE INDEX IF NOT EXISTS user_notifications_idx
ON user_notifications(id, user_id, created_at);

CREATE TABLE IF NOT EXISTS notification_recipients (
    notification_id uuid NOT NULL,
    recipient VARCHAR NOT NULL,
    CONSTRAINT notification_id_fk
        FOREIGN KEY (notification_id)
        REFERENCES notifications(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS notification_recipients_idx
ON notification_recipients(notification_id, recipient);

CREATE TABLE IF NOT EXISTS notification_channels (
    notification_id uuid NOT NULL,
    channel notification_channel NOT NULL,
    CONSTRAINT notification_id_fk
        FOREIGN KEY (notification_id)
        REFERENCES notifications(id) ON DELETE CASCADE
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
