package httpapi

import (
	"context"
	"net/http"

	"github.com/coder/websocket"
)

func (a *API) pipelineEvents(w http.ResponseWriter, r *http.Request) {
	connection, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer connection.CloseNow()

	events, unsubscribe := a.events.Subscribe()
	defer unsubscribe()
	disconnected := make(chan struct{})
	go func() {
		defer close(disconnected)
		for {
			if _, _, readErr := connection.Read(r.Context()); readErr != nil {
				return
			}
		}
	}()

	if err = connection.Write(r.Context(), websocket.MessageText, []byte(`{"scope":"pipeline","type":"connected"}`)); err != nil {
		return
	}
	for {
		select {
		case <-r.Context().Done():
			return
		case <-disconnected:
			return
		case event := <-events:
			writeCtx, cancel := context.WithTimeout(r.Context(), websocketWriteTimeout)
			err = connection.Write(writeCtx, websocket.MessageText, event)
			cancel()
			if err != nil {
				return
			}
		}
	}
}
