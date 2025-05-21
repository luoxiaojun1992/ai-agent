package impl

import (
	"context"
	"errors"

	httpPKG "github.com/luoxiaojun1992/ai-agent/pkg/http"
)

type Http struct {
	Client *httpPKG.Client
}

func (h *Http) GetDescription() string {
	//todo
	return ""
}

func (h *Http) Do(ctx context.Context, cmdCtx any, callback func(output any) (any, error)) error {
	//todo
	params, isValidParams := cmdCtx.(map[string]any)
	if !isValidParams {
		return errors.New("error converting params for http skill")
	}

	method, hasMethod := params["method"]
	if !hasMethod {
		return errors.New("not found method from params")
	}
	methodStr, isValidMethod := method.(string)
	if !isValidMethod {
		return errors.New("error converting method from params")
	}

	path, hasPath := params["path"]
	if !hasPath {
		return errors.New("not found path from params")
	}
	pathStr, isValidPath := path.(string)
	if !isValidPath {
		return errors.New("error converting path from params")
	}

	body, hasBody := params["body"]
	if !hasBody {
		return errors.New("not found body from params")
	}
	bodyStr, isValidBody := body.(string)
	if !isValidBody {
		return errors.New("error converting body from params")
	}

	//todo parse params
	res, err := h.Client.SendRequest(methodStr, pathStr, bodyStr, nil, nil)
	if err != nil {
		return err
	}
	_, err = callback(res)
	return err
}
