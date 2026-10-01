package httpapi

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type ESIMInput struct {
	SupplierID         string     `json:"supplierId"`
	TariffID           *string    `json:"tariffId"`
	ICCID              string     `json:"iccid"`
	IMSI               string     `json:"imsi"`
	ActivationCode     string     `json:"activationCode"`
	SMDPAddress        string     `json:"smdpAddress"`
	QRPayload          string     `json:"qrPayload"`
	Status             string     `json:"status"`
	TrafficTotalMB     int64      `json:"trafficTotalMb"`
	TrafficRemainingMB int64      `json:"trafficRemainingMb"`
	ActivatedAt        *time.Time `json:"activatedAt"`
	ExpiresAt          *time.Time `json:"expiresAt"`
}

const esimSelect = `SELECT e.id,'ES-'||lpad(e.esim_code::text,4,'0'),e.supplier_id,s.name,e.tariff_id,COALESCE(t.name,''),COALESCE(t.country,''),
	e.iccid,e.imsi,e.activation_code,e.smdp_address,e.qr_payload,e.install_url,e.ios_tap_link,e.android_tap_link,e.status,e.traffic_total_mb,e.traffic_remaining_mb,
	COALESCE(c.name,''),o.id,o.order_number,e.activated_at,e.expires_at,e.created_at,e.updated_at
	FROM esims e JOIN suppliers s ON s.id=e.supplier_id LEFT JOIN tariffs t ON t.id=e.tariff_id
	LEFT JOIN orders o ON o.esim_id=e.id LEFT JOIN customers c ON c.id=o.customer_id`

func scanESIM(row pgx.Row) (ESIM, error) {
	var x ESIM
	err := row.Scan(&x.ID, &x.Code, &x.SupplierID, &x.SupplierName, &x.TariffID, &x.TariffName, &x.Country, &x.ICCID, &x.IMSI, &x.ActivationCode, &x.SMDPAddress, &x.QRPayload, &x.InstallURL, &x.IOSTapLink, &x.AndroidTapLink, &x.Status, &x.TrafficTotalMB, &x.TrafficRemainingMB, &x.CustomerName, &x.OrderID, &x.OrderNumber, &x.ActivatedAt, &x.ExpiresAt, &x.CreatedAt, &x.UpdatedAt)
	return x, err
}

