package token

import "mini-erp/internal/auth-session/domain"

func ToTokenResponse(e *domain.TokenPairEntity) *TokenResponse {
	return &TokenResponse{
		AccessToken:  e.AccessToken,
		RefreshToken: e.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    e.ExpiresIn,
	}
}
