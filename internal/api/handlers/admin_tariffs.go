package handlers

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type tariffRow struct {
	ID             string
	TenantID       string
	NodeID         *string
	GameID         *string
	Name           string
	Slug           string
	BillingType    string
	PriceMonthly   float64
	Currency       string
	SlotsMin       *int
	SlotsMax       *int
	CPUCores       *float64
	CPUShares      *int
	RAMMb          *int
	DiskMb         *int
	RentalPeriods  []byte
	RenewalPeriods []byte
	Discounts      []byte
	Position       int
	Active         bool
	Meta           []byte
	CreatedAt      time.Time
	NodeName       *string
	GameName       *string
}

func tariffMetaDefaults() map[string]any {
	return map[string]any{
		"mysql_engine":       nil,
		"mysql_instance_key": nil,
		"price_per_slot":     float64(0),
		"price_per_cpu_core": float64(0),
		"price_per_ram_gb":   float64(0),
		"price_per_disk_gb":  float64(0),
		"base_price_monthly": float64(0),
		"cpu_min":            nil,
		"cpu_max":            nil,
		"cpu_step":           nil,
		"ram_min":            nil,
		"ram_max":            nil,
		"ram_step":           nil,
		"disk_min":           nil,
		"disk_max":           nil,
		"disk_step":          nil,
		"allow_antiddos":     false,
		"antiddos_price":     float64(0),
	}
}

func parseTariffMeta(raw []byte) map[string]any {
	out := tariffMetaDefaults()
	if len(raw) > 0 {
		var m map[string]any
		if json.Unmarshal(raw, &m) == nil && m != nil {
			for k, v := range m {
				out[k] = v
			}
		}
	}
	return out
}

func metaFloat(meta map[string]any, key string) float64 {
	if v, ok := meta[key]; ok && v != nil {
		switch n := v.(type) {
		case float64:
			return n
		case int:
			return float64(n)
		case json.Number:
			f, _ := n.Float64()
			return f
		default:
			f, _ := strconv.ParseFloat(fmt.Sprint(v), 64)
			return f
		}
	}
	return 0
}

func metaIntPtr(meta map[string]any, key string) any {
	if v, ok := meta[key]; ok && v != nil && fmt.Sprint(v) != "" {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		default:
			i, err := strconv.Atoi(fmt.Sprint(v))
			if err == nil {
				return i
			}
		}
	}
	return nil
}

func metaBool(meta map[string]any, key string) bool {
	if v, ok := meta[key]; ok {
		switch b := v.(type) {
		case bool:
			return b
		case string:
			return b == "true" || b == "1"
		case float64:
			return b != 0
		}
	}
	return false
}

func jsonIntSlice(raw []byte) []int {
	out := []int{}
	if len(raw) == 0 {
		return out
	}
	var arr []any
	if json.Unmarshal(raw, &arr) != nil {
		return out
	}
	for _, item := range arr {
		switch n := item.(type) {
		case float64:
			out = append(out, int(n))
		case int:
			out = append(out, n)
		default:
			if i, err := strconv.Atoi(fmt.Sprint(item)); err == nil {
				out = append(out, i)
			}
		}
	}
	return out
}

