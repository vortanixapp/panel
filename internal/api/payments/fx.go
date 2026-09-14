package payments

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Rates struct {
	Base      string
	Values    map[string]float64
	FetchedAt time.Time
}

func (r *Rates) Rate(from, to string) (float64, bool) {
	from = strings.ToUpper(strings.TrimSpace(from))
	to = strings.ToUpper(strings.TrimSpace(to))
	if from == to {
		return 1, true
	}
	if r == nil {
		return 0, false
	}
	f, okFrom := r.Values[from]
	t, okTo := r.Values[to]
	if !okFrom || !okTo || f <= 0 || t <= 0 {
		return 0, false
	}
	return t / f, true
}

func (r *Rates) Subset(codes []string) map[string]float64 {
	out := map[string]float64{}
	if r == nil {
		return out
	}
	for _, c := range codes {
		c = strings.ToUpper(strings.TrimSpace(c))
		if v, ok := r.Values[c]; ok {
			out[c] = v
		}
	}
	return out
}

const (
	ratesFreshFor  = 6 * time.Hour
	ratesUsableFor = 72 * time.Hour
)

type rateSource struct {
	mu       sync.Mutex
	endpoint string
	cached   *Rates
}

var exchangeRates = &rateSource{endpoint: "https://open.er-api.com/v6/latest/USD"}

func CurrentRates(ctx context.Context) (*Rates, error) {
	return exchangeRates.get(ctx)
}

func (s *rateSource) get(ctx context.Context) (*Rates, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cached != nil && time.Since(s.cached.FetchedAt) < ratesFreshFor {
		return s.cached, nil
	}
	fresh, err := s.fetch(ctx)
	if err == nil {
		s.cached = fresh
		return fresh, nil
	}
	if s.cached != nil && time.Since(s.cached.FetchedAt) < ratesUsableFor {
		return s.cached, nil
	}
	return nil, fmt.Errorf("курсы валют недоступны: %w", err)
}

func (s *rateSource) fetch(ctx context.Context) (*Rates, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	var wire struct {
		Result string             `json:"result"`
		Base   string             `json:"base_code"`
		Rates  map[string]float64 `json:"rates"`
	}
	if err := sendJSON(ctx, http.MethodGet, s.endpoint, nil, &wire); err != nil {
		return nil, err
	}
	if wire.Result != "success" || len(wire.Rates) == 0 {
		return nil, fmt.Errorf("источник курсов ответил %q", wire.Result)
	}
	return &Rates{Base: wire.Base, Values: wire.Rates, FetchedAt: time.Now()}, nil
}

type Quote struct {
	Amount         float64
	Currency       string
	ChargeAmount   float64
	ChargeCurrency string
	FeePercent     float64
	FXFeePercent   float64
	Rate           float64
}

func ClampPercent(p float64) float64 {
	if math.IsNaN(p) || p < 0 {
		return 0
	}
	if p > 100 {
		return 100
	}
	return p
}

func NewQuote(amount float64, currency, chargeCurrency string, feePercent, fxFeePercent float64, rates *Rates) (Quote, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	chargeCurrency = strings.ToUpper(strings.TrimSpace(chargeCurrency))
	if chargeCurrency == "" {
		chargeCurrency = currency
	}
	q := Quote{
		Amount:         amount,
		Currency:       currency,
		ChargeCurrency: chargeCurrency,
		FeePercent:     ClampPercent(feePercent),
		Rate:           1,
	}
	value := amount
	if currency != chargeCurrency {
		rate, ok := rates.Rate(currency, chargeCurrency)
		if !ok {
			return q, fmt.Errorf("нет курса %s → %s", currency, chargeCurrency)
		}
		q.Rate = rate
		q.FXFeePercent = ClampPercent(fxFeePercent)
		value = value * rate * (1 + q.FXFeePercent/100)
	}
	value = value * (1 + q.FeePercent/100)
	q.ChargeAmount = math.Ceil(value*100-1e-6) / 100
	if q.ChargeAmount <= 0 {
		return q, fmt.Errorf("сумма к оплате получилась нулевой")
	}
	return q, nil
}
