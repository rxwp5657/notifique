package storage

import (
	"time"

	"github.com/notifique/dto"
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

const GET_USER_CONFIG = `
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

const INSERT_USER_CONFIG = `
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

const UPSERT_USER_CONFIG = `
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
