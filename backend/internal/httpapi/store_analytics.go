package httpapi

import (
	"context"
	"time"
)

type DashboardKPI struct {
	Customers      int     `json:"customers"`
	ActiveESIMs    int     `json:"activeEsims"`
	ActiveOrders   int     `json:"activeOrders"`
	AvailableESIMs int     `json:"availableEsims"`
	Revenue        float64 `json:"revenue"`
	Cost           float64 `json:"cost"`
	Fees           float64 `json:"fees"`
	Tax            float64 `json:"tax"`
	Profit         float64 `json:"profit"`
	AverageCheck   float64 `json:"averageCheck"`
}
type SalesPoint struct {
	Date    string  `json:"date"`
	Orders  int     `json:"orders"`
	Revenue float64 `json:"revenue"`
	Cost    float64 `json:"cost"`
	Fees    float64 `json:"fees"`
	Tax     float64 `json:"tax"`
	Profit  float64 `json:"profit"`
}
type CountryShare struct {
	Name    string  `json:"name"`
	Orders  int     `json:"orders"`
	Revenue float64 `json:"revenue"`
	Share   float64 `json:"share"`
}
type SupplierPerformance struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Sold    int     `json:"sold"`
	Stock   int     `json:"stock"`
	Revenue float64 `json:"revenue"`
	Profit  float64 `json:"profit"`
}
type Dashboard struct {
	KPI       DashboardKPI          `json:"kpi"`
	Sales     []SalesPoint          `json:"sales"`
	Countries []CountryShare        `json:"countries"`
	Suppliers []SupplierPerformance `json:"suppliers"`
}

func (s *Store) Dashboard(ctx context.Context, from, to time.Time) (Dashboard, error) {
	var out Dashboard
	e := s.db.QueryRow(ctx, `SELECT
	(SELECT count(*)::int FROM customers WHERE deleted_at IS NULL),
	(SELECT count(*)::int FROM esims WHERE status='active' AND deleted_at IS NULL),
	(SELECT count(*)::int FROM orders WHERE status IN ('created','pending_payment','paid','esim_assigned','activated')),
	(SELECT count(*)::int FROM esims WHERE status='available' AND deleted_at IS NULL),
	COALESCE(sum(price) FILTER(WHERE status NOT IN('cancelled','refunded')),0)::float8,
	COALESCE(sum(cost) FILTER(WHERE status NOT IN('cancelled','refunded')),0)::float8,
	COALESCE(sum(fee) FILTER(WHERE status NOT IN('cancelled','refunded')),0)::float8,
	COALESCE(sum(tax) FILTER(WHERE status NOT IN('cancelled','refunded')),0)::float8,
	COALESCE(sum(profit) FILTER(WHERE status NOT IN('cancelled','refunded')),0)::float8,
	COALESCE(avg(price) FILTER(WHERE status NOT IN('cancelled','refunded')),0)::float8
	FROM orders WHERE created_at >= $1 AND created_at < $2`, from, to).Scan(&out.KPI.Customers, &out.KPI.ActiveESIMs, &out.KPI.ActiveOrders, &out.KPI.AvailableESIMs, &out.KPI.Revenue, &out.KPI.Cost, &out.KPI.Fees, &out.KPI.Tax, &out.KPI.Profit, &out.KPI.AverageCheck)
	if e != nil {
		return out, e
	}
	rows, e := s.db.Query(ctx, `SELECT to_char(d.day,'YYYY-MM-DD'),count(o.id)::int,COALESCE(sum(o.price),0)::float8,COALESCE(sum(o.cost),0)::float8,COALESCE(sum(o.fee),0)::float8,COALESCE(sum(o.tax),0)::float8,COALESCE(sum(o.profit),0)::float8 FROM generate_series(date_trunc('day',$1::timestamptz),date_trunc('day',$2::timestamptz-interval '1 second'),interval '1 day') d(day) LEFT JOIN orders o ON o.created_at>=d.day AND o.created_at<d.day+interval '1 day' AND o.status NOT IN('cancelled','refunded') GROUP BY d.day ORDER BY d.day`, from, to)
	if e != nil {
		return out, e
	}
	out.Sales = make([]SalesPoint, 0)
	for rows.Next() {
		var x SalesPoint
		if e = rows.Scan(&x.Date, &x.Orders, &x.Revenue, &x.Cost, &x.Fees, &x.Tax, &x.Profit); e != nil {
			rows.Close()
			return out, e
		}
		out.Sales = append(out.Sales, x)
	}
	rows.Close()
	rows, e = s.db.Query(ctx, `WITH x AS(SELECT country,count(*)::int orders,COALESCE(sum(price),0)::float8 revenue FROM orders WHERE created_at>=$1 AND created_at<$2 AND status NOT IN('cancelled','refunded') GROUP BY country), totals AS(SELECT COALESCE(sum(orders),0) n FROM x) SELECT country,orders,revenue,CASE WHEN totals.n=0 THEN 0 ELSE orders*100.0/totals.n END::float8 FROM x,totals ORDER BY orders DESC LIMIT 10`, from, to)
	if e != nil {
		return out, e
	}
	out.Countries = make([]CountryShare, 0)
	for rows.Next() {
		var x CountryShare
		if e = rows.Scan(&x.Name, &x.Orders, &x.Revenue, &x.Share); e != nil {
			rows.Close()
			return out, e
		}
		out.Countries = append(out.Countries, x)
	}
	rows.Close()
	rows, e = s.db.Query(ctx, `SELECT s.id,s.name,count(o.id) FILTER(WHERE o.status NOT IN('cancelled','refunded'))::int,(SELECT count(*)::int FROM esims e WHERE e.supplier_id=s.id AND e.status='available' AND e.deleted_at IS NULL),COALESCE(sum(o.price) FILTER(WHERE o.status NOT IN('cancelled','refunded')),0)::float8,COALESCE(sum(o.profit) FILTER(WHERE o.status NOT IN('cancelled','refunded')),0)::float8 FROM suppliers s LEFT JOIN orders o ON o.supplier_id=s.id AND o.created_at>=$1 AND o.created_at<$2 WHERE s.deleted_at IS NULL GROUP BY s.id ORDER BY 3 DESC`, from, to)
	if e != nil {
		return out, e
	}
	out.Suppliers = make([]SupplierPerformance, 0)
	for rows.Next() {
		var x SupplierPerformance
		if e = rows.Scan(&x.ID, &x.Name, &x.Sold, &x.Stock, &x.Revenue, &x.Profit); e != nil {
			rows.Close()
			return out, e
		}
		out.Suppliers = append(out.Suppliers, x)
	}
	rows.Close()
	return out, nil
}

