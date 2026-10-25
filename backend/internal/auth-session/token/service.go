package token

import (
	"time"

	"github.com/golang-jwt/jwt/v5"

	"mini-erp/internal/auth-session/domain"
	"mini-erp/pkg"
)

// Interface

type JWTService interface {
	Generate(userID, sessionID, refresh string) (*domain.TokenPairEntity, error)
	Parse(raw string) (*domain.ClaimsEntity, error)
}

// Implementation

type jwtClaims struct {
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

type jwtService struct {
	secretKey         []byte
	accessTokenExpiry time.Duration
}

func NewJWTService(secretKey string, accessTokenExpiry time.Duration) JWTService {
	return &jwtService{
		secretKey:         []byte(secretKey),
		accessTokenExpiry: accessTokenExpiry,
	}
}

func (s *jwtService) Generate(userID, sessionID, refresh string) (*domain.TokenPairEntity, error) {
	now := time.Now().UTC()
	claims := jwtClaims{
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "erp",
			Subject:   userID,
			Audience:  jwt.ClaimStrings{"erp-api"},
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	access, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secretKey)
	if err != nil {
		return nil, err
	}

	return &domain.TokenPairEntity{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int64(s.accessTokenExpiry.Seconds()),
	}, nil
}

func (s *jwtService) Parse(raw string) (*domain.ClaimsEntity, error) {
	claims := &jwtClaims{}
	parsed, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		return s.secretKey, nil
	},
		jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer("erp"), jwt.WithAudience("erp-api"), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if err != nil || !parsed.Valid || claims.IssuedAt == nil || claims.ExpiresAt == nil || pkg.ValidateID(claims.Subject) != nil || pkg.ValidateID(claims.SessionID) != nil {
		return nil, pkg.ErrUnauthorized
	}

	return &domain.ClaimsEntity{
		UserID:    claims.Subject,
		SessionID: claims.SessionID,
		ExpiresAt: claims.ExpiresAt.Time,
		IssuedAt:  claims.IssuedAt.Time,
	}, nil
}