func tariffToLegacyJSON(row *tariffRow) map[string]any {
	meta := parseTariffMeta(row.Meta)
	ramGB := 1
	if row.RAMMb != nil && *row.RAMMb > 0 {
		ramGB = int(math.Round(float64(*row.RAMMb) / 1024))
		if ramGB < 1 {
			ramGB = 1
		}
	}
	diskGB := 10
	if row.DiskMb != nil && *row.DiskMb > 0 {
		diskGB = int(math.Round(float64(*row.DiskMb) / 1024))
		if diskGB < 1 {
			diskGB = 1
		}
	}
	cpuCores := 0
	if row.CPUCores != nil {
		cpuCores = int(*row.CPUCores)
	}
	minSlots := 1
	maxSlots := 100
	if row.SlotsMin != nil {
		minSlots = *row.SlotsMin
	}
	if row.SlotsMax != nil {
		maxSlots = *row.SlotsMax
	}
	locationID := ""
	if row.NodeID != nil {
		locationID = *row.NodeID
	}
	gameID := ""
	if row.GameID != nil {
		gameID = *row.GameID
	}
	var location any = nil
	if row.NodeName != nil && *row.NodeName != "" {
		location = map[string]string{"name": *row.NodeName}
	}
	var game any = nil
	if row.GameName != nil && *row.GameName != "" {
		game = map[string]string{"name": *row.GameName}
	}
	mysqlEngine := metaString(meta, "mysql_engine")
	var discounts any = map[string]any{}
	if len(row.Discounts) > 0 {
		_ = json.Unmarshal(row.Discounts, &discounts)
	}
	if discounts == nil {
		discounts = map[string]any{}
	}
	return map[string]any{
		"id":                 row.ID,
		"name":               row.Name,
		"slug":               row.Slug,
		"location_id":        locationID,
		"game_id":            gameID,
		"billing_type":       row.BillingType,
		"mysql_engine":       mysqlEngine,
		"mysql_instance_key": metaString(meta, "mysql_instance_key"),
		"price_per_slot":     metaFloat(meta, "price_per_slot"),
		"price_per_cpu_core": metaFloat(meta, "price_per_cpu_core"),
		"price_per_ram_gb":   metaFloat(meta, "price_per_ram_gb"),
		"price_per_disk_gb":  metaFloat(meta, "price_per_disk_gb"),
		"base_price_monthly": metaFloat(meta, "base_price_monthly"),
		"cpu_min":            metaIntPtr(meta, "cpu_min"),
		"cpu_max":            metaIntPtr(meta, "cpu_max"),
		"cpu_step":           metaIntPtr(meta, "cpu_step"),
		"ram_min":            metaIntPtr(meta, "ram_min"),
		"ram_max":            metaIntPtr(meta, "ram_max"),
		"ram_step":           metaIntPtr(meta, "ram_step"),
		"disk_min":           metaIntPtr(meta, "disk_min"),
		"disk_max":           metaIntPtr(meta, "disk_max"),
		"disk_step":          metaIntPtr(meta, "disk_step"),
		"allow_antiddos":     metaBool(meta, "allow_antiddos"),
		"antiddos_price":     metaFloat(meta, "antiddos_price"),
		"min_slots":          minSlots,
		"max_slots":          maxSlots,
		"cpu_cores":          cpuCores,
		"cpu_shares":         row.CPUShares,
		"ram_gb":             ramGB,
		"disk_gb":            diskGB,
		"rental_periods":     jsonIntSlice(row.RentalPeriods),
		"renewal_periods":    jsonIntSlice(row.RenewalPeriods),
		"discounts":          discounts,
		"position":           row.Position,
		"is_available":       row.Active,
		"active":             row.Active,
		"price_monthly":      row.PriceMonthly,
		"currency":           row.Currency,
		"location":           location,
		"game":               game,
		"created_at":         row.CreatedAt.Format(time.RFC3339),
		"updated_at":         row.CreatedAt.Format(time.RFC3339),
	}
}

const tariffSelectSQL = `
	SELECT t.id::text, t.tenant_id::text, t.node_id::text, t.game_id::text,
		t.name, t.slug, t.billing_type, t.price_monthly::float8, t.currency,
		t.slots_min, t.slots_max, t.cpu_cores::float8, t.cpu_shares,
		t.ram_mb, t.disk_mb, t.rental_periods, t.renewal_periods, t.discounts,
		t.position, t.active, t.meta, t.created_at,
		n.name AS node_name, g.name AS game_name
	FROM core.tariffs t
	LEFT JOIN core.nodes n ON n.id = t.node_id
	LEFT JOIN core.games g ON g.id = t.game_id
`

