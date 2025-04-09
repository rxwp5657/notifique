package auth

type Scope string
type Header string

const (
	UserNotificationPublisher Scope  = "user_notifications/publisher"
	User                      Scope  = "notifications/user"
	Admin                     Scope  = "notifications/admin"
	NotificationsPublisher    Scope  = "notifications/publisher"
	UserHeader                Header = "X-User-Id"
	ScopeHeader               Header = "X-User-Scope"
)
