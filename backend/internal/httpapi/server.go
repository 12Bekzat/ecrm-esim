package httpapi

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"ecrm/backend/internal/auth"
	"ecrm/backend/internal/config"
	"ecrm/backend/internal/providers/zesimo"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type Server struct {
	cfg    config.Config
	db     *pgxpool.Pool
	store  *Store
	tokens *auth.Manager
	log    *slog.Logger
	zesimo *zesimo.Client
}
type contextKey string

const userContextKey contextKey = "user"

func NewServer(cfg config.Config, db *pgxpool.Pool, log *slog.Logger) *Server {
	return &Server{cfg: cfg, db: db, store: NewStore(db), tokens: auth.NewManager(cfg.JWTSecret, cfg.AccessTokenTTL), log: log, zesimo: zesimo.New(cfg.ZesimoBaseURL, cfg.ZesimoAPIKey, cfg.ZesimoTimeout)}
}
func (s *Server) Handler() http.Handler {
	api := http.NewServeMux()
	api.HandleFunc("GET /api/v1/auth/me", s.me)
	api.HandleFunc("GET /api/v1/search", s.search)
	api.HandleFunc("GET /api/v1/dashboard", s.dashboard)
	api.HandleFunc("GET /api/v1/customers/stats", s.customerStats)
	api.HandleFunc("GET /api/v1/customers", s.listCustomers)
	api.HandleFunc("POST /api/v1/customers", s.writeRole(s.createCustomer, "admin", "operator"))
	api.HandleFunc("GET /api/v1/customers/{id}", s.getCustomer)
	api.HandleFunc("PUT /api/v1/customers/{id}", s.writeRole(s.updateCustomer, "admin", "operator"))
	api.HandleFunc("DELETE /api/v1/customers/{id}", s.writeRole(s.deleteCustomer, "admin"))
	api.HandleFunc("GET /api/v1/suppliers", s.listSuppliers)
	api.HandleFunc("POST /api/v1/suppliers", s.writeRole(s.createSupplier, "admin"))
	api.HandleFunc("GET /api/v1/suppliers/{id}", s.getSupplier)
	api.HandleFunc("PUT /api/v1/suppliers/{id}", s.writeRole(s.updateSupplier, "admin"))
	api.HandleFunc("DELETE /api/v1/suppliers/{id}", s.writeRole(s.deleteSupplier, "admin"))
	api.HandleFunc("POST /api/v1/suppliers/{id}/sync", s.writeRole(s.syncSupplier, "admin", "operator"))
	api.HandleFunc("POST /api/v1/suppliers/sync-all", s.writeRole(s.syncAllSuppliers, "admin", "operator"))
	api.HandleFunc("GET /api/v1/tariffs", s.listTariffs)
	api.HandleFunc("POST /api/v1/tariffs", s.writeRole(s.createTariff, "admin"))
	api.HandleFunc("GET /api/v1/tariffs/{id}", s.getTariff)
	api.HandleFunc("PUT /api/v1/tariffs/{id}", s.writeRole(s.updateTariff, "admin"))
	api.HandleFunc("DELETE /api/v1/tariffs/{id}", s.writeRole(s.deleteTariff, "admin"))
	api.HandleFunc("GET /api/v1/esims/stats", s.esimStats)
	api.HandleFunc("GET /api/v1/esims", s.listESIMs)
	api.HandleFunc("POST /api/v1/esims", s.writeRole(s.createESIM, "admin", "operator"))
	api.HandleFunc("POST /api/v1/esims/import", s.writeRole(s.importESIMs, "admin", "operator"))
	api.HandleFunc("GET /api/v1/esims/{id}", s.getESIM)
	api.HandleFunc("PUT /api/v1/esims/{id}", s.writeRole(s.updateESIM, "admin", "operator"))
	api.HandleFunc("DELETE /api/v1/esims/{id}", s.writeRole(s.deleteESIM, "admin"))
	api.HandleFunc("GET /api/v1/orders", s.listOrders)
	api.HandleFunc("POST /api/v1/orders", s.writeRole(s.createOrder, "admin", "operator"))
	api.HandleFunc("GET /api/v1/orders/{id}", s.getOrder)
	api.HandleFunc("PATCH /api/v1/orders/{id}", s.writeRole(s.updateOrder, "admin", "operator", "finance"))
	api.HandleFunc("POST /api/v1/orders/{id}/provision", s.writeRole(s.provisionOrder, "admin", "operator"))
	api.HandleFunc("GET /api/v1/integrations/zesimo", s.role(s.zesimoStatus, "admin", "operator", "finance"))
	api.HandleFunc("POST /api/v1/integrations/zesimo/sync", s.role(s.syncZesimo, "admin", "operator"))
	api.HandleFunc("GET /api/v1/employees", s.role(s.listEmployees, "admin"))
	api.HandleFunc("POST /api/v1/employees", s.role(s.createEmployee, "admin"))
	api.HandleFunc("GET /api/v1/employees/{id}", s.role(s.getEmployee, "admin"))
	api.HandleFunc("PUT /api/v1/employees/{id}", s.role(s.updateEmployee, "admin"))
	api.HandleFunc("DELETE /api/v1/employees/{id}", s.role(s.deleteEmployee, "admin"))
	api.HandleFunc("GET /api/v1/notifications", s.listNotifications)
	api.HandleFunc("PATCH /api/v1/notifications/read-all", s.readAllNotifications)
	api.HandleFunc("PATCH /api/v1/notifications/{id}/read", s.readNotification)
	api.HandleFunc("GET /api/v1/audit", s.role(s.listAudit, "admin"))
	api.HandleFunc("GET /api/v1/reports/sales.csv", s.role(s.salesCSV, "admin", "finance"))

	root := http.NewServeMux()
	root.HandleFunc("GET /health/live", s.liveness)
	root.HandleFunc("GET /health/ready", s.readiness)
	root.HandleFunc("POST /api/v1/auth/register", s.register)
	root.HandleFunc("POST /api/v1/auth/login", s.login)
	root.HandleFunc("POST /api/v1/auth/refresh", s.refresh)
	root.HandleFunc("POST /api/v1/auth/logout", s.logout)
	root.Handle("/api/v1/", s.authenticate(api))
	return s.recoverer(s.securityHeaders(s.cors(s.requestLogger(root))))
}