func scanTariffRow(rows pgx.Row) (*tariffRow, error) {
	var row tariffRow
	var nodeName, gameName *string
	err := rows.Scan(
		&row.ID, &row.TenantID, &row.NodeID, &row.GameID,
		&row.Name, &row.Slug, &row.BillingType, &row.PriceMonthly, &row.Currency,
		&row.SlotsMin, &row.SlotsMax, &row.CPUCores, &row.CPUShares,
		&row.RAMMb, &row.DiskMb, &row.RentalPeriods, &row.RenewalPeriods, &row.Discounts,
		&row.Position, &row.Active, &row.Meta, &row.CreatedAt,
		&nodeName, &gameName,
	)
	if err != nil {
		return nil, err
	}
	row.NodeName = nodeName
	row.GameName = gameName
	return &row, nil
}

func (h *Handler) loadTariffRow(r *http.Request, tenantID, id string) (*tariffRow, error) {
	q := tariffSelectSQL + ` WHERE t.id = $1 AND t.tenant_id = $2`
	return scanTariffRow(h.readerOf(r.Context()).QueryRow(r.Context(), q, id, tenantID))
}

func (h *Handler) ListTariffsAdmin(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	query := r.URL.Query()
	page, _ := strconv.Atoi(query.Get("page"))
	if page < 1 {
		page = 1
	}
	perPage := 10
	if v, err := strconv.Atoi(query.Get("per_page")); err == nil && v > 0 {
		perPage = v
		if perPage > 500 {
			perPage = 500
		}
	}

	where := []string{"t.tenant_id = $1"}
	args := []any{claims.TenantID}
	addFilter := func(clause string, value any) {
		args = append(args, value)
		where = append(where, fmt.Sprintf(clause, len(args)))
	}
	if s := strings.TrimSpace(query.Get("search")); s != "" {
		addFilter("t.name ILIKE $%d", "%"+s+"%")
	}
	if v := strings.TrimSpace(query.Get("location_id")); v != "" {
		addFilter("t.node_id = $%d", v)
	}
	if v := strings.TrimSpace(query.Get("game_id")); v != "" {
		addFilter("t.game_id = $%d", v)
	}
	if v := strings.TrimSpace(query.Get("billing_type")); v == "slots" || v == "resources" {
		addFilter("t.billing_type = $%d", v)
	}
	switch strings.TrimSpace(query.Get("available")) {
	case "1", "true":
		where = append(where, "t.active = true")
	case "0", "false":
		where = append(where, "t.active = false")
	}
	whereSQL := " WHERE " + strings.Join(where, " AND ")

	var total int
	_ = h.readerOf(r.Context()).QueryRow(r.Context(),
		`SELECT COUNT(*) FROM core.tariffs t`+whereSQL, args...,
	).Scan(&total)
	lastPage := int(math.Max(1, math.Ceil(float64(total)/float64(perPage))))
	if page > lastPage {
		page = lastPage
	}
	offset := (page - 1) * perPage

	rows, err := h.readerOf(r.Context()).Query(r.Context(), tariffSelectSQL+whereSQL+fmt.Sprintf(`
		ORDER BY t.position ASC, t.created_at ASC
		LIMIT $%d OFFSET $%d
	`, len(args)+1, len(args)+2), append(args, perPage, offset)...)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"tariffs": map[string]any{"data": []any{}, "current_page": page, "last_page": 1}})
		return
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		row, err := scanTariffRow(rows)
		if err == nil {
			list = append(list, tariffToLegacyJSON(row))
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tariffs": map[string]any{
			"data":         list,
			"current_page": page,
			"last_page":    lastPage,
			"per_page":     perPage,
			"total":        total,
		},
	})
}

func (h *Handler) AdminTariffCreateForm(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"locations": h.adminTariffLocations(r, claims.TenantID),
		"games":     h.adminTariffGames(r, claims.TenantID),
	})
}

func (h *Handler) adminTariffLocations(r *http.Request, tenantID string) []map[string]any {
	rows, _ := h.readerOf(r.Context()).Query(r.Context(), `
		SELECT id::text, name FROM core.nodes WHERE tenant_id = $1 ORDER BY sort_order, name
	`, tenantID)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	list := []map[string]any{}
	if rows != nil {
		for rows.Next() {
			var id, name string
			if rows.Scan(&id, &name) == nil {
				list = append(list, map[string]any{"id": id, "name": name})
			}
		}
	}
	return list
}

