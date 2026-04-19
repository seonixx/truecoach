package truecoach

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"resty.dev/v3"
)

const (
	// dateAPIFormat is the format the API expects in request parameters (e.g. "Apr 19, 2026").
	dateAPIFormat = "Jan 2, 2006"
	// dateISOFormat is the format the API returns in responses (e.g. "2026-04-19").
	dateISOFormat = "2006-01-02"
)

// Date wraps time.Time for TrueCoach API date fields.
// It marshals to the API request format ("Jan 2, 2006") and
// unmarshals from either the response format ("2006-01-02") or request format.
type Date struct {
	time.Time
}

// NewDate creates a Date from a time.Time, discarding the time component.
func NewDate(t time.Time) Date {
	y, m, d := t.Date()
	return Date{time.Date(y, m, d, 0, 0, 0, 0, t.Location())}
}

// Today returns today's date.
func Today() Date {
	return NewDate(time.Now())
}

// ParseDate parses a date string in either "Jan 2, 2006" or "2006-01-02" format.
func ParseDate(s string) (Date, error) {
	for _, layout := range []string{dateAPIFormat, dateISOFormat} {
		if t, err := time.Parse(layout, s); err == nil {
			return Date{t}, nil
		}
	}
	return Date{}, fmt.Errorf("cannot parse date %q (expected %q or %q)", s, dateAPIFormat, dateISOFormat)
}

func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Format(dateAPIFormat))
}

func (d *Date) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	parsed, err := ParseDate(s)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

// String returns the date in API request format ("Jan 2, 2006").
func (d Date) String() string { return d.Format(dateAPIFormat) }

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

// checkStatus returns an error if the HTTP response indicates failure.
func checkStatus(res *resty.Response) error {
	if res.IsSuccess() {
		return nil
	}
	return fmt.Errorf("API error %d: %s", res.StatusCode(), res.String())
}

// NewClient returns a new TrueCoach API client with standard request headers set.
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
	var v any
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
	var out TokenResponse
	res, err := c.httpClient.R().
		SetBody(map[string]string{
			"grant_type": "password",
			"username":   email,
			"password":   password,
		}).
		SetResult(&out).
		Post("/oauth/token")
	if err != nil {
		return nil, err
	}
	if err := checkStatus(res); err != nil {
		return nil, err
	}
	return &out, nil
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
	res, err := c.httpClient.R().
		SetHeader("Authorization", "Bearer "+authToken).
		SetResult(&out).
		Get("/users/" + userID)
	if err != nil {
		return nil, err
	}
	if err := checkStatus(res); err != nil {
		return nil, err
	}
	return &out, nil
}

// HabitTrackerTracking represents a single habit tracker entry for a day.
type HabitTrackerTracking struct {
	ID        int      `json:"id"`
	Calories  *float64 `json:"calories"`
	Date      Date     `json:"date"`
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
	PreviousDuration map[string]any `json:"previous_duration"`
	NextDuration     map[string]any `json:"next_duration"`
	CurrentDuration  map[string]any `json:"current_duration"`
	IsPrevious       bool                   `json:"is_previous"`
}

// GetHabitTrackers fetches habit tracker information for a client for the given date.
func (c *Client) GetHabitTrackers(authToken string, clientID string, date Date) (*HabitTrackerResponse, error) {
	var wrapper struct {
		Response HabitTrackerResponse `json:"response"`
	}
	res, err := c.httpClient.R().
		SetQueryParam("date", date.String()).
		SetHeader("Authorization", "Bearer "+authToken).
		SetResult(&wrapper).
		Get("/clients/" + clientID + "/habit_trackers")
	if err != nil {
		return nil, err
	}
	if err := checkStatus(res); err != nil {
		return nil, err
	}
	return &wrapper.Response, nil
}

// HabitTrackingUpdateInput is the payload for updating a habit tracker entry for a day.
// Date is required; other fields are optional and only sent when set (omitempty).
type HabitTrackingUpdateInput struct {
	Date   Date     `json:"date"`
	Steps  *int     `json:"steps,omitempty"`
	Weight *float64 `json:"weight,omitempty"`
	// Optional fields the API may accept:
	Calories *float64 `json:"calories,omitempty"`
	Protein  *float64 `json:"protein,omitempty"`
	Carbs    *float64 `json:"carbs,omitempty"`
	Fat      *float64 `json:"fat,omitempty"`
	Sleep    *float64 `json:"sleep,omitempty"`
	Energy   *float64 `json:"energy,omitempty"`
	Hunger   *float64 `json:"hunger,omitempty"`
	Stress   *float64 `json:"stress,omitempty"`
	Notes    *string  `json:"notes,omitempty"`
}

// UpdateHabitTracker updates the habit tracker entry for the given client and tracking ID.
func (c *Client) UpdateHabitTracker(authToken string, clientID string, trackingID string, input HabitTrackingUpdateInput) (*HabitTrackerTracking, error) {
	body := struct {
		HabitTracking HabitTrackingUpdateInput `json:"habit_tracking"`
	}{HabitTracking: input}
	var out HabitTrackerTracking
	res, err := c.httpClient.R().
		SetHeader("Authorization", "Bearer "+authToken).
		SetBody(body).
		SetResult(&out).
		Put("/clients/" + clientID + "/habit_trackers/" + trackingID)
	if err != nil {
		return nil, err
	}
	if err := checkStatus(res); err != nil {
		return nil, err
	}
	return &out, nil
}
