package loggermiddleware

import (
	"context"
	"encoding/json"

	"github.com/apex/log"
	"github.com/aquasecurity/lmdrouter"
	"github.com/aws/aws-lambda-go/events"
)

func LoggerMiddleware(next lmdrouter.Handler) lmdrouter.Handler {
	return func(ctx context.Context, req events.APIGatewayProxyRequest) (
		res events.APIGatewayProxyResponse,
		err error,
	) {
		logger := log.WithFields(log.Fields{
			"request_id":     req.RequestContext.RequestID,
			"request_method": req.HTTPMethod,
			"request_host":   req.RequestContext.DomainName,
			"request_path":   req.Path,
		})
		if logger.Level == log.DebugLevel {
			reqJson, _ := json.Marshal(req)
			logger.Debug(string(reqJson))
		}
		ctx = context.WithValue(ctx, "log", logger)
		return next(ctx, req)
	}
}
