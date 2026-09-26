package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/CodeMainWalker/GoBaseAdmin/internal/model"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/query"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/response"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/store"
)

// LogHandler 日志管理
type LogHandler struct{}

func NewLogHandler() *LogHandler { return &LogHandler{} }

// GetLoginLogPageList GET /core/logs/getLoginLogPageList
func (h *LogHandler) GetLoginLogPageList(c *gin.Context) {
	p := query.ParseParams(c)
	db := store.DB.Model(&model.LoginLog{}).Where("delete_time IS NULL")
	db = applyLogFilters(db, c, []string{"username", "ip"}, []string{"status"})
	if t := c.Query("login_time"); t != "" {
		db = applyTimeRange(db, "login_time", t)
	}
	db = p.Apply(db, "id")
	var list []model.LoginLog
	response.Success(c, p.Paginate(db, &list))
}

// DeleteLoginLog DELETE /core/logs/deleteLoginLog
func (h *LogHandler) DeleteLoginLog(c *gin.Context) {
	ids := query.IDsFrom(getIDsParam(c))
	if len(ids) == 0 {
		response.Fail(c, "参数错误，请检查")
		return
	}
	now := model.Now()
	store.DB.Model(&model.LoginLog{}).Where("id IN ?", ids).Update("delete_time", now)
	response.Success(c, "操作成功")
}

// GetOperLogPageList GET /core/logs/getOperLogPageList
func (h *LogHandler) GetOperLogPageList(c *gin.Context) {
	p := query.ParseParams(c)
	db := store.DB.Model(&model.OperLog{}).Where("delete_time IS NULL")
	db = applyLogFilters(db, c, []string{"username", "service_name", "method", "ip"}, nil)
	if t := c.Query("create_time"); t != "" {
		db = applyTimeRange(db, "create_time", t)
	}
	db = p.Apply(db, "id")
	var list []model.OperLog
	response.Success(c, p.Paginate(db, &list))
}

// DeleteOperLog DELETE /core/logs/deleteOperLog
func (h *LogHandler) DeleteOperLog(c *gin.Context) {
	ids := query.IDsFrom(getIDsParam(c))
	if len(ids) == 0 {
		response.Fail(c, "参数错误，请检查")
		return
	}
	now := model.Now()
	store.DB.Model(&model.OperLog{}).Where("id IN ?", ids).Update("delete_time", now)
	response.Success(c, "操作成功")
}

// applyLogFilters likeCols 走模糊匹配，eqCols 走等值匹配
func applyLogFilters(db *gorm.DB, c *gin.Context, likeCols, eqCols []string) *gorm.DB {
	for _, col := range likeCols {
		if v := c.Query(col); v != "" {
			db = db.Where(col+" LIKE ?", "%"+v+"%")
		}
	}
	for _, col := range eqCols {
		if v := c.Query(col); v != "" {
			db = db.Where(col+" = ?", v)
		}
	}
	return db
}

// applyTimeRange 支持 "起,止" 区间或单值
func applyTimeRange(db *gorm.DB, field, t string) *gorm.DB {
	parts := strings.Split(t, ",")
	if len(parts) != 2 {
		return db.Where(field+" = ?", strings.TrimSpace(t))
	}
	return db.Where(field+" BETWEEN ? AND ?", strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
}
