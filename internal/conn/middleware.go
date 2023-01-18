package conn

import (
	"context"
	"errors"
	"net/http"

	"github.com/SpatiumPortae/portal/internal/logger"
	"go.uber.org/zap"
	"nhooyr.io/websocket"
)

type connKey struct{}

func WithConn(ctx context.Context, conn Conn) context.Context {
	return context.WithValue(ctx, connKey{}, conn)
}

func FromContext(ctx context.Context) (Conn, error) {
	conn, ok := ctx.Value(connKey{}).(Conn)
	if !ok {
		return nil, errors.New("unable to get Conn from context")
	}
	return conn, nil
}

func Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			logger, err := logger.FromContext(ctx)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
			wsConn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
			if err != nil {
				logger.Error("failed to upgrade connection", zap.Error(err))
				return
			}
			next.ServeHTTP(w, r.WithContext(WithConn(r.Context(), &WS{Conn: wsConn})))
		})
	}
}
