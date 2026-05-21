package message

import (
	"context"
	"os"
	"testing"

	"github.com/gliedabrennung/messenger-core/internal/entity"
	"github.com/gocql/gocql"
)

func TestScyllaStorage_Integration(t *testing.T) {
	scyllaHosts := os.Getenv("TEST_SCYLLA_HOSTS")
	if scyllaHosts == "" {
		t.Skip("TEST_SCYLLA_HOSTS not set, skipping scylla integration test")
	}

	ctx := context.Background()
	if err := InitSchema(ctx, []string{scyllaHosts}, "ws"); err != nil {
		t.Fatalf("could not initialize schema: %v", err)
	}

	cluster := gocql.NewCluster(scyllaHosts)
	cluster.Keyspace = "ws"
	session, err := cluster.CreateSession()
	if err != nil {
		t.Fatalf("could not connect to scylla: %v", err)
	}
	defer session.Close()

	storage := NewScyllaStorage(session)
	chatID := "test:scylla:1"

	msg1 := &entity.Message{
		ChatID:  chatID,
		FromID:  1,
		ToID:    2,
		Content: "first message",
	}

	err = storage.Save(ctx, msg1)
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}

	msg2 := &entity.Message{
		ChatID:  chatID,
		FromID:  2,
		ToID:    1,
		Content: "second message",
	}
	err = storage.Save(ctx, msg2)
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}

	msgs, cursor, err := storage.GetHistory(ctx, chatID, 1, "")
	if err != nil {
		t.Fatalf("get history failed: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Content != "second message" {
		t.Errorf("expected newest message to be 'second message', got %s", msgs[0].Content)
	}

	if cursor == "" {
		t.Fatal("expected non-empty cursor for next page")
	}

	msgs2, _, err := storage.GetHistory(ctx, chatID, 1, cursor)
	if err != nil {
		t.Fatalf("get history page 2 failed: %v", err)
	}
	if len(msgs2) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs2))
	}
	if msgs2[0].Content != "first message" {
		t.Errorf("expected 'first message', got %s", msgs2[0].Content)
	}
}
