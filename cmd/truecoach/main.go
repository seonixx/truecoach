// CLI tool for the TrueCoach fitness app API.
//
// Log in once to store credentials locally:
//
//	truecoach login -email you@example.com -password secret
//
// Then use other commands without extra flags:
//
//	truecoach profile
//	truecoach habits
//	truecoach update-habit -id 123 -steps 10000
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/seonixx/truecoach"
)

const configDir = ".truecoach"
const configFile = "config.json"

type config struct {
	Token    string `json:"token"`
	UserID   string `json:"user_id"`
	ClientID string `json:"client_id"`
}

func configPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fatalf("cannot determine home directory: %v", err)
	}
	return filepath.Join(home, configDir, configFile)
}

func loadConfig() config {
	data, err := os.ReadFile(configPath())
	if err != nil {
		fatalf("not logged in (run 'truecoach login' first)")
	}
	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		fatalf("corrupt config file: %v", err)
	}
	return cfg
}

func saveConfig(cfg config) {
	dir := filepath.Dir(configPath())
	if err := os.MkdirAll(dir, 0700); err != nil {
		fatalf("cannot create config directory: %v", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		fatalf("cannot marshal config: %v", err)
	}
	if err := os.WriteFile(configPath(), data, 0600); err != nil {
		fatalf("cannot write config: %v", err)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: truecoach <command> [flags]

Commands:
  login          Authenticate and store credentials
  profile        Fetch and display the user profile
  habits         Fetch habit tracker entries for a date
  update-habit   Update a habit tracker entry

Credentials are stored in ~/%s/%s after login.
`, configDir, configFile)
	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	cmd := os.Args[1]
	switch cmd {
	case "login":
		cmdLogin()
	case "profile":
		cmdProfile()
	case "habits":
		cmdHabits()
	case "update-habit":
		cmdUpdateHabit()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		usage()
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func parseDate(s string) truecoach.Date {
	if s == "" {
		return truecoach.Today()
	}
	d, err := truecoach.ParseDate(s)
	if err != nil {
		fatalf("%v", err)
	}
	return d
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

// login authenticates, resolves IDs, and saves config.
func cmdLogin() {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	email := fs.String("email", "", "account email (required)")
	password := fs.String("password", "", "account password (required)")
	fs.Parse(os.Args[2:])

	if *email == "" || *password == "" {
		fatalf("both -email and -password are required")
	}

	client := truecoach.NewClient()

	fmt.Fprintln(os.Stderr, "Logging in...")
	token, err := client.Login(*email, *password)
	if err != nil {
		fatalf("login failed: %v", err)
	}

	fmt.Fprintln(os.Stderr, "Fetching profile...")
	profile, err := client.GetUserProfile(token.AccessToken, token.UserID.String())
	if err != nil {
		fatalf("failed to fetch profile: %v", err)
	}

	cfg := config{
		Token:    token.AccessToken,
		UserID:   token.UserID.String(),
		ClientID: profile.User.ClientID.String(),
	}
	saveConfig(cfg)

	fmt.Fprintf(os.Stderr, "Logged in as %s %s (client_id=%s)\n", profile.User.FirstName, profile.User.LastName, cfg.ClientID)
	fmt.Fprintf(os.Stderr, "Credentials saved to ~/%s/%s\n", configDir, configFile)
}

// profile fetches and displays the user profile.
func cmdProfile() {
	cfg := loadConfig()
	client := truecoach.NewClient()
	profile, err := client.GetUserProfile(cfg.Token, cfg.UserID)
	if err != nil {
		fatalf("failed to fetch profile: %v", err)
	}
	printJSON(profile.User)
}

// habits fetches habit tracker entries.
func cmdHabits() {
	fs := flag.NewFlagSet("habits", flag.ExitOnError)
	dateStr := fs.String("date", "", "date to fetch (e.g. \"Apr 19, 2026\" or \"2026-04-19\"), defaults to today")
	fs.Parse(os.Args[2:])

	cfg := loadConfig()
	client := truecoach.NewClient()
	habits, err := client.GetHabitTrackers(cfg.Token, cfg.ClientID, parseDate(*dateStr))
	if err != nil {
		fatalf("failed to fetch habit trackers: %v", err)
	}
	printJSON(habits)
}

// update-habit updates a single habit tracker entry.
func cmdUpdateHabit() {
	fs := flag.NewFlagSet("update-habit", flag.ExitOnError)
	dateStr := fs.String("date", "", "date for the entry (e.g. \"Apr 19, 2026\" or \"2026-04-19\"), defaults to today")
	steps := fs.Int("steps", 0, "step count")
	weight := fs.Float64("weight", 0, "body weight")
	calories := fs.Float64("calories", 0, "calories")
	protein := fs.Float64("protein", 0, "protein (g)")
	carbs := fs.Float64("carbs", 0, "carbs (g)")
	fat := fs.Float64("fat", 0, "fat (g)")
	sleep := fs.Float64("sleep", 0, "sleep (hours)")
	energy := fs.Float64("energy", 0, "energy level")
	hunger := fs.Float64("hunger", 0, "hunger level")
	stress := fs.Float64("stress", 0, "stress level")
	notes := fs.String("notes", "", "notes")
	fs.Parse(os.Args[2:])

	date := parseDate(*dateStr)

	cfg := loadConfig()
	client := truecoach.NewClient()

	// Fetch the tracking entry for the date to get its ID.
	habits, err := client.GetHabitTrackers(cfg.Token, cfg.ClientID, date)
	if err != nil {
		fatalf("failed to fetch habit trackers: %v", err)
	}
	if len(habits.Trackings) == 0 {
		fatalf("no tracking entry found for %s", date)
	}
	trackingID := strconv.Itoa(habits.Trackings[0].ID)

	input := truecoach.HabitTrackingUpdateInput{Date: date}

	// Only set fields that were explicitly provided.
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "steps":
			input.Steps = steps
		case "weight":
			input.Weight = weight
		case "calories":
			input.Calories = calories
		case "protein":
			input.Protein = protein
		case "carbs":
			input.Carbs = carbs
		case "fat":
			input.Fat = fat
		case "sleep":
			input.Sleep = sleep
		case "energy":
			input.Energy = energy
		case "hunger":
			input.Hunger = hunger
		case "stress":
			input.Stress = stress
		case "notes":
			input.Notes = notes
		}
	})

	result, err := client.UpdateHabitTracker(cfg.Token, cfg.ClientID, trackingID, input)
	if err != nil {
		fatalf("failed to update habit tracker: %v", err)
	}
	printJSON(result)
}
