package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"fatelumen/backend/internal/auth"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/jwt"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/repository"
)

// LoginResult 登录成功后返回的数据。
type LoginResult struct {
	UserID    uint64 `json:"user_id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Token     string `json:"token"`
}

// AuthService 认证业务逻辑。依赖接口，不依赖具体 SDK。
type AuthService struct {
	userRepo   *repository.UserRepo
	authReg    *auth.Registry
	jwtSecret  string
	jwtExpHrs  int
	stateStore map[string]time.Time
	mu         sync.Mutex
	log        *logger.Logger
}

func NewAuthService(
	userRepo *repository.UserRepo,
	authReg *auth.Registry,
	jwtSecret string,
	jwtExpHours int,
	log *logger.Logger,
) *AuthService {
	s := &AuthService{
		userRepo:   userRepo,
		authReg:    authReg,
		jwtSecret:  jwtSecret,
		jwtExpHrs:  jwtExpHours,
		stateStore: make(map[string]time.Time),
		log:        log,
	}
	go s.stateGC()
	return s
}

// GetLoginURL 返回 provider 的 OAuth 跳转 URL。
func (s *AuthService) GetLoginURL(ctx context.Context, providerID string) (string, error) {
	p, ok := s.authReg.Get(providerID)
	if !ok {
		return "", fmt.Errorf("unknown provider: %s", providerID)
	}
	state := s.genState()
	s.mu.Lock()
	s.stateStore[state] = time.Now()
	s.mu.Unlock()
	return p.AuthURL(state), nil
}

// HandleCallback 处理 OAuth 回调：验 state → 换用户信息 → upsert → 签发 JWT。
func (s *AuthService) HandleCallback(ctx context.Context, providerID string, code, state string) (*LoginResult, error) {
	// 校验 state
	s.mu.Lock()
	ts, ok := s.stateStore[state]
	delete(s.stateStore, state)
	s.mu.Unlock()
	if !ok || time.Since(ts) > 10*time.Minute {
		return nil, fmt.Errorf("invalid or expired state")
	}

	p, ok := s.authReg.Get(providerID)
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", providerID)
	}

	eu, err := p.Exchange(ctx, map[string]string{"code": code})
	if err != nil {
		return nil, fmt.Errorf("exchange failed: %w", err)
	}

	// upsert 用户
	user := &model.User{
		GoogleSub: eu.ExternalID,
		Email:     eu.Email,
		Name:      eu.Name,
		AvatarURL: eu.AvatarURL,
		Locale:    "en",
	}
	user, err = s.userRepo.UpsertByGoogleSub(user)
	if err != nil {
		return nil, fmt.Errorf("upsert user failed: %w", err)
	}

	// 签发 JWT，同时更新 current_token_id 实现单设备登录
	tokenID := genTokenID()
	token, err := jwt.Generate(s.jwtSecret, s.jwtExpHrs, user.ID, tokenID)
	if err != nil {
		return nil, fmt.Errorf("jwt generation failed: %w", err)
	}
	if err := s.userRepo.UpdateCurrentToken(user.ID, tokenID); err != nil {
		return nil, fmt.Errorf("update token id failed: %w", err)
	}

	return &LoginResult{
		UserID:    user.ID,
		Email:     user.Email,
		Name:      user.Name,
		AvatarURL: user.AvatarURL,
		Token:     token,
	}, nil
}

// Logout 登出：清除 current_token_id。
func (s *AuthService) Logout(ctx context.Context, userID uint64) error {
	return s.userRepo.ClearCurrentToken(userID)
}

// GetMe 获取当前用户信息。
func (s *AuthService) GetMe(ctx context.Context, userID uint64) (*model.User, error) {
	return s.userRepo.FindByID(userID)
}

// UpdateMe 更新当前用户信息（如 locale）。
func (s *AuthService) UpdateMe(ctx context.Context, userID uint64, patches map[string]interface{}) (*model.User, error) {
	allowed := map[string]bool{"locale": true, "name": true}
	for key := range patches {
		if !allowed[key] {
			delete(patches, key)
		}
	}
	if len(patches) == 0 {
		return s.userRepo.FindByID(userID)
	}
	if err := s.userRepo.UpdateFields(userID, patches); err != nil {
		return nil, err
	}
	return s.userRepo.FindByID(userID)
}

// genState 生成随机 state 字符串。
func (s *AuthService) genState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// stateGC 定期清理过期 state。
func (s *AuthService) stateGC() {
	for {
		time.Sleep(5 * time.Minute)
		s.mu.Lock()
		for k, ts := range s.stateStore {
			if time.Since(ts) > 15*time.Minute {
				delete(s.stateStore, k)
			}
		}
		s.mu.Unlock()
	}
}

func genTokenID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
