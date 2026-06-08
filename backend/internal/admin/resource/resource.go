package resource

// ListQuery 统一查询参数（列表接口），由框架从 query string 解析。
type ListQuery struct {
	Page     int                    // 页码，从 1 起
	PageSize int                    // 每页条数，默认 20，上限 100
	Sort     string                 // 排序字段，前缀 - 表示倒序，如 "-created_at"
	Search   string                 // 全局关键词（命中资源声明的 searchable 字段）
	Filters  map[string]interface{} // 字段精确/范围筛选
}

// ListResult 统一列表响应。
type ListResult struct {
	Items    interface{} `json:"items"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

// AdminContext 后台请求上下文（注入管理员信息）。
type AdminContext struct {
	AdminID   uint64
	AdminName string
	RoleID    uint64
	Perms     []string
	IP        string
}

// Action 自定义操作（超出标准 CRUD 的业务动作）。
type Action struct {
	Name    string // "refund" / "retry" / "ban"
	Label   string // 前端按钮文案
	Perm    string // 需要的权限码，如 "order:refund"
	Handler func(ctx *AdminContext, id string, params map[string]interface{}) (interface{}, error)
}

// Resource 每个业务资源实现此接口。
type Resource interface {
	// Name 资源标识（= 路由路径 + 权限前缀），如 "orders"
	Name() string
	// Schema 字段 Schema：驱动前端列表列/筛选器/表单，并约束后端筛选/排序白名单
	Schema() []Field
	// List 标准 CRUD（返回 nil 表示该资源不支持该操作）
	List(ctx *AdminContext, q ListQuery) (*ListResult, error)
	Detail(ctx *AdminContext, id string) (interface{}, error)
	Update(ctx *AdminContext, id string, patch map[string]interface{}) (interface{}, error)
	// Actions 自定义动作列表（退款、重试、封禁…）
	Actions() []Action
}
