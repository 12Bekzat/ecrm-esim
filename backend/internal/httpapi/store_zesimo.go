package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"ecrm/backend/internal/providers/zesimo"
	"github.com/jackc/pgx/v5"
)

const zesimoSupplierName = "Zesimo"

type ZesimoSyncResult struct {
	SupplierID string    `json:"supplierId"`
	Packages   int       `json:"packages"`
	Balance    float64   `json:"balance"`
	Currency   string    `json:"currency"`
	Pages      int       `json:"pages"`
	SyncedAt   time.Time `json:"syncedAt"`
}

func (s *Store) SyncZesimoCatalog(ctx context.Context, client *zesimo.Client) (ZesimoSyncResult, error) {
	if client == nil || !client.Configured() {
		return ZesimoSyncResult{}, errors.New("Zesimo API key is not configured")
	}
	wallet, err := client.Wallet(ctx)
	if err != nil {
		return ZesimoSyncResult{}, err
	}

	packages := make([]zesimo.Package, 0, 7000)
	lastPage := 1
	rateLimit := time.NewTicker(1100 * time.Millisecond)
	defer rateLimit.Stop()
	for page := 1; page <= lastPage; page++ {
		if page > 1 {
			select {
			case <-ctx.Done():
				return ZesimoSyncResult{}, ctx.Err()
			case <-rateLimit.C:
			}
		}
		items, meta, err := client.PackagesPage(ctx, page, 100)
		if err != nil {
			return ZesimoSyncResult{}, fmt.Errorf("load Zesimo packages page %d: %w", page, err)
		}
		packages = append(packages, items...)
		if meta.LastPage > 0 {
			lastPage = meta.LastPage
		}
		if lastPage > 1000 {
			return ZesimoSyncResult{}, errors.New("Zesimo package pagination exceeded safety limit")
		}
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return ZesimoSyncResult{}, err
	}
	defer tx.Rollback(ctx)
	var supplierID string
	err = tx.QueryRow(ctx, `INSERT INTO suppliers(name,contact_name,api_base_url,api_status,currency,balance,terms,status,low_balance_threshold,last_synced_at)
		VALUES($1,'my_key','https://zesimo.com','online',$2,$3,'Bearer API · on-demand provisioning','active',25,now())
		ON CONFLICT(name) DO UPDATE SET api_status='online',currency=EXCLUDED.currency,balance=EXCLUDED.balance,last_synced_at=now(),status=CASE WHEN EXCLUDED.balance<=suppliers.low_balance_threshold THEN 'low_balance' ELSE 'active' END
		RETURNING id`, zesimoSupplierName, wallet.Currency, wallet.Balance).Scan(&supplierID)
	if err != nil {
		return ZesimoSyncResult{}, err
	}
	if _, err = tx.Exec(ctx, `CREATE TEMP TABLE zesimo_package_sync(
		external_id text PRIMARY KEY,name text,country text,region text,data_mb bigint,validity_days int,cost numeric(16,4),suggested_price numeric(16,4),metadata jsonb
	) ON COMMIT DROP`); err != nil {
		return ZesimoSyncResult{}, err
	}

	rows := make([][]any, 0, len(packages))
	for _, item := range packages {
		dataMB := item.Volume / (1024 * 1024)
		if dataMB <= 0 && item.DataGB != nil {
			dataMB = int64(math.Round(*item.DataGB * 1024))
		}
		if dataMB <= 0 {
			dataMB = 1
		}
		days := item.DurationDays
		if days <= 0 && item.MaxDurationDays != nil {
			days = *item.MaxDurationDays
		}
		if days <= 0 {
			days = 1
		}
		country := strings.Join(item.Countries, ",")
		region := item.PlanType
		metadata, _ := json.Marshal(item)
		suggested := math.Ceil(item.ResellerPrice*1.30*100) / 100
		rows = append(rows, []any{item.PackageCode, item.Name, country, region, dataMB, days, item.ResellerPrice, suggested, metadata})
	}
	_, err = tx.CopyFrom(ctx, pgx.Identifier{"zesimo_package_sync"}, []string{"external_id", "name", "country", "region", "data_mb", "validity_days", "cost", "suggested_price", "metadata"}, pgx.CopyFromRows(rows))
	if err != nil {
		return ZesimoSyncResult{}, fmt.Errorf("stage Zesimo packages: %w", err)
	}
	if _, err = tx.Exec(ctx, `UPDATE tariffs SET status='inactive' WHERE supplier_id=$1 AND deleted_at IS NULL`, supplierID); err != nil {
		return ZesimoSyncResult{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO tariffs(supplier_id,external_id,name,country,region,data_mb,validity_days,cost,price,min_price,recommended_price,currency,status)
		SELECT $1,external_id,name,country,region,data_mb,validity_days,cost,suggested_price,cost,suggested_price,$2,'active' FROM zesimo_package_sync
		ON CONFLICT(supplier_id,external_id) DO UPDATE SET name=EXCLUDED.name,country=EXCLUDED.country,region=EXCLUDED.region,data_mb=EXCLUDED.data_mb,validity_days=EXCLUDED.validity_days,cost=EXCLUDED.cost,min_price=EXCLUDED.min_price,recommended_price=EXCLUDED.recommended_price,currency=EXCLUDED.currency,status='active',deleted_at=NULL`, supplierID, wallet.Currency); err != nil {
		return ZesimoSyncResult{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO supplier_sync_logs(supplier_id,status,tariffs_received,finished_at) VALUES($1,'success',$2,now())`, supplierID, len(packages)); err != nil {
		return ZesimoSyncResult{}, err
	}
	if wallet.Balance <= 25 {
		_, _ = tx.Exec(ctx, `INSERT INTO notifications(type,message,level,entity_type,entity_id)
			SELECT 'Баланс',$1,'danger','supplier',$2 WHERE NOT EXISTS(SELECT 1 FROM notifications WHERE entity_id=$2 AND type='Баланс' AND read_at IS NULL)`, fmt.Sprintf("Zesimo balance is low: %.2f %s", wallet.Balance, wallet.Currency), supplierID)
	}
	if err = tx.Commit(ctx); err != nil {
		return ZesimoSyncResult{}, err
	}
	return ZesimoSyncResult{SupplierID: supplierID, Packages: len(packages), Balance: wallet.Balance, Currency: wallet.Currency, Pages: lastPage, SyncedAt: time.Now().UTC()}, nil
}

func (s *Store) IsZesimoSupplier(ctx context.Context, supplierID string) (bool, error) {
	var name string
	err := s.db.QueryRow(ctx, `SELECT name FROM suppliers WHERE id=$1 AND deleted_at IS NULL`, supplierID).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, ErrNotFound
	}
	return name == zesimoSupplierName, err
}

