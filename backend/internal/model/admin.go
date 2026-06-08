package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// AdminUser 后台账号（与 C 端 users 完全独立）。
type AdminUser struct {
	ID           uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Username     string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"username"`
	PasswordHash string     `gorm:"type:varchar(255);not null" json:"-"`
	DisplayName  string     `gorm:"type:varchar(64);not null;default:''" json:"display_name"`
	RoleID       uint64     `gorm:"not null" json:"role_id"`
	TOTPSecret   string     `gorm:"type:varchar(64)" json:"-"`
	Status       string     `gorm:"type:varchar(16);not null;default:'active'" json:"status"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	CreatedAt    time.Time  `gorm:"not null" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"not null" json:"updated_at"`
}

func (AdminUser) TableName() string { return "admin_users" }

// Permissions 权限码数组（存 JSON）。
type Permissions []string

func (p Permissions) Value() (driver.Value, error) {
	return json.Marshal(p)
}

func (p *Permissions) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, p)
}

// AdminRole 后台角色 + 权限码（RBAC）。
type AdminRole struct {
	ID          uint64      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string      `gorm:"type:varchar(64);uniqueIndex;not null" json:"name"`
	Permissions Permissions `gorm:"type:json;not null" json:"permissions"`
	CreatedAt   time.Time   `gorm:"not null" json:"created_at"`
	UpdatedAt   time.Time   `gorm:"not null" json:"updated_at"`
}

func (AdminRole) TableName() string { return "admin_roles" }

// AdminAuditLog 后台操作审计日志（所有写操作自动记录）。
type AdminAuditLog struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	AdminID    uint64    `gorm:"not null;index" json:"admin_id"`
	AdminName  string    `gorm:"type:varchar(64);not null" json:"admin_name"`
	Action     string    `gorm:"type:varchar(64);not null" json:"action"`
	Resource   string    `gorm:"type:varchar(64);not null;index:idx_resource" json:"resource"`
	ResourceID string    `gorm:"type:varchar(64);index:idx_resource" json:"resource_id"`
	Detail     JSONRaw   `gorm:"type:json" json:"detail"`
	IP         string    `gorm:"type:varchar(64)" json:"ip"`
	CreatedAt  time.Time `gorm:"not null;index" json:"created_at"`
}

func (AdminAuditLog) TableName() string { return "admin_audit_log" }
