// Package query 实现列表查询的检索、排序与分页，对外行为与上游 SaiAdmin 6.x 一致。
//
// 分页契约（关键）：
//
//	请求参数：page / limit / orderField / orderType / saiType
//	响应结构：{total, per_page, current_page, last_page, data, has_more}
//
// 该结构沿用上游 SaiAdmin 6.x 分页返回的字段命名，前端 useTable 依赖其字段名。
package query

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/CodeMainWalker/GoBaseAdmin/internal/store"
)

// Page 分页响应体
type Page struct {
	Total       int64       `json:"total"`
	PerPage     int         `json:"per_page"`
	CurrentPage int         `json:"current_page"`
	LastPage    int         `json:"last_page"`
	Data        interface{} `json:"data"`
	HasMore     bool        `json:"has_more"`
}

// Params 从请求中解析分页与排序参数
type Params struct {
	Page       int
	Limit      int
	OrderField string
	OrderType  string
	SaiType    string // list | all
}

// ParseParams 读取分页参数，缺省值与上游 SaiAdmin 6.x 保持一致
func ParseParams(c *gin.Context) Params {
	return Params{
		Page:       atoiDefault(c.Query("page"), 1),
		Limit:      atoiDefault(c.Query("limit"), 10),
		OrderField: c.Query("orderField"),
		OrderType:  c.Query("orderType"),
		SaiType:    defaultStr(c.Query("saiType"), "list"),
	}
}

// Apply 施加排序。orderField 为空时退化为默认字段（通常是主键）。
// orderType 默认 ASC，与上游 SaiAdmin 6.x 的默认排序方向一致。
func (p Params) Apply(db *gorm.DB, defaultField string) *gorm.DB {
	field := p.OrderField
	if field == "" {
		field = defaultField
	}
	if field == "" {
		field = "id"
	}
	dir := p.OrderType
	if dir == "" {
		dir = "ASC"
	}
	// 仅允许字母/下划线/点，避免排序注入
	if !safeIdent(field) {
		field = defaultField
		if field == "" {
			field = "id"
		}
	}
	if strings.EqualFold(dir, "DESC") {
		return db.Order(field + " DESC")
	}
	return db.Order(field + " ASC")
}

// Paginate 执行查询并按 saiType 返回对应形态（与上游 SaiAdmin 6.x 一致）：
//
//	saiType=all → 纯数组（上游取全量时的返回形态）
//	其它        → 分页对象 {total, per_page, current_page, last_page, data, has_more}
//
// 注意两者是**不同的结构**：前端取全量数据时直接把它当数组用
// （如 system/config 的 `api.configList({saiType:'all'}).then(data => data.map(...))`）。
// 若这里统一返回分页对象，前端会因 `data.map is not a function` 而拿不到数据。
//
// model 需为切片指针，如 *[]User。
func (p Params) Paginate(db *gorm.DB, model interface{}) interface{} {
	if p.SaiType == "all" {
		if err := db.Find(model).Error; err != nil {
			return []interface{}{}
		}
		return nonNilSlice(model)
	}
	return p.PageResult(db, model)
}

// PageResult 强制返回分页对象（需要给每一行附加字段时使用）。
func (p Params) PageResult(db *gorm.DB, model interface{}) Page {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Limit < 1 {
		p.Limit = 10
	}

	var total int64
	countDB := db.Session(&gorm.Session{})
	if err := countDB.Count(&total).Error; err != nil {
		total = 0
	}

	if err := db.Offset((p.Page - 1) * p.Limit).Limit(p.Limit).Find(model).Error; err != nil {
		return Page{Total: total, PerPage: p.Limit, CurrentPage: p.Page, Data: []interface{}{}}
	}

	lastPage := 0
	if p.Limit > 0 {
		lastPage = int((total + int64(p.Limit) - 1) / int64(p.Limit))
	}
	return Page{
		Total:       total,
		PerPage:     p.Limit,
		CurrentPage: p.Page,
		LastPage:    lastPage,
		Data:        nonNilSlice(model),
		HasMore:     p.Page < lastPage,
	}
}

// nonNilSlice 把空切片替换为长度 0 的切片，保证 JSON 序列化成 `[]` 而不是 `null`
// （上游空结果返回空数组，前端会直接对它调用 .map()）。
func nonNilSlice(model interface{}) interface{} {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return []interface{}{}
		}
		v = v.Elem()
	}
	if v.Kind() == reflect.Slice {
		if v.IsNil() {
			v = reflect.MakeSlice(v.Type(), 0, 0)
		}
		return v.Interface()
	}
	return model
}

// All 不分页查询（对应 getAll）
func (p Params) All(db *gorm.DB, model interface{}, defaultField string) interface{} {
	db = p.Apply(db, defaultField)
	if err := db.Find(model).Error; err != nil {
		return []interface{}{}
	}
	return nonNilSlice(model)
}

// ApplySearch 按非空值施加等值过滤（空值不参与条件，与上游搜索行为一致）
// conds 形如 {"username": 值, "status": 值}
func ApplySearch(db *gorm.DB, conds map[string]interface{}) *gorm.DB {
	for k, v := range conds {
		if isEmpty(v) {
			continue
		}
		if !safeIdent(k) {
			continue
		}
		db = db.Where(k+" = ?", v)
	}
	return db
}

func isEmpty(v interface{}) bool {
	if v == nil {
		return true
	}
	switch t := v.(type) {
	case string:
		return t == ""
	case []interface{}:
		return len(t) == 0
	}
	return false
}

// DateBetween 处理 create_time 一类的区间/等值搜索
func DateBetween(db *gorm.DB, field string, v interface{}) *gorm.DB {
	if !safeIdent(field) {
		return db
	}
	switch t := v.(type) {
	case string:
		if t == "" {
			return db
		}
		return db.Where(field+" = ?", t)
	case []interface{}:
		if len(t) == 2 {
			return db.Where(field+" BETWEEN ? AND ?", t[0], t[1])
		}
	case []string:
		if len(t) == 2 {
			return db.Where(field+" BETWEEN ? AND ?", t[0], t[1])
		}
	}
	return db
}

// LikeAny 对多个列做 OR LIKE（对应上游「a|b|c」多字段模糊搜索的语义）
func LikeAny(db *gorm.DB, cols []string, value string) *gorm.DB {
	if value == "" || len(cols) == 0 {
		return db
	}
	parts := make([]string, 0, len(cols))
	args := make([]interface{}, 0, len(cols))
	pattern := "%" + value + "%"
	for _, col := range cols {
		if !safeIdent(col) {
			continue
		}
		parts = append(parts, col+" LIKE ?")
		args = append(args, pattern)
	}
	if len(parts) == 0 {
		return db
	}
	return db.Where(strings.Join(parts, " OR "), args...)
}

// DB 便捷获取带软删除过滤的查询
func DB(table string) *gorm.DB {
	return store.DB.Table(table).Where(table + ".delete_time IS NULL")
}

func safeIdent(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func defaultStr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// IDsFrom 解析 ids 参数（可能是 "1"、"1,2,3" 或数组）
func IDsFrom(v interface{}) []uint64 {
	out := []uint64{}
	switch t := v.(type) {
	case string:
		for _, p := range strings.Split(t, ",") {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if n, err := strconv.ParseUint(p, 10, 64); err == nil {
				out = append(out, n)
			}
		}
	case float64:
		out = append(out, uint64(t))
	case int:
		out = append(out, uint64(t))
	case []interface{}:
		for _, it := range t {
			out = append(out, IDsFrom(it)...)
		}
	}
	return out
}
