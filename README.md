# calendar-mcp

MCP server for Google Calendar. Runs over stdio using JSON-RPC 2.0.

## Features

- **list_events** — upcoming events for the next N days (default: 7)
- **list_events_range** — events between two dates
- **create_event** — create an event with date and time
- **edit_event** — update an existing event
- **delete_event** — delete an event

## Requirements

- Go 1.24+
- Google Cloud service account with Calendar API enabled
- Calendar shared with the service account email

## Setup

### 1. Google Credentials

Follow [docs/google-service-account-setup.md](docs/google-service-account-setup.md) to create a service account and get the JSON key file.

### 2. Build

```
go build -o calendar-mcp
```

### 3. Environment Variables

- `GOOGLE_CREDENTIALS_FILE` — path to the service account JSON key
- `CALENDAR_ID` — Google Calendar ID (usually your email address)
- `CALENDAR_TIMEZONE` — IANA timezone (e.g. `Europe/Berlin`), defaults to `UTC`

## Usage with Claude Desktop

Add to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "google-calendar": {
      "command": "/path/to/calendar-mcp",
      "env": {
        "GOOGLE_CREDENTIALS_FILE": "/path/to/service-account.json",
        "CALENDAR_ID": "your-email@gmail.com",
        "CALENDAR_TIMEZONE": "America/New_York"
      }
    }
  }
}
```

## License

[MIT](LICENSE)
