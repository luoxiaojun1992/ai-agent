package skill

import "context"

type Skill interface {
	GetDescription() (string, error)
	Do(ctx context.Context, cmdCtx any, callback func(output any) (any, error)) error
}
