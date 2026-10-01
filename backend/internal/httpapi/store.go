package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrNotFound = errors.New("resource not found")
	ErrConflict = errors.New("resource already exists")
)

type Store struct{ db *pgxpool.Pool }

func NewStore(db *pgxpool.Pool) *Store { return &Store{db: db} }

func (s *Store) BootstrapAdmin(ctx context.Context, email, password, name string) error {
	if email == "" && password == "" {
		return nil
	}
	if email == "" || len(password) < 10 {
		return errors.New("both BOOTSTRAP_ADMIN_EMAIL and a password of at least 10 characters are required")
	}
	var count int
	if err := s.db.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	login := strings.Split(email, "@")[0]
	_, err = s.db.Exec(ctx, `INSERT INTO users(name,email,login,password_hash,role,permissions)
		VALUES($1,$2,$3,$4,'admin','["*"]'::jsonb)`, name, email, login, string(hash))
	return err
}

type authUser struct {
	User
	PasswordHash string
}

func (s *Store) FindUserForLogin(ctx context.Context, identity string) (authUser, error) {
	var u authUser
	var permissions []byte
	err := s.db.QueryRow(ctx, `SELECT id, 'E-'||lpad(employee_code::text,4,'0'), name,email,phone,login,role,permissions,status,last_seen_at,created_at,password_hash
		FROM users WHERE lower(email)=lower($1) OR lower(login)=lower($1) LIMIT 1`, strings.TrimSpace(identity)).Scan(
		&u.ID, &u.Code, &u.Name, &u.Email, &u.Phone, &u.Login, &u.Role, &permissions, &u.Status, &u.LastSeenAt, &u.CreatedAt, &u.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return authUser{}, ErrNotFound
	}
	if err != nil {
		return authUser{}, err
	}
	_ = json.Unmarshal(permissions, &u.Permissions)
	return u, nil
}

func (s *Store) UserByID(ctx context.Context, id string) (User, error) {
	var u User
	var permissions []byte
	err := s.db.QueryRow(ctx, `SELECT id, 'E-'||lpad(employee_code::text,4,'0'), name,email,phone,login,role,permissions,status,last_seen_at,created_at
		FROM users WHERE id=$1`, id).Scan(&u.ID, &u.Code, &u.Name, &u.Email, &u.Phone, &u.Login, &u.Role, &permissions, &u.Status, &u.LastSeenAt, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	_ = json.Unmarshal(permissions, &u.Permissions)
	return u, err
}

func (s *Store) RegisterOwner(ctx context.Context, name, email, phone, password string) (User, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback(ctx)
	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&count); err != nil {
		return User{}, err
	}
	if count > 0 {
		return User{}, ErrConflict
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}
	login := strings.Split(email, "@")[0]
	var id string
	if err := tx.QueryRow(ctx, `INSERT INTO users(name,email,phone,login,password_hash,role,permissions)
		VALUES($1,lower($2),$3,$4,$5,'admin','["*"]'::jsonb) RETURNING id`, name, email, phone, login, string(hash)).Scan(&id); err != nil {
		return User{}, mapDBError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return User{}, err
	}
	return s.UserByID(ctx, id)
}

func (s *Store) TouchUser(ctx context.Context, id string) {
	_, _ = s.db.Exec(ctx, `UPDATE users SET last_seen_at=now() WHERE id=$1`, id)
}

func (s *Store) SaveRefreshToken(ctx context.Context, userID, tokenHash, userAgent, ip string, expiresAt time.Time) error {
	_, err := s.db.Exec(ctx, `INSERT INTO refresh_tokens(user_id,token_hash,user_agent,ip_address,expires_at)
		VALUES($1,$2,$3,NULLIF($4,'')::inet,$5)`, userID, tokenHash, userAgent, ip, expiresAt)
	return err
}

func (s *Store) RotateRefreshToken(ctx context.Context, oldHash, newHash, userAgent, ip string, expiresAt time.Time) (User, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback(ctx)
	var userID string
	err = tx.QueryRow(ctx, `UPDATE refresh_tokens SET revoked_at=now()
		WHERE token_hash=$1 AND revoked_at IS NULL AND expires_at>now() RETURNING user_id`, oldHash).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO refresh_tokens(user_id,token_hash,user_agent,ip_address,expires_at)
		VALUES($1,$2,$3,NULLIF($4,'')::inet,$5)`, userID, newHash, userAgent, ip, expiresAt); err != nil {
		return User{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return User{}, err
	}
	return s.UserByID(ctx, userID)
}

func (s *Store) RevokeRefreshToken(ctx context.Context, hash string) error {
	_, err := s.db.Exec(ctx, `UPDATE refresh_tokens SET revoked_at=COALESCE(revoked_at,now()) WHERE token_hash=$1`, hash)
	return err
}

func (s *Store) Audit(ctx context.Context, actor User, action, entityType, entityID string, oldValue, newValue any, ip string) {
	oldJSON, _ := json.Marshal(oldValue)
	newJSON, _ := json.Marshal(newValue)
	_, _ = s.db.Exec(ctx, `INSERT INTO audit_logs(actor_id,actor_name,action,entity_type,entity_id,old_value,new_value,ip_address)
		VALUES($1,$2,$3,$4,NULLIF($5,'')::uuid,$6,$7,NULLIF($8,'')::inet)`, actor.ID, actor.Name, action, entityType, entityID, nullJSON(oldJSON), nullJSON(newJSON), ip)
}

func nullJSON(value []byte) any {
	if string(value) == "null" {
		return nil
	}
	return value
}

func mapDBError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return fmt.Errorf("%w: %s", ErrConflict, pgErr.ConstraintName)
		case "23503":
			return fmt.Errorf("invalid relation: %w", err)
		case "23514", "22P02":
			return fmt.Errorf("invalid value: %w", err)
		}
	}
	return err
}
