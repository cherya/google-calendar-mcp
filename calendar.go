package main

import (
	"context"
	"time"

	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

const defaultTimezone = "UTC"

type CalendarClient struct {
	service    *calendar.Service
	calendarID string
	timezone   string
}

type CalendarEvent struct {
	ID      string `json:"id"`
	Summary string `json:"summary"`
	Start   string `json:"start"`
	End     string `json:"end"`
}

type CalendarInfo struct {
	ID          string `json:"id"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
	Primary     bool   `json:"primary"`
}

func NewCalendarClient(credentialsFile, calendarID, timezone string) (*CalendarClient, error) {
	ctx := context.Background()

	srv, err := calendar.NewService(ctx,
		option.WithCredentialsFile(credentialsFile),
		option.WithScopes(calendar.CalendarScope),
	)
	if err != nil {
		return nil, err
	}

	if timezone == "" {
		timezone = defaultTimezone
	}

	return &CalendarClient{
		service:    srv,
		calendarID: calendarID,
		timezone:   timezone,
	}, nil
}

func (c *CalendarClient) resolveCalendarID(calendarID string) string {
	if calendarID != "" {
		return calendarID
	}
	return c.calendarID
}

// ListEventsForDays returns events for the next N days
func (c *CalendarClient) ListEventsForDays(ctx context.Context, calendarID string, days int) ([]CalendarEvent, error) {
	loc, err := time.LoadLocation(c.timezone)
	if err != nil {
		loc = time.UTC
	}

	now := time.Now().In(loc)
	timeMin := now.Format(time.RFC3339)
	timeMax := now.AddDate(0, 0, days).Format(time.RFC3339)

	return c.listEvents(ctx, calendarID, timeMin, timeMax, 100)
}

// ListEventsRange returns events between two dates (YYYY-MM-DD format)
func (c *CalendarClient) ListEventsRange(ctx context.Context, calendarID string, startDate, endDate string) ([]CalendarEvent, error) {
	loc, err := time.LoadLocation(c.timezone)
	if err != nil {
		loc = time.UTC
	}

	start, err := time.ParseInLocation("2006-01-02", startDate, loc)
	if err != nil {
		return nil, err
	}

	end, err := time.ParseInLocation("2006-01-02", endDate, loc)
	if err != nil {
		return nil, err
	}

	// End date should be inclusive, so add one day
	end = end.AddDate(0, 0, 1)

	timeMin := start.Format(time.RFC3339)
	timeMax := end.Format(time.RFC3339)

	return c.listEvents(ctx, calendarID, timeMin, timeMax, 100)
}

func (c *CalendarClient) listEvents(ctx context.Context, calendarID string, timeMin, timeMax string, maxResults int) ([]CalendarEvent, error) {
	call := c.service.Events.List(c.resolveCalendarID(calendarID)).
		SingleEvents(true).
		OrderBy("startTime").
		MaxResults(int64(maxResults)).
		TimeMin(timeMin).
		TimeMax(timeMax)

	events, err := call.Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	result := make([]CalendarEvent, 0, len(events.Items))
	for _, e := range events.Items {
		start := e.Start.DateTime
		if start == "" {
			start = e.Start.Date
		}
		end := e.End.DateTime
		if end == "" {
			end = e.End.Date
		}
		result = append(result, CalendarEvent{
			ID:      e.Id,
			Summary: e.Summary,
			Start:   start,
			End:     end,
		})
	}

	return result, nil
}

// CreateEvent creates a new calendar event
// date: YYYY-MM-DD, startTime/endTime: HH:MM
func (c *CalendarClient) CreateEvent(ctx context.Context, calendarID string, summary, description, date, startTime, endTime string) (*calendar.Event, error) {
	loc, err := time.LoadLocation(c.timezone)
	if err != nil {
		loc = time.UTC
	}

	startStr := date + "T" + startTime + ":00"
	endStr := date + "T" + endTime + ":00"

	start, err := time.ParseInLocation("2006-01-02T15:04:05", startStr, loc)
	if err != nil {
		return nil, err
	}

	end, err := time.ParseInLocation("2006-01-02T15:04:05", endStr, loc)
	if err != nil {
		return nil, err
	}

	event := &calendar.Event{
		Summary:     summary,
		Description: description,
		Start: &calendar.EventDateTime{
			DateTime: start.Format(time.RFC3339),
			TimeZone: c.timezone,
		},
		End: &calendar.EventDateTime{
			DateTime: end.Format(time.RFC3339),
			TimeZone: c.timezone,
		},
	}

	return c.service.Events.Insert(c.resolveCalendarID(calendarID), event).Context(ctx).Do()
}

// EventUpdates contains optional fields to update
type EventUpdates struct {
	Summary     *string
	Description *string
	Date        *string
	StartTime   *string
	EndTime     *string
}

// UpdateEvent updates an existing calendar event
func (c *CalendarClient) UpdateEvent(ctx context.Context, calendarID string, eventID string, updates EventUpdates) (*calendar.Event, error) {
	// First, get the existing event
	existing, err := c.service.Events.Get(c.resolveCalendarID(calendarID), eventID).Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	// Apply updates
	if updates.Summary != nil {
		existing.Summary = *updates.Summary
	}
	if updates.Description != nil {
		existing.Description = *updates.Description
	}

	// Handle date/time updates
	if updates.Date != nil || updates.StartTime != nil || updates.EndTime != nil {
		loc, err := time.LoadLocation(c.timezone)
		if err != nil {
			loc = time.UTC
		}

		// Parse existing start/end times
		var currentDate, currentStartTime, currentEndTime string

		if existing.Start.DateTime != "" {
			t, _ := time.Parse(time.RFC3339, existing.Start.DateTime)
			t = t.In(loc)
			currentDate = t.Format("2006-01-02")
			currentStartTime = t.Format("15:04")
		}
		if existing.End.DateTime != "" {
			t, _ := time.Parse(time.RFC3339, existing.End.DateTime)
			t = t.In(loc)
			currentEndTime = t.Format("15:04")
		}

		// Apply updates with fallback to current values
		date := currentDate
		startTime := currentStartTime
		endTime := currentEndTime

		if updates.Date != nil {
			date = *updates.Date
		}
		if updates.StartTime != nil {
			startTime = *updates.StartTime
		}
		if updates.EndTime != nil {
			endTime = *updates.EndTime
		}

		// Build new datetime strings
		startStr := date + "T" + startTime + ":00"
		endStr := date + "T" + endTime + ":00"

		start, err := time.ParseInLocation("2006-01-02T15:04:05", startStr, loc)
		if err != nil {
			return nil, err
		}
		end, err := time.ParseInLocation("2006-01-02T15:04:05", endStr, loc)
		if err != nil {
			return nil, err
		}

		existing.Start = &calendar.EventDateTime{
			DateTime: start.Format(time.RFC3339),
			TimeZone: c.timezone,
		}
		existing.End = &calendar.EventDateTime{
			DateTime: end.Format(time.RFC3339),
			TimeZone: c.timezone,
		}
	}

	return c.service.Events.Update(c.resolveCalendarID(calendarID), eventID, existing).Context(ctx).Do()
}

// DeleteEvent deletes a calendar event
func (c *CalendarClient) DeleteEvent(ctx context.Context, calendarID string, eventID string) error {
	return c.service.Events.Delete(c.resolveCalendarID(calendarID), eventID).Context(ctx).Do()
}

// ListCalendars returns all calendars accessible by the service account
func (c *CalendarClient) ListCalendars(ctx context.Context) ([]CalendarInfo, error) {
	res, err := c.service.CalendarList.List().Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	calendars := make([]CalendarInfo, 0, len(res.Items))
	for _, cal := range res.Items {
		calendars = append(calendars, CalendarInfo{
			ID:          cal.Id,
			Summary:     cal.Summary,
			Description: cal.Description,
			Primary:     cal.Primary,
		})
	}
	return calendars, nil
}
