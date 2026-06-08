package payment

// Registry 支付渠道注册中心。
type Registry struct {
	m map[string]PaymentProvider
}

func NewRegistry() *Registry {
	return &Registry{m: make(map[string]PaymentProvider)}
}

func (r *Registry) Register(p PaymentProvider) {
	r.m[p.ID()] = p
}

func (r *Registry) Get(id string) (PaymentProvider, bool) {
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
