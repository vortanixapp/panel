package handlers

import (
	"context"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/vortanixapp/panel/pkg/regions"
)

const (
	profileFieldMax  = 200
	profilePostalMax = 20
)

type profileExtra struct {
	MiddleName  string `json:"middle_name"`
	Country     string `json:"country"`
	AddressLine string `json:"address_line"`
	City        string `json:"city"`
	Region      string `json:"region"`
	PostalCode  string `json:"postal_code"`
}

func (p *profileExtra) normalize() {
	p.MiddleName = strings.TrimSpace(p.MiddleName)
	p.Country = regions.NormalizeCountry(p.Country)
	p.AddressLine = strings.TrimSpace(p.AddressLine)
	p.City = strings.TrimSpace(p.City)
	p.Region = strings.TrimSpace(p.Region)
	p.PostalCode = strings.TrimSpace(p.PostalCode)
}

func (p profileExtra) validate(requireCountry bool) (code, message string) {
	switch {
	case requireCountry && p.Country == "":
		return "country_required", "Укажите страну"
	case p.Country != "" && !regions.ValidCountry(p.Country):
		return "invalid_country", "Страна указана неверно"
	case utf8.RuneCountInString(p.MiddleName) > profileFieldMax,
		utf8.RuneCountInString(p.AddressLine) > profileFieldMax,
		utf8.RuneCountInString(p.City) > profileFieldMax,
		utf8.RuneCountInString(p.Region) > profileFieldMax:
		return "invalid_profile", "Поля профиля — не длиннее 200 символов"
	case utf8.RuneCountInString(p.PostalCode) > profilePostalMax:
		return "invalid_profile", "Почтовый индекс — не длиннее 20 символов"
	}
	return "", ""
}

type sqlExecer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func insertUserProfile(ctx context.Context, db sqlExecer, userID, firstName, lastName string, extra profileExtra) error {
	displayName := strings.TrimSpace(firstName + " " + lastName)
	if displayName == "" && extra == (profileExtra{}) {
		return nil
	}
	_, err := db.Exec(ctx, `
		INSERT INTO core.user_profiles
			(user_id, display_name, first_name, last_name, middle_name, country, address_line, city, region, postal_code, updated_at)
		VALUES ($1, NULLIF($2, ''), NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''),
		        NULLIF($7, ''), NULLIF($8, ''), NULLIF($9, ''), NULLIF($10, ''), now())
		ON CONFLICT (user_id) DO NOTHING
	`, userID, displayName, firstName, lastName, extra.MiddleName, extra.Country,
		extra.AddressLine, extra.City, extra.Region, extra.PostalCode)
	return err
}

func (h *Handler) userCountry(ctx context.Context, userID string) string {
	var country string
	_ = h.dbOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(country, '') FROM core.user_profiles WHERE user_id = $1
	`, userID).Scan(&country)
	return country
}

func (h *Handler) refuseIncompleteProfile(ctx context.Context, w http.ResponseWriter, userID string) bool {
	if h.userCountry(ctx, userID) != "" {
		return false
	}
	writeCodedError(w, http.StatusConflict, "profile_incomplete",
		"Укажите страну в профиле: от неё зависят налоги и документы")
	return true
}

func profileExtraFromBody(body map[string]any) (profileExtra, map[string]bool) {
	var x profileExtra
	present := map[string]bool{}
	read := func(key string, dst *string) {
		if v, ok := body[key]; ok {
			present[key] = true
			if text, isText := v.(string); isText {
				*dst = text
			}
		}
	}
	read("middle_name", &x.MiddleName)
	read("country", &x.Country)
	read("address_line", &x.AddressLine)
	read("city", &x.City)
	read("region", &x.Region)
	read("postal_code", &x.PostalCode)
	x.normalize()
	return x, present
}

func (h *Handler) saveProfileExtra(ctx context.Context, userID string, x profileExtra, present map[string]bool) {
	if len(present) == 0 {
		return
	}
	columns := map[string]string{
		"middle_name":  x.MiddleName,
		"country":      x.Country,
		"address_line": x.AddressLine,
		"city":         x.City,
		"region":       x.Region,
		"postal_code":  x.PostalCode,
	}
	for key, value := range columns {
		if !present[key] {
			continue
		}
		_, _ = h.dbOf(ctx).Exec(ctx, `
			INSERT INTO core.user_profiles (user_id, `+key+`, updated_at) VALUES ($1, NULLIF($2, ''), now())
			ON CONFLICT (user_id) DO UPDATE SET `+key+` = NULLIF($2, ''), updated_at = now()
		`, userID, value)
	}
}

func (h *Handler) saveProfileExtraIfEmpty(ctx context.Context, userID string, x profileExtra, present map[string]bool) {
	columns := map[string]string{
		"country":      x.Country,
		"address_line": x.AddressLine,
		"city":         x.City,
		"region":       x.Region,
		"postal_code":  x.PostalCode,
	}
	for key, value := range columns {
		if !present[key] {
			continue
		}
		_, _ = h.dbOf(ctx).Exec(ctx, `
			UPDATE core.user_profiles SET `+key+` = $2, updated_at = now()
			WHERE user_id = $1 AND COALESCE(`+key+`, '') = ''
		`, userID, value)
	}
}
