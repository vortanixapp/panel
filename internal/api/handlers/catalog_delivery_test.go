package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// Ссылка на архив работает без токена пользователя — её забирает агент на
// ноде. Значит вся защита держится на подписи и сроке: подделанная или
// просроченная ссылка обязана отвергаться.
func TestCatalogArchiveTokenRoundTrip(t *testing.T) {
	h := &Handler{jwtSecret: "секрет-подписи"}

	token, err := h.signCatalogArchive("tenant-1", catalogPlugins, "plugin-1")
	if err != nil {
		t.Fatalf("подпись не собралась: %v", err)
	}
	claims, err := h.parseCatalogArchiveToken(token)
	if err != nil {
		t.Fatalf("своя же ссылка отвергнута: %v", err)
	}
	if claims.Tenant != "tenant-1" || claims.Kind != catalogPlugins || claims.ID != "plugin-1" {
		t.Errorf("разобрано неверно: %+v", claims)
	}
}

func TestCatalogArchiveTokenRejectsTampering(t *testing.T) {
	h := &Handler{jwtSecret: "секрет-подписи"}
	token, _ := h.signCatalogArchive("tenant-1", catalogPlugins, "plugin-1")
	payload, sig, _ := strings.Cut(token, ".")

	// Подменяем арендатора, оставляя чужую подпись, — так выглядела бы попытка
	// вытащить архив другого клиента.
	var claims catalogArchiveToken
	body, _ := base64.RawURLEncoding.DecodeString(payload)
	_ = json.Unmarshal(body, &claims)
	claims.Tenant = "tenant-2"
	forged, _ := json.Marshal(claims)
	tampered := base64.RawURLEncoding.EncodeToString(forged) + "." + sig

	for name, bad := range map[string]string{
		"подменён арендатор": tampered,
		"чужая подпись":      payload + ".AAAA",
		"без подписи":        payload,
		"пусто":              "",
		"мусор":              "не.токен",
	} {
		if _, err := h.parseCatalogArchiveToken(bad); err == nil {
			t.Errorf("%s: ссылка принята, хотя не должна", name)
		}
	}

	// Чужой секрет — чужая подпись.
	other := &Handler{jwtSecret: "другой-секрет"}
	if _, err := other.parseCatalogArchiveToken(token); err == nil {
		t.Error("ссылка принята панелью с другим секретом")
	}
}

func TestCatalogArchiveTokenExpires(t *testing.T) {
	h := &Handler{jwtSecret: "секрет"}
	expired := catalogArchiveToken{
		Tenant: "t", Kind: catalogMaps, ID: "m",
		Exp: time.Now().Add(-time.Minute).Unix(),
	}
	body, _ := json.Marshal(expired)
	payload := base64.RawURLEncoding.EncodeToString(body)
	// Подписываем просроченную ссылку честно: отвергнуть её должен именно срок,
	// а не подпись.
	mac := hmac.New(sha256.New, []byte(h.jwtSecret))
	mac.Write([]byte(payload))
	token := payload + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	if _, err := h.parseCatalogArchiveToken(token); err == nil {
		t.Error("просроченная ссылка принята")
	} else if !strings.Contains(err.Error(), "просроч") {
		t.Errorf("отвергнута не по сроку, а по другой причине: %v", err)
	}

	fresh, err := h.signCatalogArchive("t", catalogMaps, "m")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.parseCatalogArchiveToken(fresh); err != nil {
		t.Fatalf("свежая ссылка должна приниматься: %v", err)
	}
}

// Имя в кэше зависит от размера: подменённый архив не должен спутаться со
// старым, который уже лежит на локации.
func TestCatalogCachePathChangesWithArchive(t *testing.T) {
	first := catalogCachePath(catalogPlugins, "id-1", 100)
	same := catalogCachePath(catalogPlugins, "id-1", 100)
	resized := catalogCachePath(catalogPlugins, "id-1", 200)
	other := catalogCachePath(catalogMaps, "id-1", 100)

	if first != same {
		t.Error("тот же архив должен давать то же имя, иначе кэш не переиспользуется")
	}
	if first == resized {
		t.Error("архив другого размера обязан получить другое имя")
	}
	if first == other {
		t.Error("плагины и карты не должны делить имя в кэше")
	}
	if !strings.HasPrefix(first, catalogPlugins+"/") {
		t.Errorf("путь в кэше без раздела: %s", first)
	}
}
