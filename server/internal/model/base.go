package model

import (
	"gorm.io/gorm"
)

// Base 公共字段（字段形态与库中既有表结构一致）：
//   - create_time / update_time 由 GORM 钩子写入 'Y-m-d H:i:s' 文本时间
//   - delete_time 实现软删除，且**不出现在 JSON 输出中**
//   - created_by / updated_by 由当前登录用户自动填充
type Base struct {
	ID         uint      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	CreatedBy  *int      `gorm:"column:created_by" json:"created_by"`
	UpdatedBy  *int      `gorm:"column:updated_by" json:"updated_by"`
	CreateTime DateTime  `gorm:"column:create_time" json:"create_time"`
	UpdateTime DateTime  `gorm:"column:update_time" json:"update_time"`
	DeleteTime *DateTime `gorm:"column:delete_time" json:"-"`
}

// 当前操作者 ID，由中间件在每请求开始时设置、结束时清理。
// 与上游 SaiAdmin 6.x 等价：从当前请求的 JWT 上下文取登录用户 ID。
var currentUserID *int

// SetCurrentUser 设置当前登录用户 ID
func SetCurrentUser(id *int) { currentUserID = id }

// CurrentUser 读取当前登录用户 ID
func CurrentUser() *int { return currentUserID }

// BeforeCreate 写入 created_by / create_time / update_time
func (b *Base) BeforeCreate(tx *gorm.DB) error {
	if currentUserID != nil && b.CreatedBy == nil {
		v := *currentUserID
		b.CreatedBy = &v
	}
	now := Now()
	if !b.CreateTime.Valid {
		b.CreateTime = now
	}
	b.UpdateTime = now
	return nil
}

// BeforeUpdate 写入 updated_by / update_time
func (b *Base) BeforeUpdate(tx *gorm.DB) error {
	if currentUserID != nil {
		v := *currentUserID
		b.UpdatedBy = &v
	}
	b.UpdateTime = Now()
	return nil
}

// SoftDeleteQuery 为支持软删除的模型加上 delete_time IS NULL 过滤。
// 这里改用显式 scope，避免依赖 gorm.DeletedAt（其零值语义与本项目不同）。
type SoftDeleteBase struct {
	Base
}

// ScopeNotDeleted 未删除过滤
func ScopeNotDeleted(db *gorm.DB) *gorm.DB {
	return db.Where("delete_time IS NULL")
}
