package time

import (
	"context"
	"errors"
	"time"
)

type Sleep struct {
}

func (s *Sleep) GetDescription() string {
	return `Pause execution for a specified duration. This skill introduces delays in execution, useful for rate limiting, waiting for resources, or timed operations.
Parameters:
- duration: string - Duration to sleep in Go duration format (e.g., "5s", "100ms", "1m30s")
Returns: Success status after sleep completes
Examples: "5s" (5 seconds), "100ms" (100 milliseconds), "1m30s" (1 minute 30 seconds)`
}

func (s *Sleep) ShortDescription() string {
	return "Pause execution for specified duration"
}

func (s *Sleep) Do(ctx context.Context, cmdCtx any, _ func(output any) (any, error)) error {
	params, isValidParams := cmdCtx.(map[string]any)
	if !isValidParams {
		return errors.New("error converting params for time/sleep skill")
	}

	duration, hasDuration := params["duration"]
	if !hasDuration {
		return errors.New("not found duration from params")
	}
	durationStr, isValidDuration := duration.(string)
	if !isValidDuration {
		return errors.New("error converting duration from params")
	}

	durationObj, err := time.ParseDuration(durationStr)
	if err != nil {
		return err
	}
	
	// Support context cancellation
	select {
	case <-time.After(durationObj):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
