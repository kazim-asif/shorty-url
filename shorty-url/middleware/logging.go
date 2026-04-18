package middleware

import (
	"encoding/json"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web/context"
)

type RequestLog struct {
	Timestamp    time.Time `json:"timestamp"`
	Method       string    `json:"method"`
	URL          string    `json:"url"`
	UserAgent    string    `json:"user_agent"`
	IP           string    `json:"ip"`
	StatusCode   int       `json:"status_code"`
	ResponseTime int64     `json:"response_time_ms"`
	RequestBody  string    `json:"request_body,omitempty"`
}

func LoggingMiddleware(ctx *context.Context) {
	start := time.Now()

	requestLog := RequestLog{
		Timestamp: start,
		Method:    ctx.Input.Method(),
		URL:       ctx.Input.URL(),
		UserAgent: ctx.Input.Header("User-Agent"),
		IP:        getClientIP(ctx),
	}

	if ctx.Input.Method() == "POST" || ctx.Input.Method() == "PUT" {
		if body := ctx.Input.RequestBody; len(body) > 0 && len(body) < 1024 {
			requestLog.RequestBody = string(body)
		}
	}

	defer func() {
		requestLog.ResponseTime = time.Since(start).Milliseconds()
		requestLog.StatusCode = ctx.Output.Status

		logData, _ := json.Marshal(requestLog)
		logs.Info("Request:", string(logData))
	}()
}

func CORSMiddleware(ctx *context.Context) {
	ctx.Output.Header("Access-Control-Allow-Origin", "*")
	ctx.Output.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	ctx.Output.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

	if ctx.Input.Method() == "OPTIONS" {
		ctx.Output.SetStatus(200)
		return
	}
}
