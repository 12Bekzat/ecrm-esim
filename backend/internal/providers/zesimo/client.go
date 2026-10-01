package zesimo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func New(baseURL, apiKey string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  strings.TrimSpace(apiKey),
		http:    &http.Client{Timeout: timeout},
	}
}

func (c *Client) Configured() bool { return c != nil && c.apiKey != "" }

type Wallet struct {
	Balance  float64 `json:"balance"`
	Currency string  `json:"currency"`
}

type Plan struct {
	Type             string   `json:"type"`
	DailyHighSpeedGB *float64 `json:"daily_high_speed_gb"`
	TotalGB          *float64 `json:"total_gb"`
	FUPThrottle      *string  `json:"fup_throttle"`
	ResetsDaily      bool     `json:"resets_daily"`
	DurationDays     int      `json:"duration_days"`
	HumanSummary     string   `json:"human_summary"`
}

type Package struct {
	ID                        int64    `json:"id"`
	PackageCode               string   `json:"package_code"`
	Name                      string   `json:"name"`
	PlanType                  string   `json:"plan_type"`
	DataType                  string   `json:"data_type"`
	DataGB                    *float64 `json:"data_gb"`
	DurationDays              int      `json:"duration_days"`
	IsUnlimited               bool     `json:"is_unlimited"`
	FlexibleDuration          bool     `json:"flexible_duration"`
	MaxDurationDays           *int     `json:"max_duration_days"`
	QuotedDays                *int     `json:"quoted_days"`
	VoiceMinutes              *int     `json:"voice_minutes"`
	SMSCount                  *int     `json:"sms_count"`
	APN                       *string  `json:"apn"`
	FUPPolicy                 *string  `json:"fup_policy"`
	Plan                      Plan     `json:"plan"`
	IPExport                  string   `json:"ip_export"`
	Operators                 string   `json:"operators"`
	Volume                    int64    `json:"volume"`
	ResellerPrice             float64  `json:"reseller_price"`
	PriceUnits                int64    `json:"price_units"`
	ProvisioningFee           *float64 `json:"provisioning_fee"`
	ProvisioningFeeRefundable *bool    `json:"provisioning_fee_refundable"`
	RefundPolicy              string   `json:"refund_policy"`
	Countries                 []string `json:"countries"`
}

type Meta struct {
	CurrentPage int `json:"current_page"`
	LastPage    int `json:"last_page"`
	PerPage     int `json:"per_page"`
	Total       int `json:"total"`
}

type packagePage struct {
	Data []Package `json:"data"`
	Meta Meta      `json:"meta"`
}

type ESIM struct {
	ID             string          `json:"id"`
	ICCID          string          `json:"iccid"`
	IMSI           string          `json:"imsi"`
	Status         string          `json:"status"`
	ActivationCode string          `json:"activation_code"`
	InstallURL     string          `json:"install_url"`
	IOSTapLink     string          `json:"ios_tap_link"`
	AndroidTapLink string          `json:"android_tap_link"`
	SMDPAddress    string          `json:"smdp_address"`
	MatchingID     string          `json:"matching_id"`
	ExpiresAt      string          `json:"expires_at"`
	Raw            json.RawMessage `json:"-"`
}

type Order struct {
	ID       string          `json:"id"`
	Status   string          `json:"status"`
	Currency string          `json:"currency"`
	Total    float64         `json:"total"`
	ESIMs    []ESIM          `json:"esims"`
	Raw      json.RawMessage `json:"-"`
}

type APIError struct {
	StatusCode int
	Message    string
	Errors     map[string][]string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("Zesimo API (%d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("Zesimo API returned HTTP %d", e.StatusCode)
}

func (c *Client) Wallet(ctx context.Context) (Wallet, error) {
	var out Wallet
	err := c.do(ctx, http.MethodGet, "/api/v1/wallet", nil, "", &out)
	return out, err
}

func (c *Client) PackagesPage(ctx context.Context, page, perPage int) ([]Package, Meta, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 100
	}
	params := url.Values{"page": {strconv.Itoa(page)}, "per_page": {strconv.Itoa(perPage)}}
	var out packagePage
	if err := c.do(ctx, http.MethodGet, "/api/v1/packages?"+params.Encode(), nil, "", &out); err != nil {
		return nil, Meta{}, err
	}
	return out.Data, out.Meta, nil
}

func (c *Client) Package(ctx context.Context, id int64) (Package, error) {
	var out struct {
		Data Package `json:"data"`
	}
	if err := c.do(ctx, http.MethodGet, "/api/v1/packages/"+strconv.FormatInt(id, 10), nil, "", &out); err != nil {
		return Package{}, err
	}
	if out.Data.ID == 0 {
		var direct Package
		if err := c.do(ctx, http.MethodGet, "/api/v1/packages/"+strconv.FormatInt(id, 10), nil, "", &direct); err != nil {
			return Package{}, err
		}
		return direct, nil
	}
	return out.Data, nil
}

