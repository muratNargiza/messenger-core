package messenger

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/gliedabrennung/messenger-core/internal/pkg/api"
	"github.com/gliedabrennung/messenger-core/internal/pkg/authctx"
	"github.com/hertz-contrib/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 1024
)

type Client struct {
	hub  *Hub
	id   int64
	conn *websocket.Conn
	send chan []byte
	done chan struct{}
}

type incomingMessage struct {
	To      json.Number `json:"to"`
	Message string      `json:"message"`
}

func (c *Client) readPump() {
	defer func() {
		select {
		case c.hub.unregister <- c:
		case <-c.hub.done:
		}
		<-c.done
	}()

	c.conn.SetReadLimit(maxMessageSize)
	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		hlog.Errorf("ws: set read deadline: %v", err)
		return
	}
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				hlog.Errorf("ws: read error: %v", err)
			}
			break
		}

		var inc incomingMessage
		if err := json.Unmarshal(message, &inc); err != nil {
			hlog.Errorf("ws: invalid json: %v", err)
			continue
		}

		toID, err := inc.To.Int64()
		if err != nil {
			hlog.Errorf("ws: invalid 'to' ID: %v", err)
			continue
		}

		msg := DirectMessage{
			From:    c.id,
			To:      toID,
			Message: inc.Message,
		}
		select {
		case c.hub.direct <- msg:
		case <-c.hub.done:
			return
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
		close(c.done)
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				hlog.Errorf("ws: set write deadline: %v", err)
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				hlog.Errorf("ws: set write deadline (ping): %v", err)
				return
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func NewUpgrader(allowedOrigins []string) websocket.HertzUpgrader {
	allowed := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = true
	}
	return websocket.HertzUpgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(ctx *app.RequestContext) bool {
			if len(allowed) == 0 {
				return false
			}
			if allowed["*"] {
				return true
			}
			origin := string(ctx.GetHeader("Origin"))
			return allowed[origin]
		},
	}
}

func ServeWs(hub *Hub, upgrader websocket.HertzUpgrader) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		userID, ok := authctx.UserID(c)
		if !ok {
			api.ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context", nil)
			return
		}

		err := upgrader.Upgrade(c, func(conn *websocket.Conn) {
			client := &Client{
				hub:  hub,
				id:   userID,
				conn: conn,
				send: make(chan []byte, 256),
				done: make(chan struct{}),
			}

			select {
			case hub.register <- client:
			case <-hub.done:
				conn.Close()
				return
			}

			go client.writePump()
			client.readPump()
		})

		if err != nil {
			hlog.CtxErrorf(ctx, "ws: upgrade error: %v", err)
			api.ErrorResponse(c, http.StatusInternalServerError,
				"WEBSOCKET_UPGRADE_FAILED", "could not upgrade to websocket connection", nil)
		}
	}
}
