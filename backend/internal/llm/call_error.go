package llm

import (
	"context"
	"errors"
	"net"

	openai "github.com/sashabaranov/go-openai"
)

type CallErrorInfo struct {
	Code    string
	Summary string
}

// ClassifyCallError returns stable, non-sensitive metadata suitable for
// persistence. Provider response bodies and credentials are never stored.
func ClassifyCallError(err error) CallErrorInfo {
	if err == nil {
		return CallErrorInfo{}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return CallErrorInfo{Code: "LLM_TIMEOUT", Summary: "模型调用超时"}
	}
	if errors.Is(err, context.Canceled) {
		return CallErrorInfo{Code: "LLM_CANCELED", Summary: "模型调用已取消"}
	}
	status := 0
	var apiErr *openai.APIError
	if errors.As(err, &apiErr) {
		status = apiErr.HTTPStatusCode
	}
	var requestErr *openai.RequestError
	if status == 0 && errors.As(err, &requestErr) {
		status = requestErr.HTTPStatusCode
	}
	switch {
	case status == 429:
		return CallErrorInfo{Code: "LLM_RATE_LIMITED", Summary: "模型服务触发限流"}
	case status == 401 || status == 403:
		return CallErrorInfo{Code: "LLM_AUTHENTICATION", Summary: "模型服务鉴权失败"}
	case status == 400 || status == 404 || status == 422:
		return CallErrorInfo{Code: "LLM_REQUEST_REJECTED", Summary: "模型服务拒绝请求"}
	case status >= 500:
		return CallErrorInfo{Code: "LLM_UPSTREAM", Summary: "模型上游服务异常"}
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return CallErrorInfo{Code: "LLM_NETWORK", Summary: "模型服务网络异常"}
	}
	return CallErrorInfo{Code: "LLM_PROVIDER_ERROR", Summary: "模型调用失败"}
}
