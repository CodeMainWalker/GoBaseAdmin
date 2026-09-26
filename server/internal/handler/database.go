package handler

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/CodeMainWalker/GoBaseAdmin/internal/config"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/response"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/store"
)

// DatabaseHandler 数据表维护（仅 MySQL）
type DatabaseHandler struct{}

func NewDatabaseHandler() *DatabaseHandler { return &DatabaseHandler{} }

// tableInfo 表结构摘要
type tableInfo struct {
	Name        string `gorm:"column:Name" json:"name"`
	Engine      string `gorm:"column:Engine" json:"engine"`
	Rows        int64  `gorm:"column:Rows" json:"rows"`
	DataLength  int64  `gorm:"column:Data_length" json:"data_length"`
	IndexLength int64  `gorm:"column:Index_length" json:"index_length"`
	Collation   string `gorm:"column:Collation" json:"collation"`
	Comment     string `gorm:"column:Comment" json:"comment"`
	CreateTime  string `gorm:"column:Create_time" json:"create_time"`
	DataFree    int64  `gorm:"column;Data_free" json:"data_free"`
}

// Index GET /core/database/index —— 数据表列表
func (h *DatabaseHandler) Index(c *gin.Context) {
	var tables []tableInfo
	err := store.DB.Raw(`SELECT TABLE_NAME AS Name, ENGINE AS Engine, TABLE_ROWS AS Rows,
		DATA_LENGTH AS Data_length, INDEX_LENGTH AS Index_length, TABLE_COLLATION AS Collation,
		TABLE_COMMENT AS Comment, CREATE_TIME AS Create_time, DATA_FREE AS Data_free
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = ? ORDER BY TABLE_NAME`, config.C.DBName).Scan(&tables).Error
	if err != nil {
		response.Fail(c, "读取数据表失败: "+err.Error())
		return
	}

	// 附加每表的数据行数与碎片
	out := make([]gin.H, 0, len(tables))
	for _, t := range tables {
		total := t.DataLength + t.IndexLength
		out = append(out, gin.H{
			"name":          t.Name,
			"engine":        t.Engine,
			"rows":          t.Rows,
			"data_length":   t.DataLength,
			"index_length":  t.IndexLength,
			"total_size":    total,
			"size_info":     formatBytes(total),
			"collation":     t.Collation,
			"comment":       t.Comment,
			"create_time":   t.CreateTime,
			"data_free":     t.DataFree,
			"fragment_info": formatBytes(t.DataFree),
		})
	}

	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "10")
	p := atoiSafe(page)
	l := atoiSafe(limit)
	if l <= 0 {
		l = 10
	}
	start := (p - 1) * l
	if start > len(out) {
		start = len(out)
	}
	end := start + l
	if end > len(out) {
		end = len(out)
	}

	response.Success(c, gin.H{
		"total":        len(out),
		"per_page":     l,
		"current_page": p,
		"last_page":    (len(out) + l - 1) / l,
		"data":         out[start:end],
		"has_more":     end < len(out),
	})
}

// DataSource GET /core/database/dataSource —— 数据源信息
func (h *DatabaseHandler) DataSource(c *gin.Context) {
	var version string
	store.DB.Raw(`SELECT VERSION()`).Scan(&version)
	response.Success(c, gin.H{
		"type":     "mysql",
		"host":     config.C.DBHost,
		"port":     config.C.DBPort,
		"database": config.C.DBName,
		"version":  version,
		"prefix":   config.C.DBPrefix,
		"charset":  config.C.DBCharset,
	})
}

// Detailed GET /core/database/detailed?name= —— 表结构详情
func (h *DatabaseHandler) Detailed(c *gin.Context) {
	name := c.Query("name")
	if name == "" {
		name = c.Query("table")
	}
	if !safeTableName(name) {
		response.Fail(c, "参数错误")
		return
	}

	type column struct {
		Field   string `gorm:"column:Field" json:"field"`
		Type    string `gorm:"column:Type" json:"type"`
		Null    string `gorm:"column:Null" json:"null"`
		Key     string `gorm:"column:Key" json:"key"`
		Default string `gorm:"column:Default" json:"default"`
		Extra   string `gorm:"column:Extra" json:"extra"`
		Comment string `gorm:"column:Comment" json:"comment"`
	}
	var cols []column
	err := store.DB.Raw(`SELECT COLUMN_NAME AS Field, COLUMN_TYPE AS Type, IS_NULLABLE AS `+"`Null`"+`,
		COLUMN_KEY AS `+"`Key`"+`, COLUMN_DEFAULT AS `+"`Default`"+`, EXTRA AS Extra, COLUMN_COMMENT AS Comment
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? ORDER BY ORDINAL_POSITION`,
		config.C.DBName, name).Scan(&cols).Error
	if err != nil {
		response.Fail(c, "读取表结构失败: "+err.Error())
		return
	}

	type index struct {
		KeyName string `gorm:"column:Key_name" json:"key_name"`
		Column  string `gorm:"column:Column_name" json:"column_name"`
		NonUniq int    `gorm:"column:Non_unique" json:"non_unique"`
	}
	var idx []index
	store.DB.Raw(`SELECT INDEX_NAME AS Key_name, COLUMN_NAME AS Column_name, NON_UNIQUE AS Non_unique
		FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? ORDER BY INDEX_NAME, SEQ_IN_INDEX`,
		config.C.DBName, name).Scan(&idx)

	response.Success(c, gin.H{
		"name":    name,
		"columns": cols,
		"indexes": idx,
	})
}