func (s *Server) liveness(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "time": time.Now().UTC()})
}
func (s *Server) readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if e := s.db.Ping(ctx); e != nil {
		writeError(w, http.StatusServiceUnavailable, "database_unavailable", "Database is unavailable", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, e := auth.ParseBearer(r.Header.Get("Authorization"))
		if e != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", e.Error(), nil)
			return
		}
		claims, e := s.tokens.Parse(raw)
		if e != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Access token is invalid or expired", nil)
			return
		}
		u, e := s.store.UserByID(r.Context(), claims.UserID)
		if e != nil || u.Status != "active" {
			writeError(w, http.StatusUnauthorized, "unauthorized", "User is unavailable", nil)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userContextKey, u)))
	})
}
func currentUser(r *http.Request) User { u, _ := r.Context().Value(userContextKey).(User); return u }
func (s *Server) role(next http.HandlerFunc, roles ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := currentUser(r)
		for _, role := range roles {
			if subtle.ConstantTimeCompare([]byte(u.Role), []byte(role)) == 1 {
				next(w, r)
				return
			}
		}
		writeError(w, http.StatusForbidden, "forbidden", "Insufficient permissions", nil)
	}
}
func (s *Server) writeRole(next http.HandlerFunc, roles ...string) http.HandlerFunc {
	return s.role(next, roles...)
}

