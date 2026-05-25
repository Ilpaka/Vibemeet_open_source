package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"vibemeet/internal/domain"
	"vibemeet/pkg/logger"
)

const (
	// Chat TTL - 6 hours
	ChatTTL = 6 * time.Hour

	// Redis key prefixes
	ChatMessagesKeyPrefix     = "chat:room:%s:messages"
	RoomParticipantsKeyPrefix = "room:%s:participants"
	RoomMetaKeyPrefix         = "room:%s:meta"
)

type AnonymousChatRepository interface {
	// Save a message to Redis (TTL 6 hours)
	SaveMessage(ctx context.Context, roomID uuid.UUID, message *domain.AnonymousChatMessage) error

	// Retrieve the latest N messages from Redis, sorted by time
	GetMessages(ctx context.Context, roomID uuid.UUID, limit int) ([]*domain.AnonymousChatMessage, error)

	// Retrieve messages created after the given time
	GetMessagesAfter(ctx context.Context, roomID uuid.UUID, after time.Time, limit int) ([]*domain.AnonymousChatMessage, error)

	// Delete a message
	DeleteMessage(ctx context.Context, roomID uuid.UUID, messageID string) error

	// Update a message
	UpdateMessage(ctx context.Context, roomID uuid.UUID, message *domain.AnonymousChatMessage) error

	// Check whether a room exists in Redis
	RoomExists(ctx context.Context, roomID uuid.UUID) (bool, error)
}

type anonymousChatRepository struct {
	rdb *redis.Client
	log logger.Logger
}

func NewAnonymousChatRepository(rdb *redis.Client, log logger.Logger) AnonymousChatRepository {
	return &anonymousChatRepository{
		rdb: rdb,
		log: log,
	}
}

func (r *anonymousChatRepository) getMessagesKey(roomID uuid.UUID) string {
	return fmt.Sprintf(ChatMessagesKeyPrefix, roomID.String())
}

