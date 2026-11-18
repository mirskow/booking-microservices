package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	Timezone string    `json:"timezone"`
}

type Slot struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type SlotsResponse struct {
	UserID   uuid.UUID `json:"user_id"`
	Date     string    `json:"date"`
	Timezone string    `json:"timezone"`
	Slots    []Slot    `json:"slots"`
}

type UserClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewUserClient(baseURL string) *UserClient {
	return &UserClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *UserClient) GetUser(
	ctx context.Context,
	userID uuid.UUID,
) (*User, error) {
	endpoint := fmt.Sprintf(
		"%s/users/%s",
		c.baseURL,
		userID,
	)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		endpoint,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create get user request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request user-service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrUserNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"user-service returned status %d",
			resp.StatusCode,
		)
	}

	var user User

	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("decode user response: %w", err)
	}

	return &user, nil
}

func (c *UserClient) GetSlots(
	ctx context.Context,
	userID uuid.UUID,
	date string,
) (*SlotsResponse, error) {
	endpoint := fmt.Sprintf(
		"%s/users/%s/slots?date=%s",
		c.baseURL,
		userID,
		url.QueryEscape(date),
	)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		endpoint,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create get slots request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request user-service slots: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrUserNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"user-service returned status %d for slots",
			resp.StatusCode,
		)
	}

	var slots SlotsResponse

	if err := json.NewDecoder(resp.Body).Decode(&slots); err != nil {
		return nil, fmt.Errorf("decode slots response: %w", err)
	}

	return &slots, nil
}
