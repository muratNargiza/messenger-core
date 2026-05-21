package message

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gliedabrennung/messenger-core/internal/common/logger"
	"github.com/gliedabrennung/messenger-core/internal/entity"
	"github.com/redis/go-redis/v9"
)

const (
	cacheMaxLen = 100
	cacheTTL    = 48 * time.Hour
)

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

func (r *RedisCache) CacheMessage(ctx context.Context, msg *entity.Message) {
	cacheKey := fmt.Sprintf("chat:%s:cache", msg.ChatID)
	data, err := json.Marshal(msg)
	if err != nil {
		logger.CtxErrorf(ctx, "redis marshal message failed: %v", err)
		return
	}

	pipe := r.client.Pipeline()
	pipe.LPush(ctx, cacheKey, data)
	pipe.LTrim(ctx, cacheKey, 0, cacheMaxLen-1)
	pipe.Expire(ctx, cacheKey, cacheTTL)

	if _, err := pipe.Exec(ctx); err != nil {
		logger.CtxErrorf(ctx, "redis cache message failed for chat %s: %v", msg.ChatID, err)
	} else {
		logger.CtxInfof(ctx, "redis cached message %s for chat %s", msg.MessageID, msg.ChatID)
	}
}

func (r *RedisCache) GetCachedHistory(ctx context.Context, chatID string, limit int) ([]*entity.Message, bool) {
	cacheKey := fmt.Sprintf("chat:%s:cache", chatID)
	cached, err := r.client.LRange(ctx, cacheKey, 0, int64(limit)).Result()
	if err != nil || len(cached) == 0 {
		return nil, false
	}

	messages := make([]*entity.Message, 0, len(cached))
	for _, data := range cached {
		var msg entity.Message
		if err := json.Unmarshal([]byte(data), &msg); err == nil {
			messages = append(messages, &msg)
		}
	}

	logger.CtxInfof(ctx, "redis returned %d cached messages for chat %s", len(messages), chatID)
	return messages, true
}

func (r *RedisCache) WarmUpCache(ctx context.Context, chatID string, messages []*entity.Message) {
	if len(messages) == 0 {
		return
	}

	cacheKey := fmt.Sprintf("chat:%s:cache", chatID)
	pipe := r.client.Pipeline()
	pipe.Del(ctx, cacheKey)

	for i := len(messages) - 1; i >= 0; i-- {
		data, err := json.Marshal(messages[i])
		if err == nil {
			pipe.LPush(ctx, cacheKey, data)
		}
	}

	pipe.Expire(ctx, cacheKey, cacheTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		logger.CtxErrorf(ctx, "redis warmup cache failed for chat %s: %v", chatID, err)
	} else {
		logger.CtxInfof(ctx, "redis warmed up cache with %d messages for chat %s", len(messages), chatID)
	}
}

func (r *RedisCache) Publish(ctx context.Context, msg *entity.Message) {
	pubsubKey := fmt.Sprintf("chat:%s:pubsub", msg.ChatID)
	data, err := json.Marshal(msg)
	if err != nil {
		logger.CtxErrorf(ctx, "redis marshal pubsub message failed: %v", err)
		return
	}

	if err := r.client.Publish(ctx, pubsubKey, data).Err(); err != nil {
		logger.CtxErrorf(ctx, "redis publish failed for chat %s: %v", msg.ChatID, err)
	} else {
		logger.CtxInfof(ctx, "redis published message %s for chat %s", msg.MessageID, msg.ChatID)
	}
}

func (r *RedisCache) Subscribe(ctx context.Context, chatID string) (<-chan *entity.Message, func() error, error) {
	pubsubKey := fmt.Sprintf("chat:%s:pubsub", chatID)
	pubsub := r.client.Subscribe(ctx, pubsubKey)

	msgCh := make(chan *entity.Message, 100)

	go func() {
		defer close(msgCh)
		ch := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case redisMsg, ok := <-ch:
				if !ok {
					return
				}
				var msg entity.Message
				if err := json.Unmarshal([]byte(redisMsg.Payload), &msg); err == nil {
					select {
					case msgCh <- &msg:
					default:
						logger.CtxWarnf(ctx, "redis pubsub channel full, dropped message for chat %s", chatID)
					}
				}
			}
		}
	}()

	logger.CtxInfof(ctx, "redis subscribed to chat %s", chatID)
	return msgCh, pubsub.Close, nil
}