type SearchItem struct {
	Type  string `json:"type"`
	ID    string `json:"id"`
	Code  string `json:"code"`
	Title string `json:"title"`
	Meta  string `json:"meta"`
}

func (s *Store) Search(ctx context.Context, q string, limit int) ([]SearchItem, error) {
	rows, e := s.db.Query(ctx, `SELECT type,id,code,title,meta FROM (
	SELECT 'order' type,o.id::text id,'#'||o.order_number code,c.name title,COALESCE(e.iccid,'') meta FROM orders o JOIN customers c ON c.id=o.customer_id LEFT JOIN esims e ON e.id=o.esim_id WHERE o.order_number::text ILIKE '%'||$1||'%' OR c.name ILIKE '%'||$1||'%' OR e.iccid ILIKE '%'||$1||'%'
	UNION ALL SELECT 'customer',c.id::text,'C-'||lpad(c.customer_code::text,4,'0'),c.name,c.phone FROM customers c WHERE c.deleted_at IS NULL AND (c.name ILIKE '%'||$1||'%' OR c.phone ILIKE '%'||$1||'%' OR c.email ILIKE '%'||$1||'%')
	UNION ALL SELECT 'esim',e.id::text,'ES-'||lpad(e.esim_code::text,4,'0'),e.iccid,e.status FROM esims e WHERE e.deleted_at IS NULL AND (e.iccid ILIKE '%'||$1||'%' OR e.imsi ILIKE '%'||$1||'%')
	UNION ALL SELECT 'tariff',t.id::text,'T-'||lpad(t.tariff_code::text,3,'0'),t.name,t.country FROM tariffs t WHERE t.deleted_at IS NULL AND (t.name ILIKE '%'||$1||'%' OR t.country ILIKE '%'||$1||'%')
	UNION ALL SELECT 'supplier',s.id::text,'S-'||lpad(s.supplier_code::text,2,'0'),s.name,s.api_status FROM suppliers s WHERE s.deleted_at IS NULL AND s.name ILIKE '%'||$1||'%'
	) x LIMIT $2`, q, limit)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	items := make([]SearchItem, 0)
	for rows.Next() {
		var x SearchItem
		if e = rows.Scan(&x.Type, &x.ID, &x.Code, &x.Title, &x.Meta); e != nil {
			return nil, e
		}
		items = append(items, x)
	}
	return items, rows.Err()
}

type CustomerStats struct {
	NewToday   int     `json:"newToday"`
	NewMonth   int     `json:"newMonth"`
	AverageLTV float64 `json:"averageLtv"`
}

func (s *Store) CustomerStats(ctx context.Context) (CustomerStats, error) {
	var x CustomerStats
	e := s.db.QueryRow(ctx, `SELECT count(*) FILTER(WHERE created_at>=date_trunc('day',now()))::int,count(*) FILTER(WHERE created_at>=date_trunc('month',now()))::int,COALESCE((SELECT avg(total) FROM(SELECT sum(price) total FROM orders WHERE status NOT IN('cancelled','refunded') GROUP BY customer_id)t),0)::float8 FROM customers WHERE deleted_at IS NULL`).Scan(&x.NewToday, &x.NewMonth, &x.AverageLTV)
	return x, e
}

type ESIMStats struct {
	Active    int `json:"active"`
	Available int `json:"available"`
	Sold      int `json:"sold"`
	Reserved  int `json:"reserved"`
	Expired   int `json:"expired"`
}

func (s *Store) ESIMStats(ctx context.Context) (ESIMStats, error) {
	var x ESIMStats
	e := s.db.QueryRow(ctx, `SELECT count(*)FILTER(WHERE status='active')::int,count(*)FILTER(WHERE status='available')::int,count(*)FILTER(WHERE status='sold')::int,count(*)FILTER(WHERE status='reserved')::int,count(*)FILTER(WHERE status='expired')::int FROM esims WHERE deleted_at IS NULL`).Scan(&x.Active, &x.Available, &x.Sold, &x.Reserved, &x.Expired)
	return x, e
}
