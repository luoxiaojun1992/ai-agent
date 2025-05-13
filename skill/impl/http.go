package impl

import "context"

type Http struct {
}

func (h *Http) GetDescription() string {
	//todo
	return ""
}

func (h *Http) Do(ctx context.Context, cmdCtx any, callback func(output any) (any, error)) error {
	//todo
	return nil
}