func (r *anonymousChatRepository) SaveMessage(ctx context.Context, roomID uuid.UUID, message *domain.AnonymousChatMessage) error {
	key := r.getMessagesKey(roomID)

	// Serialize the message to JSON
	messageJSON, err := json.Marshal(message)
	if err != nil {
		r.log.Error("Failed to marshal message", "error", err)
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Use the timestamp in milliseconds as the sort score
	score := float64(message.CreatedAt.UnixMilli())

	// Add to sorted set
	err = r.rdb.ZAdd(ctx, key, redis.Z{
		Score:  score,
		Member: messageJSON,
	}).Err()
	if err != nil {
		r.log.Error("Failed to save message to Redis", "error", err, "room_id", roomID)
		return fmt.Errorf("failed to save message: %w", err)
	}

	// Set the TTL on the key (6 hours)
	err = r.rdb.Expire(ctx, key, ChatTTL).Err()
	if err != nil {
		r.log.Warn("Failed to set TTL on chat key", "error", err)
		// Non-critical error, continue
	}

	return nil
}

func (r *anonymousChatRepository) GetMessages(ctx context.Context, roomID uuid.UUID, limit int) ([]*domain.AnonymousChatMessage, error) {
	key := r.getMessagesKey(roomID)

	// Fetch the latest N messages (newest to oldest)
	// Use ZREVRANGE to retrieve them in reverse order
	messagesJSON, err := r.rdb.ZRevRange(ctx, key, 0, int64(limit-1)).Result()
	if err != nil {
		if err == redis.Nil {
			return []*domain.AnonymousChatMessage{}, nil
		}
		r.log.Error("Failed to get messages from Redis", "error", err, "room_id", roomID)
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	messages := make([]*domain.AnonymousChatMessage, 0, len(messagesJSON))
	for _, msgJSON := range messagesJSON {
		var message domain.AnonymousChatMessage
		if err := json.Unmarshal([]byte(msgJSON), &message); err != nil {
			r.log.Warn("Failed to unmarshal message", "error", err)
			continue
		}
		messages = append(messages, &message)
	}

	// Reverse the slice to get chronological order (oldest to newest)
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

func (r *anonymousChatRepository) GetMessagesAfter(ctx context.Context, roomID uuid.UUID, after time.Time, limit int) ([]*domain.AnonymousChatMessage, error) {
	key := r.getMessagesKey(roomID)

	// Minimum score (timestamp in milliseconds)
	minScore := float64(after.UnixMilli())

	// Fetch messages created after the specified time
	messagesJSON, err := r.rdb.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min:    fmt.Sprintf("%.0f", minScore),
		Max:    "+inf",
		Offset: 0,
		Count:  int64(limit),
	}).Result()

	if err != nil {
		if err == redis.Nil {
			return []*domain.AnonymousChatMessage{}, nil
		}
		r.log.Error("Failed to get messages after time", "error", err, "room_id", roomID)
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	messages := make([]*domain.AnonymousChatMessage, 0, len(messagesJSON))
	for _, msgJSON := range messagesJSON {
		var message domain.AnonymousChatMessage
		if err := json.Unmarshal([]byte(msgJSON), &message); err != nil {
			r.log.Warn("Failed to unmarshal message", "error", err)
			continue
		}
		messages = append(messages, &message)
	}

	return messages, nil
}

func (r *anonymousChatRepository) DeleteMessage(ctx context.Context, roomID uuid.UUID, messageID string) error {
	key := r.getMessagesKey(roomID)

	// Fetch all messages and locate the target
	messagesJSON, err := r.rdb.ZRange(ctx, key, 0, -1).Result()
	if err != nil {
		if err == redis.Nil {
			return errors.New("message not found")
		}
		r.log.Error("Failed to get messages for deletion", "error", err)
		return fmt.Errorf("failed to get messages: %w", err)
	}

	// Look for the message with the matching ID
	for _, msgJSON := range messagesJSON {
		var message domain.AnonymousChatMessage
		if err := json.Unmarshal([]byte(msgJSON), &message); err != nil {
			continue
		}

		if message.ID == messageID {
			// Remove from sorted set
			err = r.rdb.ZRem(ctx, key, msgJSON).Err()
			if err != nil {
				r.log.Error("Failed to delete message from Redis", "error", err)
				return fmt.Errorf("failed to delete message: %w", err)
			}
			return nil
		}
	}

	return errors.New("message not found")
}

func (r *anonymousChatRepository) UpdateMessage(ctx context.Context, roomID uuid.UUID, message *domain.AnonymousChatMessage) error {
	key := r.getMessagesKey(roomID)

	// Fetch all messages and locate the target
	messagesJSON, err := r.rdb.ZRange(ctx, key, 0, -1).Result()
	if err != nil {
		if err == redis.Nil {
			return errors.New("message not found")
		}
		r.log.Error("Failed to get messages for update", "error", err)
		return fmt.Errorf("failed to get messages: %w", err)
	}

	// Look for the message with the matching ID
	for _, msgJSON := range messagesJSON {
		var oldMessage domain.AnonymousChatMessage
		if err := json.Unmarshal([]byte(msgJSON), &oldMessage); err != nil {
			continue
		}

		if oldMessage.ID == message.ID {
			// Remove the previous version
			err = r.rdb.ZRem(ctx, key, msgJSON).Err()
			if err != nil {
				r.log.Error("Failed to remove old message", "error", err)
				return fmt.Errorf("failed to update message: %w", err)
			}

			// Add the updated message
			newMessageJSON, err := json.Marshal(message)
			if err != nil {
				r.log.Error("Failed to marshal updated message", "error", err)
				return fmt.Errorf("failed to marshal message: %w", err)
			}

			score := float64(message.CreatedAt.UnixMilli())
			err = r.rdb.ZAdd(ctx, key, redis.Z{
				Score:  score,
				Member: newMessageJSON,
			}).Err()
			if err != nil {
				r.log.Error("Failed to add updated message", "error", err)
				return fmt.Errorf("failed to update message: %w", err)
			}

			// Refresh the TTL
			r.rdb.Expire(ctx, key, ChatTTL)

			return nil
		}
	}

	return errors.New("message not found")
}

func (r *anonymousChatRepository) RoomExists(ctx context.Context, roomID uuid.UUID) (bool, error) {
	key := r.getMessagesKey(roomID)
	exists, err := r.rdb.Exists(ctx, key).Result()
	if err != nil {
		r.log.Error("Failed to check room existence", "error", err)
		return false, err
	}
	return exists > 0, nil
}
