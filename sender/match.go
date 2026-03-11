package sender

import (
	"slices"

	"github.com/Kotaro7750/notifier/config"
	"github.com/Kotaro7750/notifier/notification"
)

type MatchCondition struct {
	NotificationSource []string
	Labels             map[string][]string
}

func NewMatchCondition(match config.MetadataCondition) MatchCondition {
	condition := MatchCondition{
		NotificationSource: config.NormalizeCSVValues(match.NotificationSource),
	}

	if len(match.Labels) == 0 {
		return condition
	}

	condition.Labels = make(map[string][]string, len(match.Labels))
	for key, value := range match.Labels {
		condition.Labels[key] = config.NormalizeCSVValues(value)
	}

	return condition
}

func (m MatchCondition) hasConditions() bool {
	return len(m.NotificationSource) > 0 || len(m.Labels) > 0
}

func (m MatchCondition) IsMatched(n notification.Notification) bool {
	// When no conditions are set, all notifications match
	if !m.hasConditions() {
		return true
	}

	if len(m.NotificationSource) > 0 && !slices.Contains(m.NotificationSource, n.NotificationSource) {
		return false
	}

	for key, values := range m.Labels {
		// When notification doesn't have the label specified in the condition, it doesn't match
		valueInNotification, ok := n.Labels[key]
		if !ok {
			return false
		}

		// When the condition specifies values for the label, the notification's label value must match one of them
		if !slices.Contains(values, valueInNotification) {
			return false
		}
	}

	return true
}