func (h *Handler) adminTariffGames(r *http.Request, tenantID string) []map[string]any {
	rows, _ := h.readerOf(r.Context()).Query(r.Context(), `
		SELECT id::text, name FROM core.games WHERE tenant_id = $1 ORDER BY name
	`, tenantID)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	list := []map[string]any{}
	if rows != nil {
		for rows.Next() {
			var id, name string
			if rows.Scan(&id, &name) == nil {
				list = append(list, map[string]any{"id": id, "name": name})
			}
		}
	}
	return list
}

func (h *Handler) GetAdminTariff(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	row, err := h.loadTariffRow(r, claims.TenantID, chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tariff": tariffToLegacyJSON(row)})
}

func (h *Handler) AdminTariffEditForm(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	row, err := h.loadTariffRow(r, claims.TenantID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tariff":    tariffToLegacyJSON(row),
		"locations": h.adminTariffLocations(r, claims.TenantID),
		"games":     h.adminTariffGames(r, claims.TenantID),
	})
}

type tariffPayload struct {
	Name             string
	LocationID       string
	GameID           string
	BillingType      string
	MySQLEngine      string
	MySQLInstanceKey string
	PricePerSlot     float64
	PricePerCPUCore  float64
	PricePerRAMGb    float64
	PricePerDiskGb   float64
	BasePriceMonthly float64
	CPUMin           *int
	CPUMax           *int
	CPUStep          *int
	RAMMin           *int
	RAMMax           *int
	RAMStep          *int
	DiskMin          *int
	DiskMax          *int
	DiskStep         *int
	AllowAntiddos    bool
	AntiddosPrice    float64
	MinSlots         int
	MaxSlots         int
	CPUCores         int
	CPUShares        *int
	RAMGb            int
	DiskGb           int
	RentalPeriods    []int
	RenewalPeriods   []int
	Discounts        any
	Position         int
	IsAvailable      bool
}

func tariffBodyIntPtr(body map[string]any, key string) *int {
	v, ok := body[key]
	if !ok || v == nil || fmt.Sprint(v) == "" {
		return nil
	}
	switch n := v.(type) {
	case float64:
		i := int(n)
		return &i
	case int:
		return &n
	default:
		i, err := strconv.Atoi(fmt.Sprint(v))
		if err != nil {
			return nil
		}
		return &i
	}
}

func tariffBodyInt(body map[string]any, key string, def int) int {
	if v, ok := body[key]; ok && v != nil && fmt.Sprint(v) != "" {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		default:
			i, err := strconv.Atoi(fmt.Sprint(v))
			if err == nil {
				return i
			}
		}
	}
	return def
}

func tariffBodyFloat(body map[string]any, key string) float64 {
	if v, ok := body[key]; ok && v != nil {
		switch n := v.(type) {
		case float64:
			return n
		case int:
			return float64(n)
		default:
			f, err := strconv.ParseFloat(fmt.Sprint(v), 64)
			if err == nil {
				return f
			}
		}
	}
	return 0
}

func tariffBodyBool(body map[string]any, key string) bool {
	if v, ok := body[key]; ok {
		switch b := v.(type) {
		case bool:
			return b
		case string:
			return b == "true" || b == "1"
		case float64:
			return b != 0
		}
	}
	return false
}

func tariffBodyIntSlice(body map[string]any, key string) []int {
	v, ok := body[key]
	if !ok || v == nil {
		return []int{}
	}
	switch arr := v.(type) {
	case []any:
		out := make([]int, 0, len(arr))
		for _, item := range arr {
			switch n := item.(type) {
			case float64:
				out = append(out, int(n))
			case int:
				out = append(out, n)
			default:
				if i, err := strconv.Atoi(fmt.Sprint(item)); err == nil {
					out = append(out, i)
				}
			}
		}
		return out
	case []int:
		return arr
	default:
		return []int{}
	}
}

