package abstraction

import (
	"log/slog"

	"github.com/Kotaro7750/notifier/notification"
	"gopkg.in/yaml.v3"
)

type AbstractChannelComponentBuilder func(id string, properties yaml.Node) (AbstractChannelComponent, error)

// AbstractChannelComponent is a single execution unit managed by AutonomousChannelComponent.
// Implementations process notifications until they stop by error or done, and report that
// result through the channel returned by Start. They do not own restart or supervision logic.
type AbstractChannelComponent interface {
	GetId() string
	GetLogger() *slog.Logger
	SetLogger(logger *slog.Logger)
	// Start begins one execution of the component.
	// ch is the component's notification channel: receivers write notifications to it and
	// senders read notifications from it.
	// done is closed by the supervisor to request shutdown. Implementations should stop
	// accepting new work, finish their own shutdown processing, and then exit.
	// The returned channel reports the end of this execution. Send one non-nil error when
	// the execution ends by failure, or close the channel without sending when it ends
	// normally. Implementations should then return and must not restart themselves.
	Start(ch chan notification.Notification, done <-chan struct{}) <-chan error
}
