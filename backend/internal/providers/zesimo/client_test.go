package zesimo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientWalletAndPackages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("missing bearer token")
		}
		switch r.URL.Path {
		case "/api/v1/wallet":
			_, _ = w.Write([]byte(`{"balance":42.5,"currency":"USD"}`))
		case "/api/v1/packages":
			_, _ = w.Write([]byte(`{"data":[{"id":1,"package_code":"KZ-1","name":"Kazakhstan","data_gb":3,"duration_days":7,"volume":3221225472,"reseller_price":4.2,"countries":["KZ"]}],"meta":{"current_page":1,"last_page":1,"per_page":100,"total":1}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client := New(server.URL, "secret", time.Second)
	wallet, err := client.Wallet(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if wallet.Balance != 42.5 || wallet.Currency != "USD" {
		t.Fatalf("unexpected wallet: %#v", wallet)
	}
	packages, meta, err := client.PackagesPage(context.Background(), 1, 100)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Total != 1 || len(packages) != 1 || packages[0].PackageCode != "KZ-1" {
		t.Fatalf("unexpected packages: %#v %#v", packages, meta)
	}
}

func TestCreateOrderUsesIdempotencyAndParsesESIM(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/orders" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Idempotency-Key") != "order-1" {
			t.Error("missing idempotency key")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["package_code"] != "KZ-1" || body["quantity"].(float64) != 1 {
			t.Fatalf("unexpected body: %#v", body)
		}
		_, _ = w.Write([]byte(`{"data":{"id":99,"status":"completed","esims":[{"id":7,"iccid":"8901","imsi":"40101","activation_code":"LPA:1$smdp.test$MATCH","install_url":"https://example.test/i/7"}]}}`))
	}))
	defer server.Close()
	client := New(server.URL, "secret", time.Second)
	order, err := client.CreateOrder(context.Background(), "KZ-1", 1, "order-1")
	if err != nil {
		t.Fatal(err)
	}
	if order.ID != "99" || len(order.ESIMs) != 1 || order.ESIMs[0].ICCID != "8901" || order.ESIMs[0].SMDPAddress != "" {
		t.Fatalf("unexpected order: %#v", order)
	}
}
