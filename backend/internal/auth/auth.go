package auth

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidToken = errors.New("invalid or expired token")
	ErrInvalidCreds = errors.New("invalid credentials")
)

type Claims struct {
	jwt.RegisteredClaims
	Role string `json:"role"`
}

type Authenticator struct {
	secret      []byte
	issuer      string
	duration    time.Duration
	masterHash  string
}

func New(secret, issuer string, duration time.Duration, masterPasswordHash string) *Authenticator {
	return &Authenticator{
		secret:     []byte(secret),
		issuer:     issuer,
		duration:   duration,
		masterHash: masterPasswordHash,
	}
}

func (a *Authenticator) GenerateToken(role string) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(a.duration)

	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    a.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
			ID:        fmt.Sprintf("%d", now.UnixNano()),
		},
		Role: role,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(a.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}
	return signed, exp, nil
}

func (a *Authenticator) ValidateToken(raw string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(raw, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return a.secret, nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func (a *Authenticator) AuthenticateMasterPassword(password string) bool {
	if a.masterHash == "" {
		// If no master password is configured, skip password auth
		// (JWT-only mode with pre-generated tokens e.g. via VIZHI_BOOTSTRAP_TOKEN)
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(a.masterHash), []byte(password)) == nil
}

func (a *Authenticator) SetMasterPassword(plaintext string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

// Middleware extracts and validates JWT from the Authorization header.
// Skips auth for the /auth/login endpoint.
func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/login" || r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			http.Error(w, `{"error":"invalid authorization format"}`, http.StatusUnauthorized)
			return
		}

		claims, err := a.ValidateToken(parts[1])
		if err != nil {
			http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		r.Header.Set("X-Vizhi-Role", claims.Role)
		next.ServeHTTP(w, r)
	})
}

// SecureCompare performs constant-time comparison to prevent timing attacks.
func SecureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

type LoginRequest struct {
	Password string `json:"password"`
}

type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}

func (a *Authenticator) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if !a.AuthenticateMasterPassword(req.Password) {
		http.Error(w, `{"error":"invalid password"}`, http.StatusUnauthorized)
		return
	}

	token, exp, err := a.GenerateToken("admin")
	if err != nil {
		http.Error(w, `{"error":"failed to generate token"}`, http.StatusInternalServerError)
		return
	}

	resp := LoginResponse{Token: token, ExpiresAt: exp.Unix()}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