func parseTariffPayload(body map[string]any) (tariffPayload, string) {
	name := strings.TrimSpace(fmt.Sprint(body["name"]))
	if name == "" || name == "<nil>" {
		return tariffPayload{}, "name is required"
	}
	locationID := strings.TrimSpace(fmt.Sprint(body["location_id"]))
	if locationID == "" || locationID == "<nil>" {
		return tariffPayload{}, "location_id is required"
	}
	gameID := strings.TrimSpace(fmt.Sprint(body["game_id"]))
	if gameID == "" || gameID == "<nil>" {
		return tariffPayload{}, "game_id is required"
	}
	billingType := strings.TrimSpace(fmt.Sprint(body["billing_type"]))
	if billingType == "" || billingType == "<nil>" {
		billingType = "resources"
	}
	if billingType != "resources" && billingType != "slots" {
		return tariffPayload{}, "invalid billing_type"
	}
	rental := tariffBodyIntSlice(body, "rental_periods")
	if len(rental) == 0 {
		return tariffPayload{}, "rental_periods required"
	}
	renewal := tariffBodyIntSlice(body, "renewal_periods")
	if len(renewal) == 0 {
		return tariffPayload{}, "renewal_periods required"
	}
	cpuCores := tariffBodyInt(body, "cpu_cores", 1)
	var cpuShares *int
	if cpuCores == 0 {
		if v := tariffBodyIntPtr(body, "cpu_shares"); v != nil {
			cpuShares = v
		} else {
			return tariffPayload{}, "cpu_shares required when cpu_cores is 0"
		}
	} else if v := tariffBodyIntPtr(body, "cpu_shares"); v != nil {
		cpuShares = v
	}
	mysqlEngine := strings.TrimSpace(fmt.Sprint(body["mysql_engine"]))
	if mysqlEngine == "<nil>" {
		mysqlEngine = ""
	}
	if mysqlEngine != "" && mysqlEngine != "mysql80" && mysqlEngine != "mysql57" && mysqlEngine != "mariadb" {
		return tariffPayload{}, "invalid mysql_engine"
	}
	mysqlKey := strings.TrimSpace(fmt.Sprint(body["mysql_instance_key"]))
	if mysqlKey == "<nil>" {
		mysqlKey = ""
	}
	if mysqlKey != "" {
		re := regexp.MustCompile(`^[a-zA-Z0-9._-]+(\s*,\s*[a-zA-Z0-9._-]+)*$`)
		if !re.MatchString(mysqlKey) {
			return tariffPayload{}, "invalid mysql_instance_key"
		}
	}
	var discounts any
	if d, ok := body["discounts"]; ok && d != nil {
		switch dv := d.(type) {
		case string:
			if strings.TrimSpace(dv) != "" {
				var parsed any
				if json.Unmarshal([]byte(dv), &parsed) != nil {
					return tariffPayload{}, "invalid discounts JSON"
				}
				discounts = parsed
			}
		default:
			discounts = dv
		}
	}
	minSlots := tariffBodyInt(body, "min_slots", 1)
	maxSlots := tariffBodyInt(body, "max_slots", 100)
	if billingType == "slots" {
		if minSlots < 1 || maxSlots < 1 {
			return tariffPayload{}, "min_slots and max_slots required for slots billing"
		}
	}
	return tariffPayload{
		Name:             name,
		LocationID:       locationID,
		GameID:           gameID,
		BillingType:      billingType,
		MySQLEngine:      mysqlEngine,
		MySQLInstanceKey: mysqlKey,
		PricePerSlot:     tariffBodyFloat(body, "price_per_slot"),
		PricePerCPUCore:  tariffBodyFloat(body, "price_per_cpu_core"),
		PricePerRAMGb:    tariffBodyFloat(body, "price_per_ram_gb"),
		PricePerDiskGb:   tariffBodyFloat(body, "price_per_disk_gb"),
		BasePriceMonthly: tariffBodyFloat(body, "base_price_monthly"),
		CPUMin:           tariffBodyIntPtr(body, "cpu_min"),
		CPUMax:           tariffBodyIntPtr(body, "cpu_max"),
		CPUStep:          tariffBodyIntPtr(body, "cpu_step"),
		RAMMin:           tariffBodyIntPtr(body, "ram_min"),
		RAMMax:           tariffBodyIntPtr(body, "ram_max"),
		RAMStep:          tariffBodyIntPtr(body, "ram_step"),
		DiskMin:          tariffBodyIntPtr(body, "disk_min"),
		DiskMax:          tariffBodyIntPtr(body, "disk_max"),
		DiskStep:         tariffBodyIntPtr(body, "disk_step"),
		AllowAntiddos:    tariffBodyBool(body, "allow_antiddos"),
		AntiddosPrice:    tariffBodyFloat(body, "antiddos_price"),
		MinSlots:         minSlots,
		MaxSlots:         maxSlots,
		CPUCores:         cpuCores,
		CPUShares:        cpuShares,
		RAMGb:            tariffBodyInt(body, "ram_gb", 1),
		DiskGb:           tariffBodyInt(body, "disk_gb", 10),
		RentalPeriods:    rental,
		RenewalPeriods:   renewal,
		Discounts:        discounts,
		Position:         tariffBodyInt(body, "position", 0),
		IsAvailable:      tariffBodyBool(body, "is_available"),
	}, ""
}

