package auth

import "context"

// ExternalUser 是各登录渠道归一化后的用户信息，业务层只认这个。
type ExternalUser struct {
	Provider   string // "google" / "apple" / "email"
	ExternalID string // 渠道侧唯一 ID（Google sub / Apple sub）
	Email      string
	Name       string
	AvatarURL  string
}

// AuthProvider 抽象第三方/邮箱登录，与 PaymentProvider 同套路。
// 新增登录方式 = 实现接口 + 注册，认证业务逻辑零改动。
type AuthProvider interface {
	ID() string // "google"
	// AuthURL 返回授权跳转 URL（OAuth 类）；邮箱类可返回空并走 SendCode。
	AuthURL(state string) string
	// Exchange 用回调参数换取归一化用户信息（OAuth 用 code；邮箱用 code+email）。
	Exchange(ctx context.Context, params map[string]string) (*ExternalUser, error)
}

// Registry 与支付一致：按 .env AUTH_PROVIDERS 注册启用的渠道。
type Registry struct {
	m map[string]AuthProvider
}

func NewRegistry() *Registry {
	return &Registry{m: make(map[string]AuthProvider)}
}

func (r *Registry) Register(p AuthProvider) {
	r.m[p.ID()] = p
}

func (r *Registry) Get(id string) (AuthProvider, bool) {
	p, ok := r.m[id]
	return p, ok
}

func (r *Registry) Enabled() []string {
	ids := make([]string, 0, len(r.m))
	for id := range r.m {
		ids = append(ids, id)
	}
	return ids
}
