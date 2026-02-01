package truecoach

import (
	"encoding/json"
	"fmt"
	"strconv"

	"resty.dev/v3"
)

const (
	apiBaseURL     = "https://api.truecoach.co/api"
	userAgent      = "okhttp/4.12.0"
	accept         = "application/json"
	contentType    = "application/json; charset=utf-8"
	role           = "Client"
	acceptEncoding = "gzip"
)

// Client represents a TrueCoach API client.
type Client struct {
	httpClient *resty.Client
}

// NewClient returns a new TrueCoach API client with standard request headers set.
// Call SetAccessToken after Login to use authenticated endpoints.
func NewClient() *Client {
	return &Client{
		httpClient: resty.New().
			SetBaseURL(apiBaseURL).
			SetHeader("User-Agent", userAgent).
			SetHeader("Accept", accept).
			SetHeader("Content-Type", contentType).
			SetHeader("Role", role).
			SetHeader("Accept-Encoding", acceptEncoding),
	}
}

// ClientID is the user/client ID. The API sometimes returns it as a number;
// we unmarshal either and always treat it as a string.
type ClientID string

// UnmarshalJSON accepts either a JSON number or string for the client ID.
func (c *ClientID) UnmarshalJSON(data []byte) error {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	switch v := v.(type) {
	case string:
		*c = ClientID(v)
	case float64:
		*c = ClientID(strconv.FormatFloat(v, 'f', 0, 64))
	default:
		return fmt.Errorf("user_id: expected string or number, got %T", v)
	}
	return nil
}

// String returns the client ID as a string.
func (c ClientID) String() string { return string(c) }

type TokenResponse struct {
	AccessToken string   `json:"access_token"`
	TokenType   string   `json:"token_type"`
	UserID      ClientID `json:"user_id"`
}

func (c *Client) Login(email, password string) (*TokenResponse, error) {
	res, err := c.httpClient.R().
		SetBody(map[string]string{
			"grant_type": "password",
			"username":   email,
			"password":   password,
		}).
		SetResult(&TokenResponse{}).
		Post("/oauth/token")

	if err != nil {
		return nil, err
	}

	return res.Result().(*TokenResponse), nil
}

// UserProfile is the "user" object returned by the user profile endpoint.
// It contains the authenticated user's info including client_id (used for habit trackers, etc.).
type UserProfile struct {
	ID        int      `json:"id"`
	ClientID  ClientID `json:"client_id"`
	Email     string   `json:"email"`
	FirstName string   `json:"first_name"`
	LastName  string   `json:"last_name"`
	Timezone  string   `json:"timezone"`
	Units     string   `json:"units"`
	Weight    *float64 `json:"weight"`
	Height    *int     `json:"height"`
	ImageID   *int     `json:"image_id"`
}

// UserProfileResponse is the response from GET /users/{userID}.
// The API returns a large payload; we decode the "user" object used for client_id lookup.
type UserProfileResponse struct {
	User UserProfile `json:"user"`
}

// GetUserProfile fetches the user profile for the given user ID.
// Use the ClientID from the response (profile.User.ClientID) for client-scoped endpoints like habit trackers.
func (c *Client) GetUserProfile(authToken string, userID string) (*UserProfileResponse, error) {
	var out UserProfileResponse
	_, err := c.httpClient.R().
		SetHeader("Authorization", "Bearer "+authToken).
		SetResult(&out).
		Get("/users/" + userID)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// HabitTrackerTracking represents a single habit tracker entry for a day.
type HabitTrackerTracking struct {
	ID        int      `json:"id"`
	Calories  *float64 `json:"calories"`
	Date      string   `json:"date"`
	Protein   *float64 `json:"protein"`
	Carbs     *float64 `json:"carbs"`
	Fat       *float64 `json:"fat"`
	Weight    *float64 `json:"weight"`
	Sleep     *float64 `json:"sleep"`
	Steps     *int     `json:"steps"`
	Energy    *float64 `json:"energy"`
	Hunger    *float64 `json:"hunger"`
	Stress    *float64 `json:"stress"`
	Notes     *string  `json:"notes"`
	ClientID  int      `json:"client_id"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

// HabitTrackerResponse is the response from the habit_trackers endpoint.
type HabitTrackerResponse struct {
	Trackings        []HabitTrackerTracking `json:"trackings"`
	PreviousDuration map[string]interface{} `json:"previous_duration"`
	NextDuration     map[string]interface{} `json:"next_duration"`
	CurrentDuration  map[string]interface{} `json:"current_duration"`
	IsPrevious       bool                   `json:"is_previous"`
}

// GetHabitTrackers fetches habit tracker information for a client for the given date.
// The date should be in a format the API expects, e.g. "Feb 1, 2026".
func (c *Client) GetHabitTrackers(authToken string, clientID string, date string) (*HabitTrackerResponse, error) {
	var wrapper struct {
		Response HabitTrackerResponse `json:"response"`
	}
	_, err := c.httpClient.R().
		SetQueryParam("date", date).
		SetHeader("Authorization", "Bearer "+authToken).
		SetResult(&wrapper).
		Get("/clients/" + clientID + "/habit_trackers")
	if err != nil {
		return nil, err
	}
	return &wrapper.Response, nil
}