// Recycle GET /core/database/recycle —— 回收站（软删除数据概览）
func (h *DatabaseHandler) Recycle(c *gin.Context) {
	type row struct {
		TableName string `gorm:"column:table_name"`
		Count     int64  `gorm:"column:cnt"`
	}
	tables := softDeleteTables()
	out := make([]gin.H, 0, len(tables))
	for _, t := range tables {
		var cnt int64
		store.DB.Raw(fmt.Sprintf("SELECT COUNT(*) AS cnt FROM `%s` WHERE delete_time IS NOT NULL", t)).Scan(&cnt)
		if cnt > 0 {
			out = append(out, gin.H{"table_name": t, "count": cnt})
		}
	}
	response.Success(c, out)
}

// Delete DELETE /core/database/delete —— 彻底删除回收站数据
func (h *DatabaseHandler) Delete(c *gin.Context) {
	data := bindBody(c)
	table := toStr(data["table_name"])
	if !safeTableName(table) || !isSoftDeleteTable(table) {
		response.Fail(c, "参数错误")
		return
	}
	store.DB.Exec(fmt.Sprintf("DELETE FROM `%s` WHERE delete_time IS NOT NULL", table))
	response.Success(c, "操作成功")
}

// Recovery POST /core/database/recovery —— 从回收站恢复
func (h *DatabaseHandler) Recovery(c *gin.Context) {
	data := bindBody(c)
	table := toStr(data["table_name"])
	if !safeTableName(table) || !isSoftDeleteTable(table) {
		response.Fail(c, "参数错误")
		return
	}
	store.DB.Exec(fmt.Sprintf("UPDATE `%s` SET delete_time = NULL WHERE delete_time IS NOT NULL", table))
	response.Success(c, "操作成功")
}

// Optimize POST /core/database/optimize —— 优化表
func (h *DatabaseHandler) Optimize(c *gin.Context) {
	data := bindBody(c)
	name := toStr(data["name"])
	if name == "" {
		name = toStr(data["table_name"])
	}
	if !safeTableName(name) {
		response.Fail(c, "参数错误")
		return
	}
	if err := store.DB.Exec("OPTIMIZE TABLE `" + name + "`").Error; err != nil {
		response.Fail(c, "优化失败: "+err.Error())
		return
	}
	response.Success(c, "优化成功")
}

// Fragment POST /core/database/fragment —— 碎片整理
func (h *DatabaseHandler) Fragment(c *gin.Context) {
	data := bindBody(c)
	name := toStr(data["name"])
	if name == "" {
		name = toStr(data["table_name"])
	}
	if !safeTableName(name) {
		response.Fail(c, "参数错误")
		return
	}
	// MySQL 下 ALTER TABLE ... ENGINE=InnoDB 会重建表并回收碎片
	if err := store.DB.Exec("ALTER TABLE `" + name + "` ENGINE=InnoDB").Error; err != nil {
		response.Fail(c, "碎片整理失败: "+err.Error())
		return
	}
	response.Success(c, "整理成功")
}

// safeTableName 仅允许字母数字下划线，杜绝 SQL 注入
func safeTableName(s string) bool {
	if s == "" || len(s) > 64 {
		return false
	}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' {
			continue
		}
		return false
	}
	return true
}

// softDeleteTables 具备 delete_time 列的业务表
func softDeleteTables() []string {
	return []string{
		"sa_system_attachment", "sa_system_category", "sa_system_config",
		"sa_system_config_group", "sa_system_dept", "sa_system_dict_data",
		"sa_system_dict_type", "sa_system_login_log", "sa_system_mail",
		"sa_system_menu", "sa_system_oper_log", "sa_system_post",
		"sa_system_role", "sa_system_user",
	}
}

func isSoftDeleteTable(t string) bool {
	for _, v := range softDeleteTables() {
		if v == t {
			return true
		}
	}
	return false
}

func atoiSafe(s string) int {
	n := 0
	neg := false
	for i, ch := range strings.TrimSpace(s) {
		if i == 0 && ch == '-' {
			neg = true
			continue
		}
		if ch < '0' || ch > '9' {
			if i == 0 {
				return 0
			}
			break
		}
		n = n*10 + int(ch-'0')
	}
	if neg {
		return -n
	}
	return n
}

var _ *gorm.DB
