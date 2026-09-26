package handler

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/CodeMainWalker/GoBaseAdmin/internal/menu"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/store"
)

// getIDParam 读取 id（兼容 query 与 body）
func getIDParam(c *gin.Context) string {
	if v := c.Query("id"); v != "" {
		return v
	}
	return c.PostForm("id")
}

// getIDsParam 读取 ids（兼容 query / body / JSON 数组）。
// 注意：前端删除走 DELETE 且把参数放在 body，Gin 默认不解析 DELETE body，
// 因此这里显式解析 JSON 与表单。
func getIDsParam(c *gin.Context) interface{} {
	if v, ok := c.GetPostForm("ids"); ok {
		return v
	}
	if v := c.Query("ids"); v != "" {
		return v
	}
	var body map[string]interface{}
	if err := c.ShouldBindBodyWithJSON(&body); err == nil {
		if v, ok := body["ids"]; ok {
			return v
		}
	}
	return ""
}

// bindBody 解析 JSON 或表单为 map
func bindBody(c *gin.Context) map[string]interface{} {
	out := map[string]interface{}{}
	ct := c.GetHeader("Content-Type")
	if strings.Contains(ct, "application/json") {
		_ = c.ShouldBindBodyWithJSON(&out)
		return out
	}
	_ = c.Request.ParseMultipartForm(32 << 20)
	for k, v := range c.Request.Form {
		if len(v) > 0 {
			out[k] = v[0]
		}
	}
	if len(out) == 0 {
		_ = c.ShouldBindBodyWithJSON(&out)
	}
	return out
}

// toUint 宽松地把任意值转为 uint64
func toUint(v interface{}) uint64 {
	switch t := v.(type) {
	case float64:
		return uint64(t)
	case int:
		return uint64(t)
	case int64:
		return uint64(t)
	case uint64:
		return t
	case string:
		n, _ := strconv.ParseUint(strings.TrimSpace(t), 10, 64)
		return n
	}
	return 0
}

// toInt 宽松转为 int
func toInt(v interface{}) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(t))
		return n
	}
	return 0
}

// toStr 宽松转为字符串
func toStr(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		if t {
			return "1"
		}
		return "0"
	case nil:
		return ""
	}
	return ""
}

// pick 从 map 中挑选指定字段，仅保留存在的键（避免把零值写入数据库）
func pick(data map[string]interface{}, keys ...string) map[string]interface{} {
	out := map[string]interface{}{}
	for _, k := range keys {
		if v, ok := data[k]; ok {
			out[k] = v
		}
	}
	return out
}

// setStructFields 把 map 中的值写入结构体指针（按 json tag 匹配）
func setStructFields(dst interface{}, data map[string]interface{}) {
	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Struct {
		return
	}
	rv = rv.Elem()
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		tag := f.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		name := strings.Split(tag, ",")[0]
		v, ok := data[name]
		if !ok {
			continue
		}
		field := rv.Field(i)
		if !field.CanSet() {
			continue
		}
		assign(field, v)
	}
}

func assign(field reflect.Value, v interface{}) {
	switch field.Kind() {
	case reflect.String:
		field.SetString(toStr(v))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		field.SetInt(int64(toInt(v)))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		field.SetUint(toUint(v))
	case reflect.Float32, reflect.Float64:
		switch t := v.(type) {
		case float64:
			field.SetFloat(t)
		default:
			field.SetFloat(float64(toInt(v)))
		}
	case reflect.Bool:
		field.SetBool(toInt(v) == 1 || toStr(v) == "true")
	case reflect.Ptr:
		// 支持 *uint64 之类的可空列
		if v == nil || toStr(v) == "" {
			return
		}
		elem := reflect.New(field.Type().Elem())
		assign(elem.Elem(), v)
		field.Set(elem)
	default:
		// 自定义 JSON 类型（如 model.JSONList）：用 json 往返赋值
		if !field.CanAddr() {
			return
		}
		u, ok := field.Addr().Interface().(json.Unmarshaler)
		if !ok {
			return
		}
		raw, err := json.Marshal(v)
		if err != nil {
			return
		}
		_ = u.UnmarshalJSON(raw)
	}
}

// storeDB 返回全局 GORM 实例的便捷封装
func storeDB() *gorm.DB { return store.DB }

// storeRaw 执行原生 SQL 查询
func storeRaw(sql string, args ...interface{}) *gorm.DB {
	return store.DB.Raw(sql, args...)
}

// md5Hex 计算 MD5（saveNetworkImage 的去重键，与上游 SaiAdmin 6.x 行为一致）
func md5Hex(b []byte) string {
	sum := md5.Sum(b)
	return hex.EncodeToString(sum[:])
}

// randUint16 生成 1..65535 的随机数（取值范围沿用上游 SaiAdmin 6.x 契约）
func randUint16() int {
	return rand.Intn(65535) + 1
}

// rawTimeLayouts 原生查询（Scan 到 map）时驱动返回的时间文本形态
var rawTimeLayouts = []string{
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02T15:04:05.999999999Z07:00",
	"2006-01-02",
}

// normalizeRawRow 把原生查询返回的一行规范化为前端期望的形态。
//
// 有两件事必须处理：
//  1. []byte → string（驱动可能返回字节切片，直接序列化会变成 base64）
//  2. time.Time → "Y-m-d H:i:s" 字符串。走原生 map 查询时绕过了 model.DateTime，
//     默认会输出 Go 的 RFC3339（2026-01-01T00:00:00+08:00），
//     与上游 SaiAdmin 6.x 的 'Y-m-d H:i:s' 文本时间格式不一致，前端日期列会显示异常。
//
// timeFields 指定哪些列需要按 'Y-m-d H:i:s' 文本时间格式化，避免误伤普通字符串。
func normalizeRawRow(raw map[string]interface{}, timeFields ...string) menu.Row {
	isTime := map[string]bool{}
	for _, f := range timeFields {
		isTime[f] = true
	}

	row := menu.Row{}
	for k, v := range raw {
		switch t := v.(type) {
		case []byte:
			row[k] = normalizeTimeValue(string(t), isTime[k])
		case string:
			row[k] = normalizeTimeValue(t, isTime[k])
		case time.Time:
			// 无论是否显式声明，time.Time 一律格式化为 'Y-m-d H:i:s' 形态
			row[k] = t.Format("2006-01-02 15:04:05")
		default:
			row[k] = v
		}
	}
	return row
}

// normalizeTimeValue 若该列是时间列，把文本统一为 "Y-m-d H:i:s"
func normalizeTimeValue(s string, wantTime bool) string {
	if !wantTime || s == "" {
		return s
	}
	for _, layout := range rawTimeLayouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t.Format("2006-01-02 15:04:05")
		}
	}
	return s
}

// normalizeRawRows 批量规范化
func normalizeRawRows(raws []map[string]interface{}, timeFields ...string) []menu.Row {
	rows := make([]menu.Row, 0, len(raws))
	for _, r := range raws {
		rows = append(rows, normalizeRawRow(r, timeFields...))
	}
	return rows
}
