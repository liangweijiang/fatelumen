package resource

// Registry 资源注册中心。启动时遍历注册的资源，自动挂载标准 REST 路由。
type Registry struct {
	resources map[string]Resource
}

func NewRegistry() *Registry {
	return &Registry{resources: make(map[string]Resource)}
}

func (r *Registry) Register(res Resource) {
	r.resources[res.Name()] = res
}

func (r *Registry) Get(name string) (Resource, bool) {
	res, ok := r.resources[name]
	return res, ok
}

func (r *Registry) All() []Resource {
	list := make([]Resource, 0, len(r.resources))
	for _, res := range r.resources {
		list = append(list, res)
	}
	return list
}
