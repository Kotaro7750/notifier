package notification

import (
	"log/slog"
)

type Notification struct {
	Title    string     `json:"title"`
	Severity slog.Level `json:"severity"`
	Message  string     `json:"message"`
}
