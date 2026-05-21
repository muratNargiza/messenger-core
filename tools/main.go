package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gliedabrennung/messenger-core/internal/common/logger"

	"github.com/gorilla/websocket"
)

type userResponse struct {
	User struct {
		ID int64 `json:"id"`
	} `json:"user"`
	Token string `json:"token"`
}

func req(baseURL, path string, payload map[string]any) *userResponse {
	b, _ := json.Marshal(payload)
	resp, err := http.Post(baseURL+path, "application/json", bytes.NewReader(b))
	if err != nil {
		logger.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logger.Fatalf("error %s: %s", path, body)
	}
	var out userResponse
	if err := json.Unmarshal(body, &out); err != nil {
		logger.Fatalf("unmarshal %s: %v", path, err)
	}
	return &out
}

func main() {
	baseURL := "http://localhost:8080"
	if v := os.Getenv("BASE_URL"); v != "" {
		baseURL = strings.TrimRight(v, "/")
	}

	ts := time.Now().UnixNano()
	u1 := fmt.Sprintf("u1_%d", ts)
	u2 := fmt.Sprintf("u2_%d", ts)

	req(baseURL, "/auth/register", map[string]any{"username": u1, "password": "password123"})
	req(baseURL, "/auth/register", map[string]any{"username": u2, "password": "password123"})

	r1 := req(baseURL, "/auth/login", map[string]any{"username": u1, "password": "password123"})
	r2 := req(baseURL, "/auth/login", map[string]any{"username": u2, "password": "password123"})

	fmt.Printf("User 1: %d, token: %s\n", r1.User.ID, r1.Token)
	fmt.Printf("User 2: %d, token: %s\n", r2.User.ID, r2.Token)

	wsURL := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws"}
	wsURL.RawQuery = "token=" + r1.Token
	c1, _, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
	if err != nil {
		logger.Fatal("dial c1:", err)
	}
	defer c1.Close()

	wsURL.RawQuery = "token=" + r2.Token
	c2, _, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
	if err != nil {
		logger.Fatal("dial c2:", err)
	}
	defer c2.Close()

	time.Sleep(100 * time.Millisecond)

	msg := map[string]any{
		"to":      fmt.Sprintf("%d", r2.User.ID),
		"message": "hello from 1",
	}
	if err := c1.WriteJSON(msg); err != nil {
		logger.Fatal("write:", err)
	}

	if err := c2.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		logger.Fatal("set read deadline:", err)
	}
	_, p, err := c2.ReadMessage()
	if err != nil {
		logger.Fatal("read c2:", err)
	}
	fmt.Printf("User 2 received: %s\n", p)
}
