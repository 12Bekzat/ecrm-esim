package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type EmployeeInput struct {
	Name        string   `json:"name"`
	Email       string   `json:"email"`
	Phone       string   `json:"phone"`
	Login       string   `json:"login"`
	Password    string   `json:"password"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
	Status      string   `json:"status"`
}

func scanUser(row pgx.Row) (User, error) {
	var u User
	var raw []byte
	e := row.Scan(&u.ID, &u.Code, &u.Name, &u.Email, &u.Phone, &u.Login, &u.Role, &raw, &u.Status, &u.LastSeenAt, &u.CreatedAt)
	_ = json.Unmarshal(raw, &u.Permissions)
	return u, e
}

const userSelect = `SELECT id,'E-'||lpad(employee_code::text,4,'0'),name,email,phone,login,role,permissions,status,last_seen_at,created_at FROM users`

func (s *Store) ListEmployees(ctx context.Context, query, role, status string, page, limit int) ([]User, Pagination, error) {
	query = strings.TrimSpace(query)
	where := ` WHERE ($1='' OR name ILIKE '%'||$1||'%' OR email ILIKE '%'||$1||'%' OR login ILIKE '%'||$1||'%') AND ($2='' OR role=$2) AND ($3='' OR status=$3)`
	var total int
	if e := s.db.QueryRow(ctx, `SELECT count(*) FROM users`+where, query, role, status).Scan(&total); e != nil {
		return nil, Pagination{}, e
	}
	rows, e := s.db.Query(ctx, userSelect+where+` ORDER BY created_at DESC LIMIT $4 OFFSET $5`, query, role, status, limit, (page-1)*limit)
	if e != nil {
		return nil, Pagination{}, e
	}
	defer rows.Close()
	items := make([]User, 0)
	for rows.Next() {
		x, e := scanUser(rows)
		if e != nil {
			return nil, Pagination{}, e
		}
		items = append(items, x)
	}
	return items, pagination(page, limit, total), rows.Err()
}
func (s *Store) CreateEmployee(ctx context.Context, in EmployeeInput) (User, error) {
	if in.Role == "" {
		in.Role = "operator"
	}
	if in.Status == "" {
		in.Status = "active"
	}
	if len(in.Password) < 10 {
		return User{}, errors.New("password must contain at least 10 characters")
	}
	hash, e := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if e != nil {
		return User{}, e
	}
	permissions, _ := json.Marshal(in.Permissions)
	var id string
	e = s.db.QueryRow(ctx, `INSERT INTO users(name,email,phone,login,password_hash,role,permissions,status) VALUES($1,lower($2),$3,lower($4),$5,$6,$7,$8) RETURNING id`, in.Name, in.Email, in.Phone, in.Login, string(hash), in.Role, permissions, in.Status).Scan(&id)
	if e != nil {
		return User{}, mapDBError(e)
	}
	return s.UserByID(ctx, id)
}
func (s *Store) UpdateEmployee(ctx context.Context, id string, in EmployeeInput) (User, error) {
	if in.Role == "" {
		in.Role = "operator"
	}
	if in.Status == "" {
		in.Status = "active"
	}
	permissions, _ := json.Marshal(in.Permissions)
	var hash *string
	if in.Password != "" {
		if len(in.Password) < 10 {
			return User{}, errors.New("password must contain at least 10 characters")
		}
		b, e := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
		if e != nil {
			return User{}, e
		}
		v := string(b)
		hash = &v
	}
	tag, e := s.db.Exec(ctx, `UPDATE users SET name=$2,email=lower($3),phone=$4,login=lower($5),password_hash=COALESCE($6,password_hash),role=$7,permissions=$8,status=$9 WHERE id=$1`, id, in.Name, in.Email, in.Phone, in.Login, hash, in.Role, permissions, in.Status)
	if e != nil {
		return User{}, mapDBError(e)
	}
	if tag.RowsAffected() == 0 {
		return User{}, ErrNotFound
	}
	return s.UserByID(ctx, id)
}
func (s *Store) DeleteEmployee(ctx context.Context, id string) error {
	tag, e := s.db.Exec(ctx, `UPDATE users SET status='disabled' WHERE id=$1`, id)
	if e != nil {
		return e
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	_, _ = s.db.Exec(ctx, `UPDATE refresh_tokens SET revoked_at=now() WHERE user_id=$1 AND revoked_at IS NULL`, id)
	return nil
}

type Notification struct {
	ID         string     `json:"id"`
	Type       string     `json:"type"`
	Message    string     `json:"message"`
	Level      string     `json:"level"`
	EntityType string     `json:"entityType"`
	EntityID   *string    `json:"entityId,omitempty"`
	ReadAt     *time.Time `json:"readAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
}