func tariffMetaFromPayload(p tariffPayload) ([]byte, float64) {
	meta := tariffMetaDefaults()
	if p.MySQLEngine != "" {
		meta["mysql_engine"] = p.MySQLEngine
	}
	if p.MySQLInstanceKey != "" {
		meta["mysql_instance_key"] = p.MySQLInstanceKey
	}
	meta["price_per_slot"] = p.PricePerSlot
	meta["price_per_cpu_core"] = p.PricePerCPUCore
	meta["price_per_ram_gb"] = p.PricePerRAMGb
	meta["price_per_disk_gb"] = p.PricePerDiskGb
	meta["base_price_monthly"] = p.BasePriceMonthly
	meta["cpu_min"] = p.CPUMin
	meta["cpu_max"] = p.CPUMax
	meta["cpu_step"] = p.CPUStep
	meta["ram_min"] = p.RAMMin
	meta["ram_max"] = p.RAMMax
	meta["ram_step"] = p.RAMStep
	meta["disk_min"] = p.DiskMin
	meta["disk_max"] = p.DiskMax
	meta["disk_step"] = p.DiskStep
	meta["allow_antiddos"] = p.AllowAntiddos
	meta["antiddos_price"] = p.AntiddosPrice
	metaJSON, _ := json.Marshal(meta)
	priceMonthly := p.BasePriceMonthly
	if p.BillingType == "slots" {
		priceMonthly = p.PricePerSlot
	}
	return metaJSON, priceMonthly
}

func slugFromTariffName(name string) string {
	s := strings.ToLower(name)
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s = re.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "tariff"
	}
	return s + "-" + fmt.Sprintf("%x", time.Now().UnixNano())[:8]
}

