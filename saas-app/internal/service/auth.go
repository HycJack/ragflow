// Package service implements the business logic of the SaaS platform,
// mirroring RAGFlow's handler/service separation.
package service

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"saas/internal/crypto"
	"saas/internal/dao"
	"saas/internal/entity"
)

var (
	// ErrInvalidCredentials is returned on a bad login.
	ErrInvalidCredentials = errors.New("invalid email or password")
)

// AuthService handles registration, login and session management.
type AuthService struct {
	store    *dao.Store
	plans    *PlanService
	sessions sync.Map // token -> userID
}

// NewAuthService creates an AuthService.
func NewAuthService(store *dao.Store, plans *PlanService) *AuthService {
	return &AuthService{store: store, plans: plans}
}

// RegisterRequest carries registration input.
type RegisterRequest struct {
	Email    string `json:"email"`
	Nickname string `json:"nickname"`
	Password string `json:"password"`
	// InviteCode is optional; when present the new org is created and the
	// user is joined to it as owner.
	InviteCode string `json:"invite_code,omitempty"`
}

// Session is a bearer token + profile returned on login/register.
type Session struct {
	Token    string             `json:"token"`
	Profile  map[string]any      `json:"profile"`
}

// PlatformAdminEmail / PlatformAdminPassword are the default platform admin
// credentials. A production deployment should override these via environment
// variables (see SeedAdminIfAbsent).
const (
	PlatformAdminEmail    = "admin@ragflow.dev"
	PlatformAdminPassword = "admin123"
)

// SeedAdminIfAbsent creates the platform-level administrator on startup when
// no account with PlatformAdminEmail exists. It is idempotent and never
// overwrites an existing account. Credentials can be overridden via
// SAAS_ADMIN_EMAIL / SAAS_ADMIN_PASSWORD.
func (s *AuthService) SeedAdminIfAbsent() error {
	email := PlatformAdminEmail
	password := PlatformAdminPassword
	if v := os.Getenv("SAAS_ADMIN_EMAIL"); v != "" {
		email = v
	}
	if v := os.Getenv("SAAS_ADMIN_PASSWORD"); v != "" {
		password = v
	}
	if _, err := s.store.GetUserByEmail(email); err == nil {
		return nil // already exists
	}
	hash, err := crypto.HashPassword(password)
	if err != nil {
		return err
	}
	u := &entity.User{
		Email:           email,
		Nickname:        "Platform Admin",
		PasswordHash:    hash,
		Status:          "1",
		IsPlatformAdmin: true,
		CreatedAt:       time.Now(),
	}
	if err := s.store.CreateUser(u); err != nil {
		return err
	}
	return nil
}

// Register creates a user and optionally a personal organization.
func (s *AuthService) Register(req *RegisterRequest) (string, *entity.User, error) {
	if req.Email == "" || req.Nickname == "" || req.Password == "" {
		return "", nil, fmt.Errorf("email, nickname and password are required")
	}
	if _, err := s.store.GetUserByEmail(req.Email); err == nil {
		return "", nil, fmt.Errorf("email %s has already registered", req.Email)
	}
	hash, err := crypto.HashPassword(req.Password)
	if err != nil {
		return "", nil, err
	}
	u := &entity.User{
		Email:        req.Email,
		Nickname:     req.Nickname,
		PasswordHash: hash,
		Status:       "1",
		CreatedAt:    time.Now(),
	}
	if err := s.store.CreateUser(u); err != nil {
		return "", nil, err
	}
	return s.issueToken(u), u, nil
}

// LoginByEmail authenticates and issues a session token.
func (s *AuthService) LoginByEmail(email, password string) (string, *entity.User, error) {
	u, err := s.store.GetUserByEmail(email)
	if err != nil {
		return "", nil, ErrInvalidCredentials
	}
	if u.Status == "0" {
		return "", nil, fmt.Errorf("this account has been disabled")
	}
	if !crypto.VerifyPassword(u.PasswordHash, password) {
		return "", nil, ErrInvalidCredentials
	}
	return s.issueToken(u), u, nil
}

// GetUserByToken resolves a session token to a user.
func (s *AuthService) GetUserByToken(token string) (*entity.User, error) {
	idAny, ok := s.sessions.Load(token)
	if !ok {
		return nil, dao.ErrNotFound
	}
	return s.store.GetUserByID(idAny.(string))
}

// Logout invalidates a session.
func (s *AuthService) Logout(token string) {
	s.sessions.Delete(token)
}

func (s *AuthService) issueToken(u *entity.User) string {
	token := crypto.GenerateToken()
	s.sessions.Store(token, u.ID)
	return token
}

// SessionCount returns the number of active sessions (for observability/tests).
func (s *AuthService) SessionCount() int {
	n := 0
	s.sessions.Range(func(_, _ any) bool { n++; return true })
	return n
}