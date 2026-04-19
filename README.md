# truecoach

Go client and CLI for the [TrueCoach](https://truecoach.co) fitness app API.

## Install

```sh
go install github.com/seonixx/truecoach/cmd/truecoach@latest
```

## Getting started

Log in once — this authenticates, resolves your IDs, and saves credentials to `~/.truecoach/config.json`:

```sh
truecoach login -email you@example.com -password secret
```

Then use any command:

```sh
truecoach profile
truecoach habits
truecoach habits -date "Apr 19, 2026"
truecoach update-habit -steps 10000 -weight 180.5
truecoach update-habit -date "Apr 19, 2026" -steps 10000
```

## Library usage

```go
client := truecoach.NewClient()

token, _ := client.Login("you@example.com", "secret")

profile, _ := client.GetUserProfile(token.AccessToken, token.UserID.String())
clientID := profile.User.ClientID.String()

habits, _ := client.GetHabitTrackers(token.AccessToken, clientID, truecoach.Today())
```
