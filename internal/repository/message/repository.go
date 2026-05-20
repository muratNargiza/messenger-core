package message

import (
	"context"

	"github.com/gliedabrennung/messenger-core/internal/pkg/logger"
	"github.com/gliedabrennung/messenger-core/internal/entity"
	"github.com/gocql/gocql"
	"github.com/redis/go-redis/v9"
)

type Repository struct {
	scylla *ScyllaStorage
	redis  *RedisCache
}

func NewRepository(scyllaSession *gocql.Session, rdb *redis.Client) *Repository {
	return &Repository{
		scylla: NewScyllaStorage(scyllaSession),
		redis:  NewRedisCache(rdb),
	}
}

func (r *Repository) Save(ctx context.Context, msg *entity.Message) error {
	if err := r.scylla.Save(ctx, msg); err != nil {
		return err
	}

	r.redis.CacheMessage(ctx, msg)
	r.redis.Publish(ctx, msg)

	return nil
}

func (r *Repository) GetChatHistory(ctx context.Context, chatID string, limit int, cursor string) ([]*entity.Message, string, error) {
	if cursor == "" {
		if cached, ok := r.redis.GetCachedHistory(ctx, chatID, limit); ok {
			var nextCursor string
			if len(cached) > limit {
				nextCursor = cached[limit-1].MessageID
				cached = cached[:limit]
			}
			return cached, nextCursor, nil
		}
	}

	messages, nextCursor, err := r.scylla.GetHistory(ctx, chatID, limit, cursor)
	if err != nil {
		return nil, "", err
	}

	if cursor == "" && len(messages) > 0 {
		r.redis.WarmUpCache(ctx, chatID, messages)
	}

	return messages, nextCursor, nil
}

func (r *Repository) Subscribe(ctx context.Context, chatID string) (<-chan *entity.Message, func() error, error) {
	logger.CtxInfof(ctx, "subscribing to chat %s", chatID)
	return r.redis.Subscribe(ctx, chatID)
}
