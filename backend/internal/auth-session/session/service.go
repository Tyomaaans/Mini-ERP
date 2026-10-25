package session

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"mini-erp/pkg"
)

// Interface

type SessionService interface {
	Create(ctx context.Context, userID, stamp string) (string, string, error)
	Rotate(ctx context.Context, refresh string) (string, string, string, error)
	Check(ctx context.Context, id string) (string, string, error)
	Revoke(ctx context.Context, id string) error
}

// Implementation

type sessionService struct {
	redisClient   *redis.Client
	refreshExpiry time.Duration
}

func NewSessionService(redisClient *redis.Client, refreshExpiry time.Duration) SessionService {
	return &sessionService{
		redisClient:   redisClient,
		refreshExpiry: refreshExpiry,
	}
}

func (s *sessionService) Create(ctx context.Context, userID, stamp string) (string, string, error) {
	id := uuid.NewString()
	refresh, err := secret(id)
	if err != nil {
		return "", "", err
	}

	_, err = s.redisClient.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.HSet(ctx, "session:"+id, "user_id", userID, "stamp", stamp, "refresh", digest(refresh))
		pipe.Expire(ctx, "session:"+id, s.refreshExpiry)
		return nil
	})
	return id, refresh, err
}

func (s *sessionService) Rotate(ctx context.Context, refresh string) (string, string, string, error) {
	parts := strings.Split(refresh, ".")
	if len(parts) != 2 || pkg.ValidateID(parts[0]) != nil {
		return "", "", "", pkg.ErrUnauthorized
	}

	newToken, err := secret(parts[0])
	if err != nil {
		return "", "", "", err
	}

	script := redis.NewScript(`
		local current = redis.call('HGET', KEYS[1], 'refresh')
		if not current or current ~= ARGV[1] then return '' end
		redis.call('HSET', KEYS[1], 'refresh', ARGV[2])
		return redis.call('HGET', KEYS[1], 'user_id')`)
	userID, err := script.Run(ctx, s.redisClient, []string{"session:" + parts[0]}, digest(refresh), digest(newToken)).Text()
	if err != nil {
		return "", "", "", err
	}

	if userID == "" {
		return "", "", "", pkg.ErrUnauthorized
	}

	return userID, parts[0], newToken, nil
}

func (s *sessionService) Check(ctx context.Context, id string) (string, string, error) {
	values, err := s.redisClient.HGetAll(ctx, "session:"+id).Result()
	if err != nil {
		return "", "", err
	}

	if values["user_id"] == "" {
		return "", "", pkg.ErrUnauthorized
	}

	return values["user_id"], values["stamp"], nil
}

func (s *sessionService) Revoke(ctx context.Context, id string) error {
	return s.redisClient.Del(ctx, "session:"+id).Err()
}

func CredentialStamp(hash string, updatedAt time.Time) string {
	return digest(hash + "|" + updatedAt.UTC().Format(time.RFC3339Nano))
}

// Internal Helper

func secret(id string) (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}

	return id + "." + base64.RawURLEncoding.EncodeToString(value), nil
}

func digest(raw string) string {
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}
