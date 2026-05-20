package message

import (
	"context"
	"fmt"
	"time"

	"github.com/gliedabrennung/messenger-core/internal/pkg/logger"
	"github.com/gliedabrennung/messenger-core/internal/entity"
	"github.com/gocql/gocql"
)

type ScyllaStorage struct {
	session *gocql.Session
}

func InitSchema(ctx context.Context, hosts []string, keyspace string) error {
	cluster := gocql.NewCluster(hosts...)
	cluster.Timeout = 5 * time.Second
	session, err := cluster.CreateSession()
	if err != nil {
		logger.CtxErrorf(ctx, "failed to connect to scylla cluster for schema init: %v", err)
		return fmt.Errorf("scylla schema init: connect cluster: %w", err)
	}
	defer session.Close()

	createKeyspaceQuery := fmt.Sprintf(`
		CREATE KEYSPACE IF NOT EXISTS %s
		WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1}`, keyspace)
	if err := session.Query(createKeyspaceQuery).WithContext(ctx).Exec(); err != nil {
		logger.CtxErrorf(ctx, "failed to create keyspace %s: %v", keyspace, err)
		return fmt.Errorf("scylla schema init: create keyspace: %w", err)
	}

	createTableQuery := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s.direct_messages (
			chat_id text,
			message_id timeuuid,
			from_id bigint,
			to_id bigint,
			content text,
			created_at timestamp,
			PRIMARY KEY ((chat_id), message_id)
		) WITH CLUSTERING ORDER BY (message_id DESC)`, keyspace)
	if err := session.Query(createTableQuery).WithContext(ctx).Exec(); err != nil {
		logger.CtxErrorf(ctx, "failed to create direct_messages table in %s: %v", keyspace, err)
		return fmt.Errorf("scylla schema init: create table: %w", err)
	}

	logger.CtxInfof(ctx, "scylla schema initialized successfully in keyspace %s", keyspace)
	return nil
}

func NewScyllaStorage(session *gocql.Session) *ScyllaStorage {
	return &ScyllaStorage{session: session}
}

func (s *ScyllaStorage) Save(ctx context.Context, msg *entity.Message) error {
	id := gocql.TimeUUID()
	msg.MessageID = id.String()
	msg.CreatedAt = time.Now()

	err := s.session.Query(`
		INSERT INTO messenger.direct_messages
		(chat_id, message_id, from_id, to_id, content, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		msg.ChatID, id, msg.FromID, msg.ToID, msg.Content, msg.CreatedAt,
	).WithContext(ctx).Exec()

	if err != nil {
		logger.CtxErrorf(ctx, "scylla save failed for chat %s: %v", msg.ChatID, err)
		return fmt.Errorf("scylla: save message: %w", err)
	}

	logger.CtxInfof(ctx, "scylla saved message %s for chat %s", msg.MessageID, msg.ChatID)
	return nil
}

func (s *ScyllaStorage) GetHistory(ctx context.Context, chatID string, limit int, cursor string) ([]*entity.Message, string, error) {
	var query *gocql.Query

	if cursor == "" {
		query = s.session.Query(`
			SELECT chat_id, message_id, from_id, to_id, content, created_at
			FROM messenger.direct_messages
			WHERE chat_id = ?
			ORDER BY message_id DESC
			LIMIT ?`, chatID, limit+1,
		).WithContext(ctx)
	} else {
		cursorUUID, err := gocql.ParseUUID(cursor)
		if err != nil {
			logger.CtxErrorf(ctx, "scylla parse cursor %s failed: %v", cursor, err)
			return nil, "", fmt.Errorf("scylla: parse cursor: %w", err)
		}
		query = s.session.Query(`
			SELECT chat_id, message_id, from_id, to_id, content, created_at
			FROM messenger.direct_messages
			WHERE chat_id = ? AND message_id < ?
			ORDER BY message_id DESC
			LIMIT ?`, chatID, cursorUUID, limit+1,
		).WithContext(ctx)
	}

	iter := query.Iter()
	messages := make([]*entity.Message, 0, limit+1)

	var (
		msgChatID string
		msgID     gocql.UUID
		fromID    int64
		toID      int64
		content   string
		createdAt time.Time
	)

	for iter.Scan(&msgChatID, &msgID, &fromID, &toID, &content, &createdAt) {
		messages = append(messages, &entity.Message{
			ChatID:    msgChatID,
			MessageID: msgID.String(),
			FromID:    fromID,
			ToID:      toID,
			Content:   content,
			CreatedAt: createdAt,
		})
	}

	if err := iter.Close(); err != nil {
		logger.CtxErrorf(ctx, "scylla get history failed for chat %s: %v", chatID, err)
		return nil, "", fmt.Errorf("scylla: get chat history: %w", err)
	}

	var nextCursor string
	if len(messages) > limit {
		nextCursor = messages[limit-1].MessageID
		messages = messages[:limit]
	}

	logger.CtxInfof(ctx, "scylla retrieved %d messages for chat %s", len(messages), chatID)
	return messages, nextCursor, nil
}
