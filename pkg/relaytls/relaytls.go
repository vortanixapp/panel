package relaytls

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"net/url"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vortanixapp/panel/pkg/secretbox"
)

const (
	SettingKey   = "relay.tls"
	certValidity = 10 * 365 * 24 * time.Hour
	renewBefore  = 30 * 24 * time.Hour
)

type stored struct {
	CertPEM string   `json:"cert"`
	KeyPEM  string   `json:"key"`
	Pin     string   `json:"pin"`
	Hosts   []string `json:"hosts"`
}

type Material struct {
	Certificate tls.Certificate
	Pin         string
	Hosts       []string
}

func HostsFrom(values ...string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			host := strings.TrimSpace(part)
			if host == "" {
				continue
			}
			if strings.Contains(host, "://") {
				if u, err := url.Parse(host); err == nil && u.Host != "" {
					host = u.Hostname()
				}
			} else if h, _, err := net.SplitHostPort(host); err == nil {
				host = h
			}
			host = strings.Trim(host, "[]")
			if host == "" || seen[host] {
				continue
			}
			seen[host] = true
			out = append(out, host)
		}
	}
	sort.Strings(out)
	return out
}

func Ensure(ctx context.Context, db *pgxpool.Pool, box *secretbox.Box, hosts []string) (*Material, error) {
	if db == nil {
		return nil, errors.New("нет доступа к базе")
	}
	if len(hosts) == 0 {
		return nil, errors.New("не задан адрес, по которому ноды видят relay")
	}

	if current, err := load(ctx, db, box); err == nil && current != nil && fits(current, hosts) {
		return current, nil
	}

	material, raw, err := generate(hosts)
	if err != nil {
		return nil, err
	}
	sealed := raw.KeyPEM
	if box != nil {
		if enc, encErr := box.Encrypt(raw.KeyPEM); encErr == nil {
			sealed = enc
		}
	}
	payload, err := json.Marshal(stored{CertPEM: raw.CertPEM, KeyPEM: sealed, Pin: raw.Pin, Hosts: hosts})
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO core.tenant_settings (key, value) VALUES ($1, $2::jsonb)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()
	`, SettingKey, payload); err != nil {
		return nil, err
	}
	return material, nil
}

func Pin(ctx context.Context, db *pgxpool.Pool) string {
	var raw []byte
	err := db.QueryRow(ctx, `SELECT value FROM core.tenant_settings WHERE key = $1`, SettingKey).Scan(&raw)
	if err != nil || len(raw) == 0 {
		return ""
	}
	var st stored
	if json.Unmarshal(raw, &st) != nil {
		return ""
	}
	return st.Pin
}

func load(ctx context.Context, db *pgxpool.Pool, box *secretbox.Box) (*Material, error) {
	var raw []byte
	err := db.QueryRow(ctx, `SELECT value FROM core.tenant_settings WHERE key = $1`, SettingKey).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var st stored
	if err := json.Unmarshal(raw, &st); err != nil {
		return nil, err
	}
	if st.CertPEM == "" || st.KeyPEM == "" {
		return nil, nil
	}
	keyPEM := st.KeyPEM
	if box != nil {
		keyPEM = box.MustDecrypt(keyPEM)
	}
	cert, err := tls.X509KeyPair([]byte(st.CertPEM), []byte(keyPEM))
	if err != nil {
		return nil, nil
	}
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, nil
	}
	if time.Now().Add(renewBefore).After(leaf.NotAfter) {
		return nil, nil
	}
	cert.Leaf = leaf
	return &Material{Certificate: cert, Pin: st.Pin, Hosts: st.Hosts}, nil
}

func fits(m *Material, hosts []string) bool {
	for _, host := range hosts {
		if !slices.Contains(m.Hosts, host) {
			return false
		}
	}
	return true
}

func generate(hosts []string) (*Material, *stored, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, err
	}
	template := x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "vortanix-relay"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(certValidity),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	for _, host := range hosts {
		if ip := net.ParseIP(host); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
			continue
		}
		template.DNSNames = append(template.DNSNames, host)
	}

	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return nil, nil, err
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, nil, err
	}
	certPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	keyPEM := string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}))

	cert, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	if err != nil {
		return nil, nil, err
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, nil, err
	}
	cert.Leaf = leaf
	pin := PinOf(leaf)
	return &Material{Certificate: cert, Pin: pin, Hosts: hosts},
		&stored{CertPEM: certPEM, KeyPEM: keyPEM, Pin: pin, Hosts: hosts}, nil
}

func PinOf(cert *x509.Certificate) string {
	sum := sha256.Sum256(cert.RawSubjectPublicKeyInfo)
	return "sha256:" + base64.StdEncoding.EncodeToString(sum[:])
}
