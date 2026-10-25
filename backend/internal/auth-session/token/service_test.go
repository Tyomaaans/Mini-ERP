package token

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestTokensEnforceAlgorithmIssuerAudienceAndExpiry(t *testing.T) {
	secret := "unit-test-secret-at-least-32-characters"
	service := NewJWTService(secret, time.Minute)
	pair, err := service.Generate("00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002", "refresh")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Parse(pair.AccessToken); err != nil {
		t.Fatal(err)
	}
	for _, change := range []string{"algorithm", "issuer", "audience", "expired", "missing_expiry"} {
		t.Run(change, func(t *testing.T) {
			claims := jwtClaims{
				SessionID: "00000000-0000-0000-0000-000000000002",
				RegisteredClaims: jwt.RegisteredClaims{
					Subject:   "00000000-0000-0000-0000-000000000001",
					Issuer:    "erp",
					IssuedAt:  jwt.NewNumericDate(time.Now()),
					Audience:  jwt.ClaimStrings{"erp-api"},
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
				},
			}
			var algorithm jwt.SigningMethod = jwt.SigningMethodHS256
			switch change {
			case "algorithm":
				algorithm = jwt.SigningMethodHS384
			case "issuer":
				claims.Issuer = "other"
			case "audience":
				claims.Audience = jwt.ClaimStrings{"other"}
			case "expired":
				claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Minute))
			case "missing_expiry":
				claims.ExpiresAt = nil
			}
			raw, err := jwt.NewWithClaims(algorithm, claims).SignedString([]byte(secret))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := service.Parse(raw); err == nil {
				t.Fatal("invalid token was accepted")
			}
		})
	}
}
