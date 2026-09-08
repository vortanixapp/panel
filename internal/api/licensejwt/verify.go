package licensejwt

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

type PlanLimits struct {
	MaxNodes    int `json:"max_nodes"`
	MaxServers  int `json:"max_servers"`
	MaxAdmins   int `json:"max_admins"`
	APIRPM      int `json:"api_rpm"`
	MaxConsoles int `json:"max_consoles"`
}

type Claims struct {
	TenantSlug     string     `json:"tenant_slug"`
	InstallationID string     `json:"installation_id"`
	Plan           string     `json:"plan"`
	Limits         PlanLimits `json:"limits"`
	Status         string     `json:"status"`
	Revision       int64      `json:"revision"`
	LicenseExpires *int64     `json:"license_expires_at,omitempty"`
	GraceSeconds   int        `json:"grace_seconds"`
	KeyHint        string     `json:"key_hint"`
	jwtlib.RegisteredClaims
}

func (c *Claims) LicenseExpiresAt() *time.Time {
	if c.LicenseExpires == nil {
		return nil
	}
	t := time.Unix(*c.LicenseExpires, 0)
	return &t
}

type StateClaims struct {
	Status         string       `json:"status"`
	Revision       int64        `json:"revision"`
	Plan           string       `json:"plan"`
	LicenseExpires *int64       `json:"license_expires_at,omitempty"`
	MinPollSeconds int          `json:"min_poll_seconds"`
	Nonce          string       `json:"nonce"`
	Update         *UpdateState `json:"update,omitempty"`
	jwtlib.RegisteredClaims
}

type UpdateState struct {
	TargetVersion  string `json:"target_version"`
	MandatoryAfter *int64 `json:"mandatory_after,omitempty"`
	Notes          string `json:"notes,omitempty"`
	// AutoUpdate — ставить ли назначенную версию без участия владельца.
	AutoUpdate bool `json:"auto_update"`
}

func (u *UpdateState) MandatoryAt() *time.Time {
	if u == nil || u.MandatoryAfter == nil {
		return nil
	}
	t := time.Unix(*u.MandatoryAfter, 0)
	return &t
}

type Verifier struct {
	publicKey ed25519.PublicKey
	issuer    string
	pem       string
}

func LoadVerifier(publicKeyPath, issuer string) (*Verifier, error) {
	pemBytes, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, err
	}
	return LoadVerifierFromPEM(string(pemBytes), issuer)
}

func LoadVerifierFromPEM(pemText, issuer string) (*Verifier, error) {
	block, _ := pem.Decode([]byte(pemText))
	if block == nil {
		return nil, fmt.Errorf("invalid public key pem")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	edPub, ok := pub.(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not ed25519 public key")
	}
	return &Verifier{publicKey: edPub, issuer: issuer, pem: pemText}, nil
}

func (v *Verifier) PEM() string { return v.pem }

func (v *Verifier) Parse(tokenString string) (*Claims, error) {
	return v.parse(tokenString, 0)
}

var ErrTokenTooOld = errors.New("license token expired beyond grace")

func (v *Verifier) ParseAllowExpired(tokenString string, maxAge time.Duration) (*Claims, error) {
	return v.parse(tokenString, maxAge)
}

func (v *Verifier) parse(tokenString string, allowExpiredFor time.Duration) (*Claims, error) {
	var claims Claims
	opts := []jwtlib.ParserOption{
		jwtlib.WithValidMethods([]string{jwtlib.SigningMethodEdDSA.Alg()}),
	}
	if allowExpiredFor > 0 {
		opts = append(opts, jwtlib.WithoutClaimsValidation())
	}

	token, err := jwtlib.NewParser(opts...).ParseWithClaims(tokenString, &claims, func(t *jwtlib.Token) (any, error) {
		return v.publicKey, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	if claims.Issuer != v.issuer {
		return nil, fmt.Errorf("invalid issuer")
	}
	if allowExpiredFor > 0 {
		if claims.ExpiresAt == nil {
			return nil, fmt.Errorf("token has no expiry")
		}
		if time.Since(claims.ExpiresAt.Time) > allowExpiredFor {
			return nil, ErrTokenTooOld
		}
	}
	return &claims, nil
}

func (v *Verifier) ParseState(tokenString string) (*StateClaims, error) {
	var claims StateClaims
	token, err := jwtlib.NewParser(
		jwtlib.WithValidMethods([]string{jwtlib.SigningMethodEdDSA.Alg()}),
	).ParseWithClaims(tokenString, &claims, func(t *jwtlib.Token) (any, error) {
		return v.publicKey, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid state token")
	}
	if claims.Issuer != v.issuer {
		return nil, fmt.Errorf("invalid issuer")
	}
	return &claims, nil
}

func (c *StateClaims) LicenseExpiresAt() *time.Time {
	if c.LicenseExpires == nil {
		return nil
	}
	t := time.Unix(*c.LicenseExpires, 0)
	return &t
}
