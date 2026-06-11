package payment

// Registry 支付渠道注册中心。
type Registry struct {
	m map[string]PaymentProvider
}

func NewRegistry() *Registry {
	return &Registry{m: make(map[string]PaymentProvider)}
}

func (r *Registry) Register(name string, p PaymentProvider) {
	r.m[name] = p
}

func (r *Registry) Get(name string) (PaymentProvider, bool) {
	p, ok := r.m[name]
	return p, ok
}

func (r *Registry) Enabled() []string {
	ids := make([]string, 0, len(r.m))
	for id := range r.m {
		ids = append(ids, id)
	}
	return ids
}