type authRequest struct {
	Identity string `json:"identity"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Phone    string `json:"phone"`
}
type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var in authRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	if strings.TrimSpace(in.Name) == "" || !strings.Contains(in.Email, "@") || len(in.Password) < 10 {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", "Name, valid email and password of at least 10 characters are required", nil)
		return
	}
	u, e := s.store.RegisterOwner(r.Context(), strings.TrimSpace(in.Name), strings.TrimSpace(in.Email), strings.TrimSpace(in.Phone), in.Password)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	s.issueSession(w, r, u, http.StatusCreated)
}
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in authRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	identity := in.Identity
	if identity == "" {
		identity = in.Email
	}
	u, e := s.store.FindUserForLogin(r.Context(), identity)
	if e != nil || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(in.Password)) != nil {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "Invalid email/login or password", nil)
		return
	}
	if u.Status != "active" {
		writeError(w, http.StatusForbidden, "account_disabled", "Account is disabled", nil)
		return
	}
	s.store.TouchUser(r.Context(), u.ID)
	s.issueSession(w, r, u.User, http.StatusOK)
}
func (s *Server) issueSession(w http.ResponseWriter, r *http.Request, u User, status int) {
	access, expires, e := s.tokens.Issue(u.ID, u.Role, u.Name)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	refresh, hash, e := auth.RandomToken()
	if e != nil {
		respondStoreError(w, e)
		return
	}
	refreshExpiry := time.Now().UTC().Add(s.cfg.RefreshTokenTTL)
	if e = s.store.SaveRefreshToken(r.Context(), u.ID, hash, r.UserAgent(), clientIP(r), refreshExpiry); e != nil {
		respondStoreError(w, e)
		return
	}
	writeJSONStatus(w, status, map[string]any{"user": u, "accessToken": access, "accessExpiresAt": expires, "refreshToken": refresh, "refreshExpiresAt": refreshExpiry})
}
func (s *Server) refresh(w http.ResponseWriter, r *http.Request) {
	var in refreshRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	plain, newHash, e := auth.RandomToken()
	if e != nil {
		respondStoreError(w, e)
		return
	}
	expiry := time.Now().UTC().Add(s.cfg.RefreshTokenTTL)
	u, e := s.store.RotateRefreshToken(r.Context(), auth.HashToken(in.RefreshToken), newHash, r.UserAgent(), clientIP(r), expiry)
	if e != nil {
		writeError(w, http.StatusUnauthorized, "invalid_refresh_token", "Refresh token is invalid or expired", nil)
		return
	}
	access, accessExpiry, e := s.tokens.Issue(u.ID, u.Role, u.Name)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	writeJSONStatus(w, http.StatusOK, map[string]any{"user": u, "accessToken": access, "accessExpiresAt": accessExpiry, "refreshToken": plain, "refreshExpiresAt": expiry})
}
func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	var in refreshRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	_ = s.store.RevokeRefreshToken(r.Context(), auth.HashToken(in.RefreshToken))
	w.WriteHeader(http.StatusNoContent)
}
func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, currentUser(r))
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if e := dec.Decode(dst); e != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body is invalid", map[string]any{"detail": e.Error()})
		return false
	}
	return true
}
func writeJSON(w http.ResponseWriter, status int, data any) {
	writeJSONStatus(w, status, map[string]any{"data": data})
}
func writeJSONStatus(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
func writeError(w http.ResponseWriter, status int, code, message string, details any) {
	body := map[string]any{"error": map[string]any{"code": code, "message": message}}
	if details != nil {
		body["error"].(map[string]any)["details"] = details
	}
	writeJSONStatus(w, status, body)
}
func respondStoreError(w http.ResponseWriter, e error) {
	switch {
	case errors.Is(e, ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "Resource was not found", nil)
	case errors.Is(e, ErrConflict):
		writeError(w, http.StatusConflict, "conflict", e.Error(), nil)
	case strings.Contains(e.Error(), "invalid") || strings.Contains(e.Error(), "required") || strings.Contains(e.Error(), "password") || strings.Contains(e.Error(), "price"):
		writeError(w, http.StatusUnprocessableEntity, "validation_error", e.Error(), nil)
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
	}
}
func clientIP(r *http.Request) string {
	if value := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]); value != "" {
		return value
	}
	host, _, e := net.SplitHostPort(r.RemoteAddr)
	if e == nil {
		return host
	}
	return ""
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := false
		for _, item := range s.cfg.CORSOrigins {
			if item == "*" || item == origin {
				allowed = true
				break
			}
		}
		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}
func (s *Server) requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		s.log.Info("request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start), "ip", clientIP(r))
	})
}
func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if value := recover(); value != nil {
				s.log.Error("panic", "error", fmt.Sprint(value), "stack", string(debug.Stack()))
				writeError(w, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
