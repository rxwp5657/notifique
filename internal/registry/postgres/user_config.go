package postgresresgistry

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/notifique/internal/server/dto"
)

type userConfig struct {
	EmailOptIn       bool       `db:"email_opt_in"`
	EmailSnoozeUntil *time.Time `db:"email_snooze_until"`
	SMSOptIn         bool       `db:"sms_opt_in"`
	smsSoozeUntil    *time.Time `db:"sms_snooze_until"`
	InAppOptIn       bool       `db:"in_app_opt_in"`
	InAppSnoozeUntil *time.Time `db:"in_app_snooze_until"`
	PushOptIn        bool       `db:"push_opt_in"`
	PushSnoozeUntil  *time.Time `db:"push_snooze_until"`
}

func (cf *userConfig) toDTO() dto.UserConfig {

	toStr := func(t *time.Time) *string {
		if t == nil {
			return nil
		}

		str := t.Format(time.RFC3339Nano)
		return &str
	}

	return dto.UserConfig{
		EmailConfig: dto.ChannelConfig{
			OptIn:       cf.EmailOptIn,
			SnoozeUntil: toStr(cf.EmailSnoozeUntil),
		},
		SMSConfig: dto.ChannelConfig{
			OptIn:       cf.SMSOptIn,
			SnoozeUntil: toStr(cf.smsSoozeUntil),
		},
		InAppConfig: dto.ChannelConfig{
			OptIn:       cf.InAppOptIn,
			SnoozeUntil: toStr(cf.InAppSnoozeUntil),
		},
	}
}

const GetUserConfig = `
SELECT
	email_opt_in,
	email_snooze_until,
	sms_opt_in,
	sms_snooze_until,
	in_app_opt_in,
	in_app_snooze_until,
	push_opt_in,
	push_snooze_until
FROM
	user_config
WHERE
	user_id = @userId;
`

const InsertUserConfig = `
INSERT INTO user_config (
	user_id,
	email_opt_in,
	email_snooze_until,
	sms_opt_in,
	sms_snooze_until,
	in_app_opt_in,
	in_app_snooze_until,
	push_opt_in,
	push_snooze_until
) VALUES (
	@userId,
	@emailOptIn,
	@emailSnoozeUntil,
	@smsOptIn,
	@smsSoozeUntil,
	@inAppOptIn,
	@inAppSnoozeUntil,
	@pushOptIn,
	@pushSnoozeUntil
);
`

const UpsertUserConfig = `
INSERT INTO user_config (
	user_id,
	email_opt_in,
	email_snooze_until,
	sms_opt_in,
	sms_snooze_until,
	in_app_opt_in,
	in_app_snooze_until,
	push_opt_in,
	push_snooze_until
) VALUES (
	@userId,
	@emailOptIn,
	@emailSnoozeUntil,
	@smsOptIn,
	@smsSoozeUntil,
	@inAppOptIn,
	@inAppSnoozeUntil,
	@pushOptIn,
	@pushSnoozeUntil
) ON CONFLICT
	(user_id)
DO UPDATE SET
	email_opt_in = EXCLUDED.email_opt_in,
	email_snooze_until = EXCLUDED.email_snooze_until,
	sms_opt_in = EXCLUDED.sms_opt_in,
	sms_snooze_until = EXCLUDED.sms_snooze_until,
	in_app_opt_in = EXCLUDED.in_app_opt_in,
	in_app_snooze_until = EXCLUDED.in_app_snooze_until,
	push_opt_in = EXCLUDED.push_opt_in,
	push_snooze_until = EXCLUDED.push_snooze_until;
`

func (ps *Registry) makeUserConfig(ctx context.Context, userId string) (*userConfig, error) {

	tx, err := ps.conn.Begin(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to start transaction - %w", err)
	}

	cfg := userConfig{
		EmailOptIn: true,
		SMSOptIn:   true,
		PushOptIn:  true,
		InAppOptIn: true,
	}

	args := pgx.NamedArgs{
		"userId":           userId,
		"emailOptIn":       cfg.EmailOptIn,
		"emailSnoozeUntil": cfg.EmailSnoozeUntil,
		"smsOptIn":         cfg.SMSOptIn,
		"smsSoozeUntil":    cfg.smsSoozeUntil,
		"inAppOptIn":       cfg.InAppOptIn,
		"inAppSnoozeUntil": cfg.InAppSnoozeUntil,
		"pushOptIn":        cfg.PushOptIn,
		"pushSnoozeUntil":  cfg.PushSnoozeUntil,
	}

	_, err = tx.Exec(ctx, InsertUserConfig, args)

	if err != nil {
		tx.Rollback(ctx)
		return nil, fmt.Errorf("failed to inser user config - %w", err)
	}

	err = tx.Commit(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to commit user config - %w", err)
	}

	return &cfg, nil
}

func (ps *Registry) GetUserConfig(ctx context.Context, userId string) (dto.UserConfig, error) {

	args := pgx.NamedArgs{"userId": userId}

	var cfg userConfig

	err := ps.conn.QueryRow(ctx, GetUserConfig, args).Scan(
		&cfg.EmailOptIn,
		&cfg.EmailSnoozeUntil,
		&cfg.SMSOptIn,
		&cfg.smsSoozeUntil,
		&cfg.InAppOptIn,
		&cfg.InAppSnoozeUntil,
		&cfg.PushOptIn,
		&cfg.PushSnoozeUntil,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			newCfg, err := ps.makeUserConfig(ctx, userId)

			if err != nil {
				return dto.UserConfig{}, err
			}

			cfg = *newCfg
		}
	}

	return cfg.toDTO(), nil
}

func (ps *Registry) UpdateUserConfig(ctx context.Context, userId string, config dto.UserConfig) error {

	tx, err := ps.conn.Begin(ctx)

	if err != nil {
		return fmt.Errorf("failed to start transaction - %w", err)
	}

	args := pgx.NamedArgs{
		"userId":           userId,
		"emailOptIn":       config.EmailConfig.OptIn,
		"emailSnoozeUntil": config.EmailConfig.SnoozeUntil,
		"smsOptIn":         config.SMSConfig.OptIn,
		"smsSoozeUntil":    config.SMSConfig.SnoozeUntil,
		"inAppOptIn":       config.InAppConfig.OptIn,
		"inAppSnoozeUntil": config.InAppConfig.SnoozeUntil,
		"pushOptIn":        true,
		"pushSnoozeUntil":  nil,
	}

	_, err = tx.Exec(ctx, UpsertUserConfig, args)

	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("failed to upsert user config - %w", err)
	}

	err = tx.Commit(ctx)

	if err != nil {
		return fmt.Errorf("failed to commit user config update - %w", err)
	}

	return nil
}
