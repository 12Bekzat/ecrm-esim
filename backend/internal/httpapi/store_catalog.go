package httpapi

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

type CustomerInput struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Email   string `json:"email"`
	Country string `json:"country"`
	Status  string `json:"status"`
	Notes   string `json:"notes"`
}

const customerSelect = `SELECT c.id, 'C-'||lpad(c.customer_code::text,4,'0'), c.name,c.phone,c.email,c.country,c.status,c.notes,
	count(o.id)::int, COALESCE(sum(o.price) FILTER (WHERE o.status NOT IN ('cancelled','refunded')),0)::float8,
	COALESCE(sum(o.profit) FILTER (WHERE o.status NOT IN ('cancelled','refunded')),0)::float8, max(o.created_at),c.created_at,c.updated_at
	FROM customers c LEFT JOIN orders o ON o.customer_id=c.id`

func scanCustomer(row pgx.Row) (Customer, error) {
	var c Customer
	err := row.Scan(&c.ID, &c.Code, &c.Name, &c.Phone, &c.Email, &c.Country, &c.Status, &c.Notes, &c.Orders, &c.Total, &c.Profit, &c.LastOrder, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

func (s *Store) ListCustomers(ctx context.Context, query, status string, page, limit int) ([]Customer, Pagination, error) {
	query = strings.TrimSpace(query)
	var total int
	err := s.db.QueryRow(ctx, `SELECT count(*) FROM customers c WHERE c.deleted_at IS NULL
		AND ($1='' OR c.name ILIKE '%'||$1||'%' OR c.phone ILIKE '%'||$1||'%' OR c.email ILIKE '%'||$1||'%')
		AND ($2='' OR c.status=$2)`, query, status).Scan(&total)
	if err != nil {
		return nil, Pagination{}, err
	}
	rows, err := s.db.Query(ctx, customerSelect+` WHERE c.deleted_at IS NULL
		AND ($1='' OR c.name ILIKE '%'||$1||'%' OR c.phone ILIKE '%'||$1||'%' OR c.email ILIKE '%'||$1||'%')
		AND ($2='' OR c.status=$2)
		GROUP BY c.id ORDER BY c.created_at DESC LIMIT $3 OFFSET $4`, query, status, limit, (page-1)*limit)
	if err != nil {
		return nil, Pagination{}, err
	}
	defer rows.Close()
	items := make([]Customer, 0)
	for rows.Next() {
		item, err := scanCustomer(rows)
		if err != nil {
			return nil, Pagination{}, err
		}
		items = append(items, item)
	}
	return items, pagination(page, limit, total), rows.Err()
}

func (s *Store) CustomerByID(ctx context.Context, id string) (Customer, error) {
	c, err := scanCustomer(s.db.QueryRow(ctx, customerSelect+` WHERE c.id=$1 AND c.deleted_at IS NULL GROUP BY c.id`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Customer{}, ErrNotFound
	}
	return c, err
}

func (s *Store) CreateCustomer(ctx context.Context, in CustomerInput) (Customer, error) {
	if in.Status == "" {
		in.Status = "active"
	}
	var id string
	err := s.db.QueryRow(ctx, `INSERT INTO customers(name,phone,email,country,status,notes) VALUES($1,$2,lower($3),$4,$5,$6) RETURNING id`,
		strings.TrimSpace(in.Name), strings.TrimSpace(in.Phone), strings.TrimSpace(in.Email), strings.TrimSpace(in.Country), in.Status, strings.TrimSpace(in.Notes)).Scan(&id)
	if err != nil {
		return Customer{}, mapDBError(err)
	}
	return s.CustomerByID(ctx, id)
}

func (s *Store) UpdateCustomer(ctx context.Context, id string, in CustomerInput) (Customer, error) {
	if in.Status == "" {
		in.Status = "active"
	}
	tag, err := s.db.Exec(ctx, `UPDATE customers SET name=$2,phone=$3,email=lower($4),country=$5,status=$6,notes=$7 WHERE id=$1 AND deleted_at IS NULL`,
		id, strings.TrimSpace(in.Name), strings.TrimSpace(in.Phone), strings.TrimSpace(in.Email), strings.TrimSpace(in.Country), in.Status, strings.TrimSpace(in.Notes))
	if err != nil {
		return Customer{}, mapDBError(err)
	}
	if tag.RowsAffected() == 0 {
		return Customer{}, ErrNotFound
	}
	return s.CustomerByID(ctx, id)
}

func (s *Store) DeleteCustomer(ctx context.Context, id string) error {
	var orders int
	if err := s.db.QueryRow(ctx, `SELECT count(*) FROM orders WHERE customer_id=$1`, id).Scan(&orders); err != nil {
		return err
	}
	if orders > 0 {
		return fmt.Errorf("%w: customer has orders", ErrConflict)
	}
	tag, err := s.db.Exec(ctx, `UPDATE customers SET deleted_at=now() WHERE id=$1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type SupplierInput struct {
	Name                string  `json:"name"`
	ContactName         string  `json:"contactName"`
	ContactEmail        string  `json:"contactEmail"`
	ContactPhone        string  `json:"contactPhone"`
	APIBaseURL          string  `json:"apiBaseUrl"`
	APIKey              string  `json:"apiKey"`
	APIStatus           string  `json:"apiStatus"`
	Currency            string  `json:"currency"`
	Deposit             float64 `json:"deposit"`
	Balance             float64 `json:"balance"`
	Terms               string  `json:"terms"`
	Status              string  `json:"status"`
	LowBalanceThreshold float64 `json:"lowBalanceThreshold"`
}

const supplierSelect = `SELECT s.id,'S-'||lpad(s.supplier_code::text,2,'0'),s.name,s.contact_name,s.contact_email,s.contact_phone,s.api_base_url,s.api_status,s.currency,
	s.deposit::float8,s.balance::float8,s.terms,s.status,s.low_balance_threshold::float8,
	(SELECT count(DISTINCT t.country)::int FROM tariffs t WHERE t.supplier_id=s.id AND t.deleted_at IS NULL),
	(SELECT count(*)::int FROM esims e WHERE e.supplier_id=s.id AND e.deleted_at IS NULL),s.last_synced_at,s.created_at,s.updated_at FROM suppliers s`

func scanSupplier(row pgx.Row) (Supplier, error) {
	var x Supplier
	err := row.Scan(&x.ID, &x.Code, &x.Name, &x.ContactName, &x.ContactEmail, &x.ContactPhone, &x.APIBaseURL, &x.APIStatus, &x.Currency, &x.Deposit, &x.Balance, &x.Terms, &x.Status, &x.LowBalanceThreshold, &x.Countries, &x.ESIMs, &x.LastSyncedAt, &x.CreatedAt, &x.UpdatedAt)
	return x, err
}

func (s *Store) ListSuppliers(ctx context.Context, query string, page, limit int) ([]Supplier, Pagination, error) {
	query = strings.TrimSpace(query)
	var total int
	if err := s.db.QueryRow(ctx, `SELECT count(*) FROM suppliers WHERE deleted_at IS NULL AND ($1='' OR name ILIKE '%'||$1||'%')`, query).Scan(&total); err != nil {
		return nil, Pagination{}, err
	}
	rows, err := s.db.Query(ctx, supplierSelect+` WHERE s.deleted_at IS NULL AND ($1='' OR s.name ILIKE '%'||$1||'%') ORDER BY s.created_at DESC LIMIT $2 OFFSET $3`, query, limit, (page-1)*limit)
	if err != nil {
		return nil, Pagination{}, err
	}
	defer rows.Close()
	items := make([]Supplier, 0)
	for rows.Next() {
		x, e := scanSupplier(rows)
		if e != nil {
			return nil, Pagination{}, e
		}
		items = append(items, x)
	}
	return items, pagination(page, limit, total), rows.Err()
}
func (s *Store) SupplierByID(ctx context.Context, id string) (Supplier, error) {
	x, e := scanSupplier(s.db.QueryRow(ctx, supplierSelect+` WHERE s.id=$1 AND s.deleted_at IS NULL`, id))
	if errors.Is(e, pgx.ErrNoRows) {
		return Supplier{}, ErrNotFound
	}
	return x, e
}
func (s *Store) CreateSupplier(ctx context.Context, in SupplierInput) (Supplier, error) {
	defaultsSupplier(&in)
	var id string
	e := s.db.QueryRow(ctx, `INSERT INTO suppliers(name,contact_name,contact_email,contact_phone,api_base_url,api_key_encrypted,api_status,currency,deposit,balance,terms,status,low_balance_threshold) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) RETURNING id`, in.Name, in.ContactName, in.ContactEmail, in.ContactPhone, in.APIBaseURL, in.APIKey, in.APIStatus, in.Currency, in.Deposit, in.Balance, in.Terms, in.Status, in.LowBalanceThreshold).Scan(&id)
	if e != nil {
		return Supplier{}, mapDBError(e)
	}
	return s.SupplierByID(ctx, id)
}
func (s *Store) UpdateSupplier(ctx context.Context, id string, in SupplierInput) (Supplier, error) {
	defaultsSupplier(&in)
	tag, e := s.db.Exec(ctx, `UPDATE suppliers SET name=$2,contact_name=$3,contact_email=$4,contact_phone=$5,api_base_url=$6,api_key_encrypted=CASE WHEN $7='' THEN api_key_encrypted ELSE $7 END,api_status=$8,currency=$9,deposit=$10,balance=$11,terms=$12,status=$13,low_balance_threshold=$14 WHERE id=$1 AND deleted_at IS NULL`, id, in.Name, in.ContactName, in.ContactEmail, in.ContactPhone, in.APIBaseURL, in.APIKey, in.APIStatus, in.Currency, in.Deposit, in.Balance, in.Terms, in.Status, in.LowBalanceThreshold)
	if e != nil {
		return Supplier{}, mapDBError(e)
	}
	if tag.RowsAffected() == 0 {
		return Supplier{}, ErrNotFound
	}
	return s.SupplierByID(ctx, id)
}
func defaultsSupplier(in *SupplierInput) {
	if in.APIStatus == "" {
		in.APIStatus = "offline"
	}
	if in.Currency == "" {
		in.Currency = "KZT"
	}
	if in.Status == "" {
		in.Status = "active"
	}
	if in.LowBalanceThreshold == 0 {
		in.LowBalanceThreshold = 100000
	}
}
func (s *Store) DeleteSupplier(ctx context.Context, id string) error {
	var n int
	if e := s.db.QueryRow(ctx, `SELECT count(*) FROM tariffs WHERE supplier_id=$1 AND deleted_at IS NULL`, id).Scan(&n); e != nil {
		return e
	}
	if n > 0 {
		return fmt.Errorf("%w: supplier has tariffs", ErrConflict)
	}
	tag, e := s.db.Exec(ctx, `UPDATE suppliers SET deleted_at=now(),status='disabled' WHERE id=$1 AND deleted_at IS NULL`, id)
	if e != nil {
		return e
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type TariffInput struct {
	SupplierID       string  `json:"supplierId"`
	ExternalID       string  `json:"externalId"`
	Name             string  `json:"name"`
	Country          string  `json:"country"`
	Region           string  `json:"region"`
	DataMB           int64   `json:"dataMb"`
	ValidityDays     int     `json:"validityDays"`
	Cost             float64 `json:"cost"`
	Price            float64 `json:"price"`
	MinPrice         float64 `json:"minPrice"`
	RecommendedPrice float64 `json:"recommendedPrice"`
	Currency         string  `json:"currency"`
	Status           string  `json:"status"`
}

const tariffSelect = `SELECT t.id,'T-'||lpad(t.tariff_code::text,3,'0'),t.supplier_id,s.name,t.external_id,t.name,t.country,t.region,t.data_mb,t.validity_days,t.cost::float8,t.price::float8,t.min_price::float8,t.recommended_price::float8,t.currency,t.status,(SELECT count(*)::int FROM esims e WHERE e.tariff_id=t.id AND e.status='available' AND e.deleted_at IS NULL),t.created_at,t.updated_at FROM tariffs t JOIN suppliers s ON s.id=t.supplier_id`

func scanTariff(row pgx.Row) (Tariff, error) {
	var x Tariff
	e := row.Scan(&x.ID, &x.Code, &x.SupplierID, &x.SupplierName, &x.ExternalID, &x.Name, &x.Country, &x.Region, &x.DataMB, &x.ValidityDays, &x.Cost, &x.Price, &x.MinPrice, &x.RecommendedPrice, &x.Currency, &x.Status, &x.Stock, &x.CreatedAt, &x.UpdatedAt)
	return x, e
}
func (s *Store) ListTariffs(ctx context.Context, query, country, supplierID, status string, page, limit int) ([]Tariff, Pagination, error) {
	query = strings.TrimSpace(query)
	where := ` WHERE t.deleted_at IS NULL AND ($1='' OR t.name ILIKE '%'||$1||'%' OR t.country ILIKE '%'||$1||'%') AND ($2='' OR t.country=$2) AND ($3='' OR t.supplier_id::text=$3) AND ($4='' OR t.status=$4)`
	var total int
	if e := s.db.QueryRow(ctx, `SELECT count(*) FROM tariffs t`+where, query, country, supplierID, status).Scan(&total); e != nil {
		return nil, Pagination{}, e
	}
	rows, e := s.db.Query(ctx, tariffSelect+where+` ORDER BY t.created_at DESC LIMIT $5 OFFSET $6`, query, country, supplierID, status, limit, (page-1)*limit)
	if e != nil {
		return nil, Pagination{}, e
	}
	defer rows.Close()
	items := make([]Tariff, 0)
	for rows.Next() {
		x, e := scanTariff(rows)
		if e != nil {
			return nil, Pagination{}, e
		}
		items = append(items, x)
	}
	return items, pagination(page, limit, total), rows.Err()
}
func (s *Store) TariffByID(ctx context.Context, id string) (Tariff, error) {
	x, e := scanTariff(s.db.QueryRow(ctx, tariffSelect+` WHERE t.id=$1 AND t.deleted_at IS NULL`, id))
	if errors.Is(e, pgx.ErrNoRows) {
		return Tariff{}, ErrNotFound
	}
	return x, e
}
func (s *Store) CreateTariff(ctx context.Context, in TariffInput) (Tariff, error) {
	defaultsTariff(&in)
	var id string
	e := s.db.QueryRow(ctx, `INSERT INTO tariffs(supplier_id,external_id,name,country,region,data_mb,validity_days,cost,price,min_price,recommended_price,currency,status) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) RETURNING id`, in.SupplierID, in.ExternalID, in.Name, in.Country, in.Region, in.DataMB, in.ValidityDays, in.Cost, in.Price, in.MinPrice, in.RecommendedPrice, in.Currency, in.Status).Scan(&id)
	if e != nil {
		return Tariff{}, mapDBError(e)
	}
	return s.TariffByID(ctx, id)
}
func (s *Store) UpdateTariff(ctx context.Context, id string, in TariffInput) (Tariff, error) {
	defaultsTariff(&in)
	tag, e := s.db.Exec(ctx, `UPDATE tariffs SET supplier_id=$2,external_id=$3,name=$4,country=$5,region=$6,data_mb=$7,validity_days=$8,cost=$9,price=$10,min_price=$11,recommended_price=$12,currency=$13,status=$14 WHERE id=$1 AND deleted_at IS NULL`, id, in.SupplierID, in.ExternalID, in.Name, in.Country, in.Region, in.DataMB, in.ValidityDays, in.Cost, in.Price, in.MinPrice, in.RecommendedPrice, in.Currency, in.Status)
	if e != nil {
		return Tariff{}, mapDBError(e)
	}
	if tag.RowsAffected() == 0 {
		return Tariff{}, ErrNotFound
	}
	return s.TariffByID(ctx, id)
}
func defaultsTariff(in *TariffInput) {
	if in.Currency == "" {
		in.Currency = "KZT"
	}
	if in.Status == "" {
		in.Status = "active"
	}
}
func (s *Store) DeleteTariff(ctx context.Context, id string) error {
	var n int
	if e := s.db.QueryRow(ctx, `SELECT count(*) FROM orders WHERE tariff_id=$1`, id).Scan(&n); e != nil {
		return e
	}
	if n > 0 {
		return fmt.Errorf("%w: tariff has orders", ErrConflict)
	}
	tag, e := s.db.Exec(ctx, `UPDATE tariffs SET deleted_at=now(),status='inactive' WHERE id=$1 AND deleted_at IS NULL`, id)
	if e != nil {
		return e
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func pagination(page, limit, total int) Pagination {
	pages := 0
	if total > 0 {
		pages = (total + limit - 1) / limit
	}
	return Pagination{Page: page, Limit: limit, Total: total, TotalPages: pages}
}
