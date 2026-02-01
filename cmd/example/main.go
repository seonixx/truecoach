// Example script to test TrueCoach API authentication and habit tracker fetching.
//
// Usage:
//
//	TRUECOACH_EMAIL=you@example.com TRUECOACH_PASSWORD=secret go run ./cmd/example
//
// Optional env var:
//   - TRUECOACH_DATE: date for habit trackers, e.g. "Feb 1, 2026" (default: today in that format)
package main

import (
	"fmt"
	"os"
	"time"

	"truecoach"
)

func main() {
	email := os.Getenv("TRUECOACH_EMAIL")
	password := os.Getenv("TRUECOACH_PASSWORD")
	if email == "" || password == "" {
		fmt.Fprintln(os.Stderr, "TRUECOACH_EMAIL and TRUECOACH_PASSWORD must be set")
		os.Exit(1)
	}

	client := truecoach.NewClient()

	// 1. Authenticate
	fmt.Println("Logging in...")
	token, err := client.Login(email, password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Login failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Logged in (user_id=%s)\n", token.UserID.String())

	// 2. Get user profile to resolve client ID (user ID ≠ client ID)
	fmt.Println("Fetching user profile...")
	profile, err := client.GetUserProfile(token.AccessToken, token.UserID.String())
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetUserProfile failed: %v\n", err)
		os.Exit(1)
	}
	clientID := profile.User.ClientID.String()
	fmt.Printf("Client ID: %s\n", clientID)

	date := os.Getenv("TRUECOACH_DATE")
	if date == "" {
		date = time.Now().Format("Jan 2, 2006")
	}

	// 3. Fetch habit trackers
	fmt.Printf("Fetching habit trackers for client %s on %s...\n", clientID, date)
	habit, err := client.GetHabitTrackers(token.AccessToken, clientID, date)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetHabitTrackers failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d tracking(s), is_previous=%v\n", len(habit.Trackings), habit.IsPrevious)
	for i, t := range habit.Trackings {
		fmt.Printf("  [%d] id=%d date=%s client_id=%d\n", i+1, t.ID, t.Date, t.ClientID)
		if t.Weight != nil {
			fmt.Printf("       weight=%.2f\n", *t.Weight)
		}
		if t.Steps != nil {
			fmt.Printf("       steps=%d\n", *t.Steps)
		}
		if t.Notes != nil && *t.Notes != "" {
			fmt.Printf("       notes=%s\n", *t.Notes)
		}
	}
}