func (c *Client) CreateOrder(ctx context.Context, packageCode string, quantity int, idempotencyKey string) (Order, error) {
	if packageCode == "" {
		return Order{}, errors.New("Zesimo package code is required")
	}
	if quantity < 1 {
		return Order{}, errors.New("Zesimo order quantity must be positive")
	}
	if idempotencyKey == "" {
		return Order{}, errors.New("idempotency key is required")
	}
	body := map[string]any{"package_code": packageCode, "quantity": quantity}
	var raw json.RawMessage
	if err := c.do(ctx, http.MethodPost, "/api/v1/orders", body, idempotencyKey, &raw); err != nil {
		return Order{}, err
	}
	return parseOrder(raw)
}

func (c *Client) Order(ctx context.Context, id string) (Order, error) {
	if id == "" {
		return Order{}, errors.New("Zesimo order ID is required")
	}
	var raw json.RawMessage
	if err := c.do(ctx, http.MethodGet, "/api/v1/orders/"+url.PathEscape(id), nil, "", &raw); err != nil {
		return Order{}, err
	}
	return parseOrder(raw)
}

func (c *Client) do(ctx context.Context, method, path string, body any, idempotencyKey string, out any) error {
	if !c.Configured() {
		return errors.New("Zesimo API key is not configured")
	}
	var payload []byte
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		payload = encoded
	}
	var responseBody []byte
	var statusCode int
	for attempt := 0; attempt < 13; attempt++ {
		var reader io.Reader
		if payload != nil {
			reader = bytes.NewReader(payload)
		}
		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
		if err != nil {
			return err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if idempotencyKey != "" {
			req.Header.Set("Idempotency-Key", idempotencyKey)
		}
		resp, err := c.http.Do(req)
		if err != nil {
			return fmt.Errorf("Zesimo request: %w", err)
		}
		responseBody, err = io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("read Zesimo response: %w", err)
		}
		statusCode = resp.StatusCode
		if statusCode != http.StatusTooManyRequests || attempt == 12 {
			break
		}
		timer := time.NewTimer(5 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	if statusCode < 200 || statusCode >= 300 {
		apiErr := &APIError{StatusCode: statusCode}
		_ = json.Unmarshal(responseBody, apiErr)
		return apiErr
	}
	if out == nil || len(responseBody) == 0 {
		return nil
	}
	if raw, ok := out.(*json.RawMessage); ok {
		*raw = append((*raw)[:0], responseBody...)
		return nil
	}
	if err := json.Unmarshal(responseBody, out); err != nil {
		return fmt.Errorf("decode Zesimo response: %w", err)
	}
	return nil
}

func parseOrder(raw json.RawMessage) (Order, error) {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return Order{}, fmt.Errorf("decode Zesimo order: %w", err)
	}
	if root, ok := value.(map[string]any); ok {
		if data, exists := root["data"]; exists {
			value = data
		}
	}
	object, _ := value.(map[string]any)
	order := Order{ID: stringValue(object, "id", "order_id", "order_number"), Status: stringValue(object, "status"), Currency: stringValue(object, "currency"), Total: numberValue(object, "total", "amount", "total_amount"), Raw: append(json.RawMessage(nil), raw...)}
	order.ESIMs = collectESIMs(value)
	return order, nil
}

func collectESIMs(value any) []ESIM {
	items := make([]ESIM, 0)
	seen := map[string]bool{}
	var walk func(any)
	walk = func(current any) {
		switch x := current.(type) {
		case map[string]any:
			if iccid := stringValue(x, "iccid"); iccid != "" && !seen[iccid] {
				encoded, _ := json.Marshal(x)
				items = append(items, ESIM{ID: stringValue(x, "id", "esim_id"), ICCID: iccid, IMSI: stringValue(x, "imsi"), Status: stringValue(x, "status"), ActivationCode: stringValue(x, "activation_code", "lpa"), InstallURL: stringValue(x, "install_url"), IOSTapLink: stringValue(x, "ios_tap_link"), AndroidTapLink: stringValue(x, "android_tap_link"), SMDPAddress: stringValue(x, "smdp_address", "smdp_plus"), MatchingID: stringValue(x, "matching_id"), ExpiresAt: stringValue(x, "expires_at"), Raw: encoded})
				seen[iccid] = true
			}
			for _, child := range x {
				walk(child)
			}
		case []any:
			for _, child := range x {
				walk(child)
			}
		}
	}
	walk(value)
	return items
}

func stringValue(object map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := object[key]
		if !ok || value == nil {
			continue
		}
		switch x := value.(type) {
		case string:
			return x
		case float64:
			return strconv.FormatInt(int64(x), 10)
		case json.Number:
			return x.String()
		}
	}
	return ""
}

func numberValue(object map[string]any, keys ...string) float64 {
	for _, key := range keys {
		switch x := object[key].(type) {
		case float64:
			return x
		case string:
			value, _ := strconv.ParseFloat(x, 64)
			return value
		case json.Number:
			value, _ := x.Float64()
			return value
		}
	}
	return 0
}
