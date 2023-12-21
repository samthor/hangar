package wrap

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"nhooyr.io/websocket"
)

var (
	sw serverWatcher
)

// WebSocketAcceptOptions are the default options applied to WebSocket.
// You may want to set InsecureSkipVerify if you are being lazy/don't care about hosts.
var WebSocketAcceptOptions = websocket.AcceptOptions{}

// HttpFunc is a handler used by Http which allows generating simple result types.
type HttpFunc func(http.ResponseWriter, *http.Request) interface{}

// Http returns a http.HandlerFunc that wraps a HttpFunc capable of convenient return types.
func Http(fn HttpFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out := fn(w, r)
		switch x := out.(type) {
		case nil:
			return
		case []byte:
			w.Write(x)
		case string:
			w.Write([]byte(x))
			return
		case int:
			w.WriteHeader(x)
			return
		case error:
			// ignore
		default:
			w.Header().Set("Content-Type", "application/json")
			err := json.NewEncoder(w).Encode(x)
			if err == nil {
				return
			}
			out = err
		}

		log.Printf("got err handling %s: %v", r.URL.Path, out)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// WebSocketFunc is a handler used in WebSocket.
type WebSocketFunc func(context.Context, *websocket.Conn) error

// WebSocket returns a http.HandlerFunc that wraps a websocket setup/teardown.
func WebSocket(fn WebSocketFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, &WebSocketAcceptOptions)
		if err != nil {
			log.Printf("got err setting up websocket %s: %v", r.URL.Path, err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		ctx := sw.RegisterHttpContext(r.Context())
		err = fn(ctx, c)

		var closeError websocket.CloseError
		if !(errors.Is(err, context.Canceled) || errors.As(err, &closeError)) {
			log.Printf("got err handling websocket %s: %v", r.URL.Path, err)
			c.Close(websocket.StatusInternalError, "")
		} else {
			c.Close(websocket.StatusNormalClosure, "")
		}
	}
}
