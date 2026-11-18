package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type NotificationRecipient struct {
	UserID uuid.UUID `json:"user_id"`
	Name   string    `json:"name"`
	Email  string    `json:"email"`
}

type NotificationEvent struct {
	EventID     uuid.UUID               `json:"event_id"`
	BookingID   uuid.UUID               `json:"booking_id"`
	Type        string                  `json:"type"`
	Title       string                  `json:"title"`
	Description string                  `json:"description"`
	StartsAt    time.Time               `json:"starts_at"`
	EndsAt      time.Time               `json:"ends_at"`
	Recipients  []NotificationRecipient `json:"recipients"`
}

type NotificationClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewNotificationClient(baseURL string) *NotificationClient {
	return &NotificationClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *NotificationClient) SendEvent(
	ctx context.Context,
	event NotificationEvent,
) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal notification event: %w", err)
	}

	endpoint := c.baseURL + "/events"

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("create notification request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request notification-service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"notification-service returned status %d",
			resp.StatusCode,
		)
	}

	return nil
}