func (s *Store) PrepareZesimoOrder(ctx context.Context, orderID string) (packageCode, idempotencyKey string, err error) {
	err = s.db.QueryRow(ctx, `SELECT t.external_id,COALESCE(o.provider_idempotency_key,'') FROM orders o JOIN tariffs t ON t.id=o.tariff_id JOIN suppliers s ON s.id=o.supplier_id WHERE o.id=$1 AND s.name=$2`, orderID, zesimoSupplierName).Scan(&packageCode, &idempotencyKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrNotFound
	}
	if err != nil {
		return "", "", err
	}
	if idempotencyKey == "" {
		idempotencyKey = "ecrm_order_" + strings.ReplaceAll(orderID, "-", "")
		_, err = s.db.Exec(ctx, `UPDATE orders SET provider_idempotency_key=$2,provisioning_status='pending',provisioning_error='' WHERE id=$1`, orderID, idempotencyKey)
	}
	return packageCode, idempotencyKey, err
}

func (s *Store) CompleteZesimoProvisioning(ctx context.Context, orderID string, external zesimo.Order) (Order, error) {
	if len(external.ESIMs) == 0 {
		return Order{}, errors.New("Zesimo order has no provisioned eSIM yet")
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Order{}, err
	}
	defer tx.Rollback(ctx)
	var supplierID, tariffID string
	var totalMB int64
	if err = tx.QueryRow(ctx, `SELECT supplier_id,tariff_id,(SELECT data_mb FROM tariffs WHERE id=orders.tariff_id) FROM orders WHERE id=$1 FOR UPDATE`, orderID).Scan(&supplierID, &tariffID, &totalMB); err != nil {
		return Order{}, err
	}
	item := external.ESIMs[0]
	activation := item.ActivationCode
	smdp := item.SMDPAddress
	if smdp == "" && strings.HasPrefix(activation, "LPA:") {
		parts := strings.Split(activation, "$")
		if len(parts) > 1 {
			smdp = parts[1]
		}
	}
	payload := activation
	providerPayload := item.Raw
	if len(providerPayload) == 0 {
		providerPayload = external.Raw
	}
	var esimID string
	err = tx.QueryRow(ctx, `INSERT INTO esims(supplier_id,tariff_id,external_id,iccid,imsi,activation_code,smdp_address,qr_payload,install_url,ios_tap_link,android_tap_link,status,traffic_total_mb,traffic_remaining_mb,provider_payload)
		VALUES($1,$2,NULLIF($3,''),$4,$5,$6,$7,$8,$9,$10,$11,'sold',$12,$12,$13)
		ON CONFLICT(iccid) DO UPDATE SET external_id=EXCLUDED.external_id,activation_code=EXCLUDED.activation_code,smdp_address=EXCLUDED.smdp_address,qr_payload=EXCLUDED.qr_payload,install_url=EXCLUDED.install_url,ios_tap_link=EXCLUDED.ios_tap_link,android_tap_link=EXCLUDED.android_tap_link,provider_payload=EXCLUDED.provider_payload
		RETURNING id`, supplierID, tariffID, item.ID, item.ICCID, item.IMSI, activation, smdp, payload, item.InstallURL, item.IOSTapLink, item.AndroidTapLink, totalMB, providerPayload).Scan(&esimID)
	if err != nil {
		return Order{}, mapDBError(err)
	}
	providerID := external.ID
	if providerID == "" {
		providerID = "unknown-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	_, err = tx.Exec(ctx, `UPDATE orders SET esim_id=$2,provider_order_id=$3,provider_payload=$4,provisioning_status='provisioned',provisioning_error='',status=CASE WHEN payment_status='paid' THEN 'esim_assigned' ELSE status END WHERE id=$1`, orderID, esimID, providerID, external.Raw)
	if err != nil {
		return Order{}, mapDBError(err)
	}
	if err = tx.Commit(ctx); err != nil {
		return Order{}, err
	}
	return s.OrderByID(ctx, orderID)
}

func (s *Store) RecordZesimoProvisioning(ctx context.Context, orderID string, external zesimo.Order) error {
	var providerID any
	if external.ID != "" {
		providerID = external.ID
	}
	_, err := s.db.Exec(ctx, `UPDATE orders SET provider_order_id=COALESCE($2,provider_order_id),provider_payload=$3,provisioning_status='pending',provisioning_error='' WHERE id=$1`, orderID, providerID, external.Raw)
	return mapDBError(err)
}

func (s *Store) FailZesimoProvisioning(ctx context.Context, orderID string, provisionErr error) {
	message := provisionErr.Error()
	if len(message) > 1000 {
		message = message[:1000]
	}
	_, _ = s.db.Exec(ctx, `UPDATE orders SET provisioning_status='failed',provisioning_error=$2 WHERE id=$1`, orderID, message)
}
