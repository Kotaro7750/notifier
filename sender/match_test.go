package sender

import (
	"testing"

	"github.com/Kotaro7750/notifier/notification"
)

func TestMatchConditionIsMatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		match        MatchCondition
		notification notification.Notification
		want         bool
	}{
		{
			name:  "no conditions always matches",
			match: MatchCondition{},
			notification: notification.Notification{
				NotificationSource: "billing",
				Labels: map[string]string{
					"env": "prod",
				},
			},
			want: true,
		},
		{
			name: "notification source contains actual value",
			match: MatchCondition{
				NotificationSource: []string{"billing", "payments"},
			},
			notification: notification.Notification{
				NotificationSource: "billing",
			},
			want: true,
		},
		{
			name: "notification source mismatch does not match",
			match: MatchCondition{
				NotificationSource: []string{"billing", "payments"},
			},
			notification: notification.Notification{
				NotificationSource: "ops, support",
			},
			want: false,
		},
		{
			name: "labels match when expected set contains actual value",
			match: MatchCondition{
				Labels: map[string][]string{
					"env": {"prod", "stg"},
				},
			},
			notification: notification.Notification{
				Labels: map[string]string{
					"env": "prod",
				},
			},
			want: true,
		},
		{
			name: "all label keys must match",
			match: MatchCondition{
				Labels: map[string][]string{
					"env":  {"prod", "stg"},
					"team": {"ops", "sre"},
				},
			},
			notification: notification.Notification{
				Labels: map[string]string{
					"env":  "prod",
					"team": "qa",
				},
			},
			want: false,
		},
		{
			name: "missing label key does not match",
			match: MatchCondition{
				Labels: map[string][]string{
					"env": {"prod"},
				},
			},
			notification: notification.Notification{
				Labels: map[string]string{},
			},
			want: false,
		},
		{
			name: "notification source and labels both required",
			match: MatchCondition{
				NotificationSource: []string{"billing"},
				Labels: map[string][]string{
					"env": {"prod"},
				},
			},
			notification: notification.Notification{
				NotificationSource: "billing",
				Labels: map[string]string{
					"env": "prod",
				},
			},
			want: true,
		},
		{
			name: "notification source and labels fail if one side mismatches",
			match: MatchCondition{
				NotificationSource: []string{"billing"},
				Labels: map[string][]string{
					"env": {"prod"},
				},
			},
			notification: notification.Notification{
				NotificationSource: "billing",
				Labels: map[string]string{
					"env": "stg",
				},
			},
			want: false,
		},
		{
			name: "configured values are matched exactly",
			match: MatchCondition{
				NotificationSource: []string{" billing", " ", " payments "},
				Labels: map[string][]string{
					"env": {" prod", " ", " stg "},
				},
			},
			notification: notification.Notification{
				NotificationSource: " payments ",
				Labels: map[string]string{
					"env": "stg",
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.match.IsMatched(tt.notification)
			if got != tt.want {
				t.Fatalf("IsMatch() = %v, want %v", got, tt.want)
			}
		})
	}
}
