package time

import (
	"context"
	"errors"
	"time"
)

type Sleep struct {
}

func (s *Sleep) GetDescription() string {
	//todo CollectionName
	return ""
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
	time.Sleep(durationObj)
	return nil
}
