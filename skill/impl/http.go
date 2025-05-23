package impl

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"slices"

	httpPKG "github.com/luoxiaojun1992/ai-agent/pkg/http"
)

type Http struct {
	Client         *httpPKG.Client
	AllowedURLList []string
}

func (h *Http) GetDescription() string {
	//todo
	return ""
}

func (h *Http) ShortDescription() string {
	//todo
	return ""
}

func (h *Http) Do(ctx context.Context, cmdCtx any, callback func(output any) (any, error)) error {
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
	if len(h.AllowedURLList) > 0 && slices.Contains(h.AllowedURLList, pathStr) {
		return errors.New("path is not allowed")
	}

	body, hasBody := params["body"]
	if !hasBody {
		return errors.New("not found body from params")
	}
	bodyStr, isValidBody := body.(string)
	if !isValidBody {
		return errors.New("error converting body from params")
	}

	queryParams, hasQueryParams := params["query_params"]
	if !hasQueryParams {
		return errors.New("not found query_params from params")
	}
	queryParamsMap, isValidQueryParams := queryParams.(url.Values)
	if !isValidQueryParams {
		return errors.New("error converting query_params from params")
	}

	httpHeader, hasHttpHeader := params["http_header"]
	if !hasHttpHeader {
		return errors.New("not found http_header from params")
	}
	httpHeaderMap, isValidHttpHeader := httpHeader.(http.Header)
	if !isValidHttpHeader {
		return errors.New("error converting http_header from params")
	}

	res, err := h.Client.SendRequest(methodStr, pathStr, bodyStr, queryParamsMap, httpHeaderMap)
	if err != nil {
		return err
	}
	_, err = callback(res)
	return err
}
