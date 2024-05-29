BEGIN;

DROP TABLE IF EXISTS notification_channels;
DROP TABLE IF EXISTS notification_recipients;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS distribution_lists;
DROP TABLE IF EXISTS user_notifications;
DROP TABLE IF EXISTS user_config;

DROP INDEX IF EXISTS distribution_lists_idx;
DROP INDEX IF EXISTS notifications_idx;
DROP INDEX IF EXISTS user_notifications_idx;
DROP INDEX IF EXISTS notification_recipients_idx;

DROP TYPE IF EXISTS notification_priority;
DROP TYPE IF EXISTS notification_channel;

COMMIT;
