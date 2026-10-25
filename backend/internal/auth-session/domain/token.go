package domain

import "time"

type ClaimsEntity struct {
	UserID    string
	SessionID string
	ExpiresAt time.Time
	IssuedAt  time.Time
}

type TokenPairEntity struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}