func (s *Store) ListESIMs(ctx context.Context, query, status, supplierID, tariffID string, page, limit int) ([]ESIM, Pagination, error) {
	query = strings.TrimSpace(query)
	where := ` WHERE e.deleted_at IS NULL AND ($1='' OR e.iccid ILIKE '%'||$1||'%' OR e.imsi ILIKE '%'||$1||'%') AND ($2='' OR e.status=$2) AND ($3='' OR e.supplier_id::text=$3) AND ($4='' OR e.tariff_id::text=$4)`
	var total int
	if err := s.db.QueryRow(ctx, `SELECT count(*) FROM esims e`+where, query, status, supplierID, tariffID).Scan(&total); err != nil {
		return nil, Pagination{}, err
	}
	rows, err := s.db.Query(ctx, esimSelect+where+` ORDER BY e.created_at DESC LIMIT $5 OFFSET $6`, query, status, supplierID, tariffID, limit, (page-1)*limit)
	if err != nil {
		return nil, Pagination{}, err
	}
	defer rows.Close()
	items := make([]ESIM, 0)
	for rows.Next() {
		x, e := scanESIM(rows)
		if e != nil {
			return nil, Pagination{}, e
		}
		items = append(items, x)
	}
	return items, pagination(page, limit, total), rows.Err()
}
func (s *Store) ESIMByID(ctx context.Context, id string) (ESIM, error) {
	x, e := scanESIM(s.db.QueryRow(ctx, esimSelect+` WHERE e.id=$1 AND e.deleted_at IS NULL`, id))
	if errors.Is(e, pgx.ErrNoRows) {
		return ESIM{}, ErrNotFound
	}
	return x, e
}
func (s *Store) CreateESIM(ctx context.Context, in ESIMInput) (ESIM, error) {
	if in.Status == "" {
		in.Status = "available"
	}
	var id string
	e := s.db.QueryRow(ctx, `INSERT INTO esims(supplier_id,tariff_id,iccid,imsi,activation_code,smdp_address,qr_payload,status,traffic_total_mb,traffic_remaining_mb,activated_at,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING id`, in.SupplierID, in.TariffID, strings.TrimSpace(in.ICCID), strings.TrimSpace(in.IMSI), in.ActivationCode, in.SMDPAddress, in.QRPayload, in.Status, in.TrafficTotalMB, in.TrafficRemainingMB, in.ActivatedAt, in.ExpiresAt).Scan(&id)
	if e != nil {
		return ESIM{}, mapDBError(e)
	}
	return s.ESIMByID(ctx, id)
}
func (s *Store) UpdateESIM(ctx context.Context, id string, in ESIMInput) (ESIM, error) {
	if in.Status == "" {
		in.Status = "available"
	}
	tag, e := s.db.Exec(ctx, `UPDATE esims SET supplier_id=$2,tariff_id=$3,iccid=$4,imsi=$5,activation_code=$6,smdp_address=$7,qr_payload=$8,status=$9,traffic_total_mb=$10,traffic_remaining_mb=$11,activated_at=$12,expires_at=$13 WHERE id=$1 AND deleted_at IS NULL`, id, in.SupplierID, in.TariffID, strings.TrimSpace(in.ICCID), strings.TrimSpace(in.IMSI), in.ActivationCode, in.SMDPAddress, in.QRPayload, in.Status, in.TrafficTotalMB, in.TrafficRemainingMB, in.ActivatedAt, in.ExpiresAt)
	if e != nil {
		return ESIM{}, mapDBError(e)
	}
	if tag.RowsAffected() == 0 {
		return ESIM{}, ErrNotFound
	}
	return s.ESIMByID(ctx, id)
}
func (s *Store) DeleteESIM(ctx context.Context, id string) error {
	var used bool
	if e := s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM orders WHERE esim_id=$1)`, id).Scan(&used); e != nil {
		return e
	}
	if used {
		return fmt.Errorf("%w: eSIM is assigned to an order", ErrConflict)
	}
	tag, e := s.db.Exec(ctx, `UPDATE esims SET deleted_at=now(),status='disabled' WHERE id=$1 AND deleted_at IS NULL`, id)
	if e != nil {
		return e
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
func (s *Store) ImportESIMs(ctx context.Context, items []ESIMInput) (int, []string, error) {
	tx, e := s.db.Begin(ctx)
	if e != nil {
		return 0, nil, e
	}
	defer tx.Rollback(ctx)
	created := 0
	duplicates := make([]string, 0)
	for _, in := range items {
		if in.Status == "" {
			in.Status = "available"
		}
		tag, e := tx.Exec(ctx, `INSERT INTO esims(supplier_id,tariff_id,iccid,imsi,activation_code,smdp_address,qr_payload,status,traffic_total_mb,traffic_remaining_mb) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT(iccid) DO NOTHING`, in.SupplierID, in.TariffID, strings.TrimSpace(in.ICCID), strings.TrimSpace(in.IMSI), in.ActivationCode, in.SMDPAddress, in.QRPayload, in.Status, in.TrafficTotalMB, in.TrafficRemainingMB)
		if e != nil {
			return 0, nil, mapDBError(e)
		}
		if tag.RowsAffected() == 0 {
			duplicates = append(duplicates, in.ICCID)
		} else {
			created++
		}
	}
	if e = tx.Commit(ctx); e != nil {
		return 0, nil, e
	}
	return created, duplicates, nil
}

type OrderCreateInput struct {
	CustomerID    string         `json:"customerId"`
	Customer      *CustomerInput `json:"customer,omitempty"`
	TariffID      string         `json:"tariffId"`
	Price         *float64       `json:"price,omitempty"`
	Fee           float64        `json:"fee"`
	Tax           float64        `json:"tax"`
	PaymentMethod string         `json:"paymentMethod"`
	PaymentStatus string         `json:"paymentStatus"`
}
type OrderUpdateInput struct {
	Status             string `json:"status"`
	PaymentStatus      string `json:"paymentStatus"`
	PaymentMethod      string `json:"paymentMethod"`
	CancellationReason string `json:"cancellationReason"`
}

const orderSelect = `SELECT o.id,o.order_number,o.customer_id,c.name,c.phone,c.email,o.tariff_id,t.name,o.supplier_id,s.name,o.esim_id,
	COALESCE(e.iccid,''),COALESCE(e.activation_code,''),COALESCE(e.smdp_address,''),COALESCE(e.qr_payload,''),COALESCE(e.install_url,''),COALESCE(e.ios_tap_link,''),COALESCE(e.android_tap_link,''),o.operator_id,COALESCE(u.name,''),
	o.country,o.plan_name,o.cost::float8,o.price::float8,o.fee::float8,o.tax::float8,o.profit::float8,o.currency,o.payment_method,o.payment_status,o.status,
	o.activation_at,o.expires_at,o.cancelled_at,o.cancellation_reason,o.provider_order_id,o.provisioning_status,o.provisioning_error,o.created_at,o.updated_at
	FROM orders o JOIN customers c ON c.id=o.customer_id JOIN tariffs t ON t.id=o.tariff_id JOIN suppliers s ON s.id=o.supplier_id
	LEFT JOIN esims e ON e.id=o.esim_id LEFT JOIN users u ON u.id=o.operator_id`

func scanOrder(row pgx.Row) (Order, error) {
	var x Order
	e := row.Scan(&x.ID, &x.Number, &x.CustomerID, &x.CustomerName, &x.CustomerPhone, &x.CustomerEmail, &x.TariffID, &x.TariffName, &x.SupplierID, &x.SupplierName, &x.ESIMID, &x.ICCID, &x.ActivationCode, &x.SMDPAddress, &x.QRPayload, &x.InstallURL, &x.IOSTapLink, &x.AndroidTapLink, &x.OperatorID, &x.OperatorName, &x.Country, &x.PlanName, &x.Cost, &x.Price, &x.Fee, &x.Tax, &x.Profit, &x.Currency, &x.PaymentMethod, &x.PaymentStatus, &x.Status, &x.ActivationAt, &x.ExpiresAt, &x.CancelledAt, &x.CancellationReason, &x.ProviderOrderID, &x.ProvisioningStatus, &x.ProvisioningError, &x.CreatedAt, &x.UpdatedAt)
	return x, e
}
func (s *Store) ListOrders(ctx context.Context, query, status, paymentStatus, customerID string, from, to *time.Time, page, limit int) ([]Order, Pagination, error) {
	query = strings.TrimSpace(query)
	where := ` WHERE ($1='' OR o.order_number::text ILIKE '%'||$1||'%' OR c.name ILIKE '%'||$1||'%' OR c.phone ILIKE '%'||$1||'%' OR e.iccid ILIKE '%'||$1||'%') AND ($2='' OR o.status=$2) AND ($3='' OR o.payment_status=$3) AND ($4='' OR o.customer_id::text=$4) AND ($5::timestamptz IS NULL OR o.created_at >= $5) AND ($6::timestamptz IS NULL OR o.created_at < $6)`
	var total int
	if e := s.db.QueryRow(ctx, `SELECT count(*) FROM orders o JOIN customers c ON c.id=o.customer_id LEFT JOIN esims e ON e.id=o.esim_id`+where, query, status, paymentStatus, customerID, from, to).Scan(&total); e != nil {
		return nil, Pagination{}, e
	}
	rows, e := s.db.Query(ctx, orderSelect+where+` ORDER BY o.created_at DESC LIMIT $7 OFFSET $8`, query, status, paymentStatus, customerID, from, to, limit, (page-1)*limit)
	if e != nil {
		return nil, Pagination{}, e
	}
	defer rows.Close()
	items := make([]Order, 0)
	for rows.Next() {
		x, e := scanOrder(rows)
		if e != nil {
			return nil, Pagination{}, e
		}
		items = append(items, x)
	}
	return items, pagination(page, limit, total), rows.Err()
}
func (s *Store) OrderByID(ctx context.Context, id string) (Order, error) {
	x, e := scanOrder(s.db.QueryRow(ctx, orderSelect+` WHERE o.id=$1`, id))
	if errors.Is(e, pgx.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	return x, e
}
func (s *Store) CreateOrder(ctx context.Context, in OrderCreateInput, operatorID string) (Order, error) {
	tx, e := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		return Order{}, e
	}
	defer tx.Rollback(ctx)
	customerID := in.CustomerID
	if customerID == "" && in.Customer != nil {
		ci := in.Customer
		if ci.Status == "" {
			ci.Status = "active"
		}
		e = tx.QueryRow(ctx, `INSERT INTO customers(name,phone,email,country,status,notes) VALUES($1,$2,lower($3),$4,$5,$6) ON CONFLICT(phone) WHERE deleted_at IS NULL DO UPDATE SET name=EXCLUDED.name,email=EXCLUDED.email,country=EXCLUDED.country RETURNING id`, ci.Name, ci.Phone, ci.Email, ci.Country, ci.Status, ci.Notes).Scan(&customerID)
		if e != nil {
			return Order{}, mapDBError(e)
		}
	}
	if customerID == "" {
		return Order{}, errors.New("customerId or customer is required")
	}
	var supplierID, supplierName, country, tariffName, currency string
	var cost, price float64
	var validity int
	e = tx.QueryRow(ctx, `SELECT t.supplier_id,s.name,t.country,t.name,t.cost::float8,t.price::float8,t.currency,t.validity_days FROM tariffs t JOIN suppliers s ON s.id=t.supplier_id WHERE t.id=$1 AND t.status='active' AND t.deleted_at IS NULL`, in.TariffID).Scan(&supplierID, &supplierName, &country, &tariffName, &cost, &price, &currency, &validity)
	if errors.Is(e, pgx.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	if e != nil {
		return Order{}, e
	}
	if in.Price != nil {
		price = *in.Price
	}
	if price < cost+in.Fee+in.Tax {
		return Order{}, errors.New("price is below total cost")
	}
	var esimID string
	e = tx.QueryRow(ctx, `SELECT id FROM esims WHERE tariff_id=$1 AND status='available' AND deleted_at IS NULL ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT 1`, in.TariffID).Scan(&esimID)
	if errors.Is(e, pgx.ErrNoRows) {
		if supplierName != zesimoSupplierName {
			return Order{}, fmt.Errorf("%w: no available eSIM for tariff", ErrConflict)
		}
		esimID = ""
		e = nil
	}
	if e != nil {
		return Order{}, e
	}
	if in.PaymentMethod == "" {
		in.PaymentMethod = "cash"
	}
	if in.PaymentStatus == "" {
		in.PaymentStatus = "pending"
	}
	status := "pending_payment"
	esimStatus := "reserved"
	if in.PaymentStatus == "paid" {
		status = "esim_assigned"
		esimStatus = "sold"
	}
	provisioningStatus := "not_required"
	var esimValue any = esimID
	if supplierName == zesimoSupplierName && esimID == "" {
		provisioningStatus = "pending"
		esimValue = nil
		if in.PaymentStatus == "paid" {
			status = "paid"
		}
	}
	var orderID string
	e = tx.QueryRow(ctx, `INSERT INTO orders(customer_id,tariff_id,supplier_id,esim_id,operator_id,country,plan_name,cost,price,fee,tax,currency,payment_method,payment_status,status,provisioning_status) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16) RETURNING id`, customerID, in.TariffID, supplierID, esimValue, operatorID, country, tariffName, cost, price, in.Fee, in.Tax, currency, in.PaymentMethod, in.PaymentStatus, status, provisioningStatus).Scan(&orderID)
	if e != nil {
		return Order{}, mapDBError(e)
	}
	if esimID != "" {
		if _, e = tx.Exec(ctx, `UPDATE esims SET status=$2 WHERE id=$1`, esimID, esimStatus); e != nil {
			return Order{}, e
		}
	}
	if e = tx.Commit(ctx); e != nil {
		return Order{}, e
	}
	return s.OrderByID(ctx, orderID)
}

var allowedOrderTransitions = map[string]map[string]bool{"created": {"pending_payment": true, "cancelled": true}, "pending_payment": {"paid": true, "esim_assigned": true, "cancelled": true}, "paid": {"esim_assigned": true, "refunded": true}, "esim_assigned": {"activated": true, "cancelled": true, "refunded": true}, "activated": {"completed": true}, "completed": {}, "cancelled": {}, "refunded": {}}

func (s *Store) UpdateOrder(ctx context.Context, id string, in OrderUpdateInput) (Order, error) {
	tx, e := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		return Order{}, e
	}
	defer tx.Rollback(ctx)
	var current, payment, method, esimID string
	e = tx.QueryRow(ctx, `SELECT status,payment_status,payment_method,COALESCE(esim_id::text,'') FROM orders WHERE id=$1 FOR UPDATE`, id).Scan(&current, &payment, &method, &esimID)
	if errors.Is(e, pgx.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	if e != nil {
		return Order{}, e
	}
	if in.Status == "" {
		in.Status = current
	}
	if in.PaymentStatus == "" {
		in.PaymentStatus = payment
	}
	if in.PaymentMethod == "" {
		in.PaymentMethod = method
	}
	if in.Status != current && !allowedOrderTransitions[current][in.Status] {
		return Order{}, fmt.Errorf("invalid status transition from %s to %s", current, in.Status)
	}
	if in.PaymentStatus == "paid" && in.Status == "pending_payment" {
		if esimID == "" {
			in.Status = "paid"
		} else {
			in.Status = "esim_assigned"
		}
	}
	activationSQL := "activation_at"
	expiresSQL := "expires_at"
	cancelSQL := "cancelled_at"
	if in.Status == "activated" {
		activationSQL = "now()"
		expiresSQL = "now() + (SELECT validity_days FROM tariffs t JOIN orders ox ON ox.tariff_id=t.id WHERE ox.id=$1) * interval '1 day'"
	}
	if in.Status == "cancelled" {
		cancelSQL = "now()"
	}
	query := fmt.Sprintf(`UPDATE orders SET status=$2,payment_status=$3,payment_method=$4,cancellation_reason=$5,activation_at=%s,expires_at=%s,cancelled_at=%s WHERE id=$1`, activationSQL, expiresSQL, cancelSQL)
	if _, e = tx.Exec(ctx, query, id, in.Status, in.PaymentStatus, in.PaymentMethod, in.CancellationReason); e != nil {
		return Order{}, mapDBError(e)
	}
	if esimID != "" {
		es := "reserved"
		switch in.Status {
		case "esim_assigned":
			es = "sold"
		case "activated", "completed":
			es = "active"
		case "cancelled", "refunded":
			es = "available"
		}
		if in.Status == "activated" {
			_, e = tx.Exec(ctx, `UPDATE esims SET status=$2,activated_at=now(),expires_at=(SELECT expires_at FROM orders WHERE id=$3) WHERE id=$1`, esimID, es, id)
		} else {
			_, e = tx.Exec(ctx, `UPDATE esims SET status=$2 WHERE id=$1`, esimID, es)
		}
		if e != nil {
			return Order{}, e
		}
	}
	if e = tx.Commit(ctx); e != nil {
		return Order{}, e
	}
	return s.OrderByID(ctx, id)
}
