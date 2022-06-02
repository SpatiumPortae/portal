package conn

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
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
	wsUpgrader := websocket.Upgrader{}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wsConn, err := wsUpgrader.Upgrade(w, r, nil)
			if err != nil {
				log.Println("failed to upgrade connection:", err)
				return
			}
			next.ServeHTTP(w, r.WithContext(WithConn(r.Context(), &WS{Conn: wsConn})))
		})
	}
}
