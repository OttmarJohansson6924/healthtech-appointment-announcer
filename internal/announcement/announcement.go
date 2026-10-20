package announcement

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

const memberChannel = "healthtech-members"

type Change struct {
	AppointmentID string    `json:"appointment_id"`
	Facility      string    `json:"facility"`
	Status        string    `json:"status"`
	EffectiveAt   time.Time `json:"effective_at"`
}

type Result struct {
	Broadcast bool   `json:"broadcast"`
	Channel   string `json:"channel,omitempty"`
	Reason    string `json:"reason"`
}

type Publisher interface {
	Publish(context.Context, string, string, any, string) error
}

type Service struct {
	publisher Publisher
}

func NewService(publisher Publisher) *Service {
	return &Service{publisher: publisher}
}

func (s *Service) Handle(ctx context.Context, change Change) (Result, error) {
	if strings.TrimSpace(change.AppointmentID) == "" || strings.TrimSpace(change.Facility) == "" || change.EffectiveAt.IsZero() {
		return Result{}, errors.New("appointment_id, facility, and effective_at are required")
	}

	status := strings.ToLower(strings.TrimSpace(change.Status))
	switch status {
	case "scheduled", "completed":
		return Result{Broadcast: false, Reason: "routine appointment state"}, nil
	case "cancelled", "rescheduled":
		change.Status = status
		key := fmt.Sprintf("appointment:%s:%s", change.AppointmentID, status)
		if err := s.publisher.Publish(ctx, memberChannel, "appointment.workflow.changed", change, key); err != nil {
			return Result{}, err
		}
		return Result{Broadcast: true, Channel: memberChannel, Reason: "operational action required"}, nil
	default:
		return Result{}, fmt.Errorf("unsupported appointment status %q", change.Status)
	}
}
