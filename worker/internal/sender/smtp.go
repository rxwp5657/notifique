package sender

import (
	"context"
	"fmt"
	"net/smtp"

	"github.com/notifique/shared/dto"
)

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

type SMTPConfigurator interface {
	GetSMTPConfig() (SMTPConfig, error)
}

type SMTP struct {
	cfg SMTPConfig
}

func NewSMTP(cfg SMTPConfig) *SMTP {
	return &SMTP{
		cfg: cfg,
	}
}

func (s *SMTP) SendNotifications(ctx context.Context, batch []dto.UserEmailNotificationReq) error {

	auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	for _, notification := range batch {
		contentType := "text/plain"

		if notification.IsHtml {
			contentType = "text/html"
		}

		msg := fmt.Appendf(nil, "From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"Content-Type: %s; charset=UTF-8\r\n"+
			"\r\n"+
			"%s\r\n",
			s.cfg.From,
			notification.Email,
			notification.Title,
			contentType,
			notification.Contents)

		err := smtp.SendMail(addr, auth, s.cfg.From, []string{notification.Email}, msg)

		if err != nil {
			return fmt.Errorf("failed to send email to %s: %w", notification.Email, err)
		}
	}

	return nil
}
