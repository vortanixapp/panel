package paneljwt

import (
	"fmt"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"

	"github.com/vortanixapp/panel/pkg/settingsreg"
)

const (
	typeAccess  = "access"
	typeRefresh = "refresh"
)

type Claims struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	SessionID string `json:"session_id,omitempty"`
	TokenType string `json:"typ,omitempty"`
	jwtlib.RegisteredClaims
}

type refreshClaims struct {
	UserID    string `json:"user_id,omitempty"`
	SessionID string `json:"sid,omitempty"`
	TokenType string `json:"typ,omitempty"`
	jwtlib.RegisteredClaims
}

type Manager struct {
	secret []byte
}

func New(secret string, accessMin, refreshDays int) *Manager {
	if accessMin > 0 {
		settingsreg.AuthAccessTTL.SetDefaultInt(accessMin)
	}
	if refreshDays > 0 {
		settingsreg.AuthRefreshTTLDefault.SetDefaultInt(refreshDays)
	}
	return &Manager{secret: []byte(secret)}
}

func (m *Manager) AccessToken(userID, email, role, sessionID string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:    userID,
		Email:     email,
		Role:      role,
		SessionID: sessionID,
		TokenType: typeAccess,
		RegisteredClaims: jwtlib.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwtlib.NewNumericDate(now.Add(settingsreg.AuthAccessTTL.Duration())),
			IssuedAt:  jwtlib.NewNumericDate(now),
		},
	}
	return jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims).SignedString(m.secret)
}

func (m *Manager) RefreshToken(userID, sessionID, tokenID string) (string, error) {
	return m.RefreshTokenWithTTL(userID, sessionID, tokenID, m.DefaultRefreshTTL())
}

func (m *Manager) RefreshTokenWithTTL(userID, sessionID, tokenID string, ttl time.Duration) (string, error) {
	if ttl <= 0 {
		ttl = m.DefaultRefreshTTL()
	}
	now := time.Now()
	claims := refreshClaims{
		SessionID: sessionID,
		TokenType: typeRefresh,
		RegisteredClaims: jwtlib.RegisteredClaims{
			Subject:   userID,
			ID:        tokenID,
			ExpiresAt: jwtlib.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwtlib.NewNumericDate(now),
		},
	}
	return jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims).SignedString(m.secret)
}

func (m *Manager) DefaultRefreshTTL() time.Duration {
	return settingsreg.AuthRefreshTTLDefault.Duration()
}

func (m *Manager) ParseAccess(tokenString string) (*Claims, error) {
	token, err := jwtlib.ParseWithClaims(tokenString, &Claims{}, func(t *jwtlib.Token) (any, error) {
		if t.Method != jwtlib.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	if claims.UserID == "" || (claims.TokenType != "" && claims.TokenType != typeAccess) {
		return nil, fmt.Errorf("not an access token")
	}
	return claims, nil
}

func (m *Manager) ParseRefresh(tokenString string) (userID, sessionID string, err error) {
	userID, sessionID, _, _, err = m.ParseRefreshDetails(tokenString)
	return userID, sessionID, err
}

func (m *Manager) ParseRefreshDetails(tokenString string) (userID, sessionID, tokenID string, expiresAt time.Time, err error) {
	token, err := jwtlib.ParseWithClaims(tokenString, &refreshClaims{}, func(t *jwtlib.Token) (any, error) {
		if t.Method != jwtlib.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return "", "", "", time.Time{}, err
	}
	claims, ok := token.Claims.(*refreshClaims)
	if !ok || !token.Valid {
		return "", "", "", time.Time{}, fmt.Errorf("invalid token")
	}
	if claims.UserID != "" || (claims.TokenType != "" && claims.TokenType != typeRefresh) {
		return "", "", "", time.Time{}, fmt.Errorf("not a refresh token")
	}
	expiresAt = time.Time{}
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	}
	sessionID = claims.SessionID
	if sessionID == "" {
		sessionID = claims.ID
	}
	return claims.Subject, sessionID, claims.ID, expiresAt, nil
}