func (h *Handler) CreateTariff(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	var body map[string]any
	if json.NewDecoder(r.Body).Decode(&body) != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	payload, errMsg := parseTariffPayload(body)
	if errMsg != "" {
		writeError(w, http.StatusBadRequest, errMsg)
		return
	}
	metaJSON, priceMonthly := tariffMetaFromPayload(payload)
	rentalJSON, _ := json.Marshal(payload.RentalPeriods)
	renewalJSON, _ := json.Marshal(payload.RenewalPeriods)
	discountsJSON := []byte("{}")
	if payload.Discounts != nil {
		discountsJSON, _ = json.Marshal(payload.Discounts)
	}
	slotsMin := payload.MinSlots
	slotsMax := payload.MaxSlots
	if payload.BillingType != "slots" {
		slotsMin = 1
		slotsMax = 100
	}
	ramMb := payload.RAMGb * 1024
	diskMb := payload.DiskGb * 1024
	cpuCores := float64(payload.CPUCores)
	slug := slugFromTariffName(payload.Name)
	_, err := h.dbOf(r.Context()).Exec(r.Context(), `
		INSERT INTO core.tariffs (
			tenant_id, node_id, game_id, name, slug, billing_type, price_monthly,
			slots_min, slots_max, cpu_cores, cpu_shares, ram_mb, disk_mb,
			rental_periods, renewal_periods, discounts, position, active, meta
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12, $13,
			$14::jsonb, $15::jsonb, $16::jsonb, $17, $18, $19::jsonb
		)
	`, claims.TenantID, payload.LocationID, payload.GameID, payload.Name, slug, payload.BillingType, priceMonthly,
		slotsMin, slotsMax, cpuCores, payload.CPUShares, ramMb, diskMb,
		rentalJSON, renewalJSON, discountsJSON, payload.Position, payload.IsAvailable, metaJSON)
	if err != nil {
		writeError(w, http.StatusBadRequest, "create failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) updateTariffFromBody(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	var body map[string]any
	if json.NewDecoder(r.Body).Decode(&body) != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	payload, errMsg := parseTariffPayload(body)
	if errMsg != "" {
		writeError(w, http.StatusBadRequest, errMsg)
		return
	}
	metaJSON, priceMonthly := tariffMetaFromPayload(payload)
	rentalJSON, _ := json.Marshal(payload.RentalPeriods)
	renewalJSON, _ := json.Marshal(payload.RenewalPeriods)
	discountsJSON := []byte("{}")
	if payload.Discounts != nil {
		discountsJSON, _ = json.Marshal(payload.Discounts)
	}
	slotsMin := payload.MinSlots
	slotsMax := payload.MaxSlots
	if payload.BillingType != "slots" {
		slotsMin = 1
		slotsMax = 100
	}
	ramMb := payload.RAMGb * 1024
	diskMb := payload.DiskGb * 1024
	cpuCores := float64(payload.CPUCores)
	tag, err := h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.tariffs SET
			node_id = $3, game_id = $4, name = $5, billing_type = $6, price_monthly = $7,
			slots_min = $8, slots_max = $9, cpu_cores = $10, cpu_shares = $11,
			ram_mb = $12, disk_mb = $13, rental_periods = $14::jsonb, renewal_periods = $15::jsonb,
			discounts = $16::jsonb, position = $17, active = $18, meta = $19::jsonb
		WHERE id = $1 AND tenant_id = $2
	`, id, claims.TenantID, payload.LocationID, payload.GameID, payload.Name, payload.BillingType, priceMonthly,
		slotsMin, slotsMax, cpuCores, payload.CPUShares, ramMb, diskMb,
		rentalJSON, renewalJSON, discountsJSON, payload.Position, payload.IsAvailable, metaJSON)
	if err != nil || tag.RowsAffected() == 0 {
		writeError(w, http.StatusBadRequest, "update failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) UpdateTariff(w http.ResponseWriter, r *http.Request) {
	h.updateTariffFromBody(w, r)
}

func (h *Handler) PutAdminTariff(w http.ResponseWriter, r *http.Request) {
	h.updateTariffFromBody(w, r)
}

func (h *Handler) DeleteTariff(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	ctx := r.Context()
	tariffID := chi.URLParam(r, "id")

	// Удаление обнуляло ссылку у серверов (FK со SET NULL), и продление такого
	// сервера считалось по пустому тарифу — то есть бесплатно. Тихая потеря
	// выручки, которую никто не замечает: клиент платит ноль и остаётся.
	var live int
	if err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT count(*) FROM core.servers WHERE tariff_id = $1::uuid AND tenant_id = $2
	`, tariffID, claims.TenantID).Scan(&live); err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось проверить серверы тарифа")
		return
	}
	if live > 0 {
		writeError(w, http.StatusConflict,
			fmt.Sprintf("на тарифе %d сервер(ов): перенесите их на другой тариф или отключите тариф вместо удаления", live))
		return
	}

	// Ошибка удаления глушилась, а ответ всё равно был успешным.
	if _, err := h.dbOf(ctx).Exec(ctx, `
		DELETE FROM core.tariffs WHERE id = $1 AND tenant_id = $2
	`, tariffID, claims.TenantID); err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось удалить тариф")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
