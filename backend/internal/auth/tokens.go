package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Claims struct {
	UserID string `json:"sub"`
	Role   string `json:"role"`
	Name   string `json:"name"`
	Issued int64  `json:"iat"`
	Expiry int64  `json:"exp"`
}

type Manager struct {
	secret []byte
	ttl    time.Duration
}

func NewManager(secret string, ttl time.Duration) *Manager {
	return &Manager{secret: []byte(secret), ttl: ttl}
}

func (m *Manager) Issue(userID, role, name string) (string, time.Time, error) {
	now := time.Now().UTC()
	expires := now.Add(m.ttl)
	header, _ := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	claims, err := json.Marshal(Claims{UserID: userID, Role: role, Name: name, Issued: now.Unix(), Expiry: expires.Unix()})
	if err != nil {
		return "", time.Time{}, err
	}
	unsigned := encode(header) + "." + encode(claims)
	return unsigned + "." + m.sign(unsigned), expires, nil
}

func (m *Manager) Parse(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, errors.New("invalid token")
	}
	unsigned := parts[0] + "." + parts[1]
	if !hmac.Equal([]byte(parts[2]), []byte(m.sign(unsigned))) {
		return Claims{}, errors.New("invalid signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, errors.New("invalid payload")
	}
	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return Claims{}, errors.New("invalid claims")
	}
	if claims.UserID == "" || claims.Expiry <= time.Now().Unix() {
		return Claims{}, errors.New("expired token")
	}
	return claims, nil
}

func RandomToken() (plain, hash string, err error) {
	buffer := make([]byte, 48)
	if _, err = rand.Read(buffer); err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}
	plain = base64.RawURLEncoding.EncodeToString(buffer)
	return plain, HashToken(plain), nil
}

func HashToken(token string) string {
	digest := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}

func (m *Manager) sign(value string) string {
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(value))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func encode(value []byte) string { return base64.RawURLEncoding.EncodeToString(value) }

func ParseBearer(header string) (string, error) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("bearer token required")
	}
	return parts[1], nil
}

func ParseLimit(value string, fallback, max int) int {
	n, err := strconv.Atoi(value)
	if err != nil || n < 1 {
		return fallback
	}
	if n > max {
		return max
	}
	return n
}
