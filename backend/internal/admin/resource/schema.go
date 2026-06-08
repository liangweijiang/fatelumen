package resource

// EnumOption 枚举可选值。
type EnumOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// Field 字段 Schema（驱动前端渲染 + 约束后端白名单）。
type Field struct {
	Key        string       `json:"key"`        // 字段名（对应 DB 列 / JSON key）
	Label      string       `json:"label"`      // 列名/表单标签
	Type       string       `json:"type"`       // string/int/money/datetime/enum/bool/json/relation
	Enum       []EnumOption `json:"enum,omitempty"` // Type=enum 时的可选值
	Sortable   bool         `json:"sortable"`
	Filterable bool         `json:"filterable"`
	Searchable bool         `json:"searchable"`
	Editable   bool         `json:"editable"` // 详情页是否可编辑（进 Update 白名单）
	Hidden     bool         `json:"hidden"`   // 列表是否默认隐藏
}
