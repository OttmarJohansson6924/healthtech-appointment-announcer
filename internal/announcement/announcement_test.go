package announcement

import (
	"context"
	"testing"
	"time"
)

type recordedPublish struct {
	calls   int
	channel string
	event   string
	key     string
}

func (p *recordedPublish) Publish(_ context.Context, channel, event string, _ any, key string) error {
	p.calls++
	p.channel, p.event, p.key = channel, event, key
	return nil
}

func TestAppointmentBroadcastDecision(t *testing.T) {
	tests := []struct {
		name          string
		status        string
		wantBroadcast bool
		wantCalls     int
	}{
		{name: "new booking stays quiet", status: "scheduled", wantBroadcast: false, wantCalls: 0},
		{name: "completion stays quiet", status: "completed", wantBroadcast: false, wantCalls: 0},
		{name: "cancellation alerts members", status: "cancelled", wantBroadcast: true, wantCalls: 1},
		{name: "reschedule alerts members", status: "rescheduled", wantBroadcast: true, wantCalls: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			publisher := &recordedPublish{}
			service := NewService(publisher)
			result, err := service.Handle(context.Background(), Change{
				AppointmentID: "apt-4821",
				Facility:      "north-clinic",
				Status:        tt.status,
				EffectiveAt:   time.Date(2026, 10, 2, 9, 30, 0, 0, time.UTC),
			})
			if err != nil {
				t.Fatalf("Handle() error = %v", err)
			}
			if result.Broadcast != tt.wantBroadcast || publisher.calls != tt.wantCalls {
				t.Fatalf("broadcast = %v, calls = %d; want %v, %d", result.Broadcast, publisher.calls, tt.wantBroadcast, tt.wantCalls)
			}
			if tt.wantBroadcast {
				if publisher.channel != memberChannel || publisher.event != "appointment.workflow.changed" {
					t.Fatalf("publish target = %q/%q", publisher.channel, publisher.event)
				}
				if publisher.key != "appointment:apt-4821:"+tt.status {
					t.Fatalf("idempotency key = %q", publisher.key)
				}
			}
		})
	}
}