func (s *Store) ListNotifications(ctx context.Context, unreadOnly bool, page, limit int) ([]Notification, Pagination, error) {
	var total int
	if e := s.db.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE (NOT $1 OR read_at IS NULL)`, unreadOnly).Scan(&total); e != nil {
		return nil, Pagination{}, e
	}
	rows, e := s.db.Query(ctx, `SELECT id,type,message,level,entity_type,entity_id,read_at,created_at FROM notifications WHERE (NOT $1 OR read_at IS NULL) ORDER BY created_at DESC LIMIT $2 OFFSET $3`, unreadOnly, limit, (page-1)*limit)
	if e != nil {
		return nil, Pagination{}, e
	}
	defer rows.Close()
	items := make([]Notification, 0)
	for rows.Next() {
		var x Notification
		if e := rows.Scan(&x.ID, &x.Type, &x.Message, &x.Level, &x.EntityType, &x.EntityID, &x.ReadAt, &x.CreatedAt); e != nil {
			return nil, Pagination{}, e
		}
		items = append(items, x)
	}
	return items, pagination(page, limit, total), rows.Err()
}
func (s *Store) ReadNotification(ctx context.Context, id string) error {
	tag, e := s.db.Exec(ctx, `UPDATE notifications SET read_at=COALESCE(read_at,now()) WHERE id=$1`, id)
	if e != nil {
		return e
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
func (s *Store) ReadAllNotifications(ctx context.Context) error {
	_, e := s.db.Exec(ctx, `UPDATE notifications SET read_at=now() WHERE read_at IS NULL`)
	return e
}

type AuditLog struct {
	ID         string    `json:"id"`
	ActorID    *string   `json:"actorId,omitempty"`
	ActorName  string    `json:"actorName"`
	Action     string    `json:"action"`
	EntityType string    `json:"entityType"`
	EntityID   *string   `json:"entityId,omitempty"`
	OldValue   any       `json:"oldValue,omitempty"`
	NewValue   any       `json:"newValue,omitempty"`
	IPAddress  *string   `json:"ipAddress,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
}

func (s *Store) ListAudit(ctx context.Context, entityType, actorID string, page, limit int) ([]AuditLog, Pagination, error) {
	where := ` WHERE ($1='' OR entity_type=$1) AND ($2='' OR actor_id::text=$2)`
	var total int
	if e := s.db.QueryRow(ctx, `SELECT count(*) FROM audit_logs`+where, entityType, actorID).Scan(&total); e != nil {
		return nil, Pagination{}, e
	}
	rows, e := s.db.Query(ctx, `SELECT id,actor_id,actor_name,action,entity_type,entity_id,old_value,new_value,host(ip_address),created_at FROM audit_logs`+where+` ORDER BY created_at DESC LIMIT $3 OFFSET $4`, entityType, actorID, limit, (page-1)*limit)
	if e != nil {
		return nil, Pagination{}, e
	}
	defer rows.Close()
	items := make([]AuditLog, 0)
	for rows.Next() {
		var x AuditLog
		var oldRaw, newRaw []byte
		if e := rows.Scan(&x.ID, &x.ActorID, &x.ActorName, &x.Action, &x.EntityType, &x.EntityID, &oldRaw, &newRaw, &x.IPAddress, &x.CreatedAt); e != nil {
			return nil, Pagination{}, e
		}
		if len(oldRaw) > 0 {
			_ = json.Unmarshal(oldRaw, &x.OldValue)
		}
		if len(newRaw) > 0 {
			_ = json.Unmarshal(newRaw, &x.NewValue)
		}
		items = append(items, x)
	}
	return items, pagination(page, limit, total), rows.Err()
}

func (s *Store) SyncSupplier(ctx context.Context, id string) error {
	tx, e := s.db.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	var exists bool
	if e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM suppliers WHERE id=$1 AND deleted_at IS NULL)`, id).Scan(&exists); e != nil {
		return e
	}
	if !exists {
		return ErrNotFound
	}
	var logID string
	if e = tx.QueryRow(ctx, `INSERT INTO supplier_sync_logs(supplier_id,status) VALUES($1,'running') RETURNING id`, id).Scan(&logID); e != nil {
		return e
	}
	var tariffs, esims int
	if e = tx.QueryRow(ctx, `SELECT count(*) FROM tariffs WHERE supplier_id=$1 AND deleted_at IS NULL`, id).Scan(&tariffs); e != nil {
		return e
	}
	if e = tx.QueryRow(ctx, `SELECT count(*) FROM esims WHERE supplier_id=$1 AND deleted_at IS NULL`, id).Scan(&esims); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, `UPDATE suppliers SET last_synced_at=now(),api_status=CASE WHEN balance<=low_balance_threshold THEN 'warning' ELSE 'online' END,status=CASE WHEN balance<=low_balance_threshold THEN 'low_balance' ELSE status END WHERE id=$1`, id); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, `UPDATE supplier_sync_logs SET status='success',tariffs_received=$2,esims_received=$3,finished_at=now() WHERE id=$1`, logID, tariffs, esims); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
