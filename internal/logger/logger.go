package logger

import (
	"context"
	"errors"
	"net/http"

	"github.com/tomasen/realip"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type loggerKey struct{}

func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

func FromContext(ctx context.Context) (*zap.Logger, error) {
	logger, ok := ctx.Value(loggerKey{}).(*zap.Logger)
	if !ok {
		return nil, errors.New("unable to get logger from context")
	}
	return logger, nil
}

func Middleware(baseLogger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := baseLogger.With(
				zap.String("request_ip", realip.FromRequest(r)),
				zap.String("endpoint", r.URL.Path),
			)
			next.ServeHTTP(w, r.WithContext(WithLogger(r.Context(), logger)))
		})
	}
}

func New() *zap.Logger {
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	logger, _ := cfg.Build()
	return logger
}
