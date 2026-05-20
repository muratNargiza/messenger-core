package messenger

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/gliedabrennung/messenger-core/internal/pkg/logger"
	"github.com/gliedabrennung/messenger-core/internal/entity"
	"github.com/gliedabrennung/messenger-core/internal/pkg/api"
	"github.com/gliedabrennung/messenger-core/internal/pkg/authctx"
	"github.com/hertz-contrib/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
	maxTextLength  = 4000
)

type Client struct {
	hub      *Hub
	id       int64
	conn     *websocket.Conn
	send     chan []byte
	done     chan struct{}
	tokenExp time.Time
}

type incomingMessage struct {
	To      json.Number `json:"to"`
	Message string      `json:"message"`
}

func (c *Client) nextReadDeadline() time.Time {
	deadline := time.Now().Add(pongWait)
	if !c.tokenExp.IsZero() && c.tokenExp.Before(deadline) {
		return c.tokenExp
	}
	return deadline
}

func (c *Client) readPump() {
	defer func() {
		c.hub.Unregister(c)
		<-c.done
	}()

	c.conn.SetReadLimit(maxMessageSize)
	if err := c.conn.SetReadDeadline(c.nextReadDeadline()); err != nil {
		logger.Errorf("ws: set read deadline: %v", err)
		return
	}
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(c.nextReadDeadline())
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Errorf("ws: read error: %v", err)
			}
			break
		}

		var inc incomingMessage
		if err := json.Unmarshal(message, &inc); err != nil {
			logger.Errorf("ws: invalid json: %v", err)
			continue
		}

		inc.Message = strings.TrimSpace(inc.Message)
		if inc.Message == "" || len(inc.Message) > maxTextLength {
			continue
		}

		toID, err := inc.To.Int64()
		if err != nil {
			logger.Errorf("ws: invalid 'to' ID: %v", err)
			continue
		}

		if toID == c.id {
			continue
		}

		msg := DirectMessage{
			From:    c.id,
			To:      toID,
			Message: inc.Message,
		}

		if c.hub.MsgRepo != nil {
			repoMsg := &entity.Message{
				ChatID:  entity.MakeChatID(c.id, toID),
				FromID:  c.id,
				ToID:    toID,
				Content: inc.Message,
			}
			if err := c.hub.MsgRepo.Save(context.Background(), repoMsg); err != nil {
				logger.Errorf("ws: save message: %v", err)
			}
		}

		if !c.hub.Send(msg) {
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
				logger.Errorf("ws: set write deadline: %v", err)
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				logger.Errorf("ws: set write deadline (ping): %v", err)
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

		tokenExp, _ := authctx.TokenExp(c)

		err := upgrader.Upgrade(c, func(conn *websocket.Conn) {
			client := &Client{
				hub:      hub,
				id:       userID,
				conn:     conn,
				send:     make(chan []byte, 256),
				done:     make(chan struct{}),
				tokenExp: tokenExp,
			}

			hub.Register(client)

			go client.writePump()
			client.readPump()
		})

		if err != nil {
			logger.CtxErrorf(ctx, "ws: upgrade error: %v", err)
			api.ErrorResponse(c, http.StatusInternalServerError,
				"WEBSOCKET_UPGRADE_FAILED", "could not upgrade to websocket connection", nil)
		}
	}
}
