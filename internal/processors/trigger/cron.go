package trigger

import (
	"context"
	"fmt"
	"time"

	"github.com/bakkerme/curator-ai/internal/core"
	"github.com/robfig/cron/v3"
)

type CronProcessor struct {
	name     string
	schedule string
	timezone string
	cron     *cron.Cron
	events   chan core.TriggerEvent
}

func NewCronProcessor(schedule, timezone string) *CronProcessor {
	return &CronProcessor{
		name:     "cron",
		schedule: schedule,
		timezone: timezone,
	}
}

func (c *CronProcessor) Name() string {
	return c.name
}

func (c *CronProcessor) Configure(config map[string]interface{}) error {
	if schedule, ok := config["schedule"].(string); ok {
		c.schedule = schedule
	}
	if timezone, ok := config["timezone"].(string); ok {
		c.timezone = timezone
	}
	return nil
}

func (c *CronProcessor) Validate() error {
	if c.schedule == "" {
		return fmt.Errorf("cron schedule is required")
	}
	if c.timezone != "" {
		if _, err := time.LoadLocation(c.timezone); err != nil {
			return fmt.Errorf("invalid timezone: %w", err)
		}
	}
	return nil
}

func (c *CronProcessor) Start(ctx context.Context, flowID string) (<-chan core.TriggerEvent, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}

	location := time.UTC
	if c.timezone != "" {
		tz, err := time.LoadLocation(c.timezone)
		if err != nil {
			return nil, err
		}
		location = tz
	}

	c.events = make(chan core.TriggerEvent, 1)
	c.cron = cron.New(cron.WithLocation(location))
	_, err := c.cron.AddFunc(c.schedule, func() {
		select {
		case c.events <- core.TriggerEvent{FlowID: flowID, Timestamp: time.Now().UTC()}:
		default:
		}
	})
	if err != nil {
		return nil, err
	}

	c.cron.Start()

	go func() {
		<-ctx.Done()
		_ = c.Stop()
	}()

	return c.events, nil
}

func (c *CronProcessor) Stop() error {
	if c.cron != nil {
		ctx := c.cron.Stop()
		<-ctx.Done()
	}
	if c.events != nil {
		close(c.events)
	}
	return nil
}
