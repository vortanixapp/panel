package payments

import (
	"bytes"
	"context"
	"encoding/json"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
)

type CheckoutInput struct {
	PaymentID   string
	InvoiceNo   int64
	Amount      float64
	Currency    string
	Description string
	ReturnURL   string
	FailURL     string
	NotifyURL   string
	MethodID    string
	Email       string
	UserID      string
}

type FormField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Form struct {
	Action string      `json:"action"`
	Method string      `json:"method"`
	Fields []FormField `json:"fields"`
}

type Checkout struct {
	RedirectURL       string
	Form              *Form
	ProviderPaymentID string
	Manual            bool
}

type Provider interface {
	Code() string
	CreateCheckout(ctx context.Context, cfg map[string]any, in CheckoutInput) (Checkout, error)
}

type NotifyStatus int

const (
	NotifyIgnore NotifyStatus = iota
	NotifyCheck
	NotifyPaid
	NotifyFailed
)

type Notification struct {
	Status            NotifyStatus
	PaymentID         string
	InvoiceNo         int64
	ProviderPaymentID string
	Amount            float64
	Currency          string
}

type NotifyRequest struct {
	Method    string
	NotifyURL string
	Header    http.Header
	Query     url.Values
	Form      url.Values
	Body      []byte
}

type Notifier interface {
	HandleNotification(ctx context.Context, cfg map[string]any, req *NotifyRequest) (Notification, error)
}

type Response struct {
	Status      int
	ContentType string
	Body        string
}

type Acknowledger interface {
	Accept(n Notification) Response
	Reject(n Notification, err error) Response
}

func NewNotifyRequest(r *http.Request, body []byte, notifyURL string) *NotifyRequest {
	req := &NotifyRequest{
		Method:    r.Method,
		NotifyURL: notifyURL,
		Header:    r.Header,
		Query:     r.URL.Query(),
		Form:      url.Values{},
		Body:      body,
	}
	mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return req
	}
	switch mediaType {
	case "application/x-www-form-urlencoded":
		if values, err := url.ParseQuery(string(body)); err == nil {
			req.Form = values
		}
	case "multipart/form-data":
		reader := multipart.NewReader(bytes.NewReader(body), params["boundary"])
		if form, err := reader.ReadForm(1 << 20); err == nil {
			req.Form = url.Values(form.Value)
		}
	}
	return req
}

func (r *NotifyRequest) Value(keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(r.Form.Get(k)); v != "" {
			return v
		}
		if v := strings.TrimSpace(r.Query.Get(k)); v != "" {
			return v
		}
	}
	return ""
}

func (r *NotifyRequest) IsJSON() bool {
	trimmed := bytes.TrimSpace(r.Body)
	return len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[')
}

func (r *NotifyRequest) DecodeJSON(dest any) error {
	dec := json.NewDecoder(bytes.NewReader(r.Body))
	dec.UseNumber()
	return dec.Decode(dest)
}

func textResponse(status int, body string) Response {
	return Response{Status: status, ContentType: "text/plain; charset=utf-8", Body: body}
}

func jsonResponse(status int, v any) Response {
	raw, _ := json.Marshal(v)
	return Response{Status: status, ContentType: "application/json", Body: string(raw)}
}

func AcceptResponse(p Provider, n Notification) Response {
	if a, ok := p.(Acknowledger); ok {
		return a.Accept(n)
	}
	return textResponse(http.StatusOK, "OK")
}

func RejectResponse(p Provider, n Notification, err error) Response {
	if a, ok := p.(Acknowledger); ok {
		return a.Reject(n, err)
	}
	msg := "rejected"
	if err != nil {
		msg = err.Error()
	}
	return textResponse(http.StatusBadRequest, msg)
}
