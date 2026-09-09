package paneljwt

import (
	"fmt"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	TenantID   string `json:"tenant_id"`
	TenantSlug string `json:"tenant_slug"`
	UserID     string `json:"user_id"`
	Email      string `json:"email"`
	Role       string `json:"role"`
	SessionID  string `json:"session_id,omitempty"`
	jwtlib.RegisteredClaims
}

type Manager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func New(secret string, accessMin, refreshDays int) *Manager {
	return &Manager{
		secret:     []byte(secret),
		accessTTL:  time.Duration(accessMin) * time.Minute,
		refreshTTL: time.Duration(refreshDays) * 24 * time.Hour,
	}
}

func (m *Manager) AccessToken(tenantID, tenantSlug, userID, email, role, sessionID string) (string, error) {
	now := time.Now()
	claims := Claims{
		TenantID:   tenantID,
		TenantSlug: tenantSlug,
		UserID:     userID,
		Email:      email,
		Role:       role,
		SessionID:  sessionID,
		RegisteredClaims: jwtlib.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwtlib.NewNumericDate(now.Add(m.accessTTL)),
			IssuedAt:  jwtlib.NewNumericDate(now),
		},
	}
	return jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims).SignedString(m.secret)
}

func (m *Manager) RefreshToken(tenantID, userID, sessionID string) (string, error) {
	return m.RefreshTokenWithTTL(tenantID, userID, sessionID, m.refreshTTL)
}

func (m *Manager) RefreshTokenWithTTL(tenantID, userID, sessionID string, ttl time.Duration) (string, error) {
	if ttl <= 0 {
		ttl = m.refreshTTL
	}
	now := time.Now()
	claims := jwtlib.RegisteredClaims{
		Subject:   userID,
		Audience:  jwtlib.ClaimStrings{tenantID},
		ID:        sessionID,
		ExpiresAt: jwtlib.NewNumericDate(now.Add(ttl)),
		IssuedAt:  jwtlib.NewNumericDate(now),
	}
	return jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims).SignedString(m.secret)
}

func (m *Manager) DefaultRefreshTTL() time.Duration {
	return m.refreshTTL
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
	return claims, nil
}

func (m *Manager) ParseRefresh(tokenString string) (tenantID, userID, sessionID string, err error) {
	tenantID, userID, sessionID, _, err = m.ParseRefreshDetails(tokenString)
	return tenantID, userID, sessionID, err
}

func (m *Manager) ParseRefreshDetails(tokenString string) (tenantID, userID, sessionID string, expiresAt time.Time, err error) {
	token, err := jwtlib.ParseWithClaims(tokenString, &jwtlib.RegisteredClaims{}, func(t *jwtlib.Token) (any, error) {
		if t.Method != jwtlib.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return "", "", "", time.Time{}, err
	}
	claims, ok := token.Claims.(*jwtlib.RegisteredClaims)
	if !ok || !token.Valid {
		return "", "", "", time.Time{}, fmt.Errorf("invalid token")
	}
	if len(claims.Audience) == 0 {
		return "", "", "", time.Time{}, fmt.Errorf("missing tenant")
	}
	expiresAt = time.Time{}
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	}
	return claims.Audience[0], claims.Subject, claims.ID, expiresAt, nil
}
