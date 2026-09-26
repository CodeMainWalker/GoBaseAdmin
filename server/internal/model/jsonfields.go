package model

import (
	"database/sql/driver"
	"encoding/json"
	"strings"
)

// JSONList 表示「以 JSON 文本存在单个列里」的列表，目前用于
// sa_system_config.config_select_data（select / radio 的候选项）。
//
// 为什么要单独做一个类型：
//   - 库里是 varchar/text，存的是 `[{"label":"本地上传","value":"1"}]` 这样的文本；
//   - 接口对外必须是**数组**：前端把该字段直接当成数组用
//     （Element Plus 的 el-select / el-radio-group 的 options 只接受 Array，
//     配置项编辑弹窗还会对它 v-for / push）。
//
// 该列在库中是 JSON 文本，这里读时解码、写时编码（沿用上游的字段语义），
// 空值对外统一为 []，非法内容退化为 []（不报错，避免整页拿不到数据）。
type JSONList []map[string]interface{}

// Scan 实现 sql.Scanner：把列里的文本解码成列表
func (l *JSONList) Scan(v interface{}) error {
	switch t := v.(type) {
	case nil:
		*l = JSONList{}
	case []byte:
		l.decode(string(t))
	case string:
		l.decode(t)
	default:
		*l = JSONList{}
	}
	return nil
}

// Value 实现 driver.Valuer：把列表编码为 JSON 文本。
// 空列表写空串，与库中既有数据（input 类型的配置项为空串）保持一致。
func (l JSONList) Value() (driver.Value, error) {
	if len(l) == 0 {
		return "", nil
	}
	b, err := json.Marshal([]map[string]interface{}(l))
	if err != nil {
		return "", nil
	}
	return string(b), nil
}

// MarshalJSON 保证空值序列化为 [] 而不是 null（前端会直接 .map / v-for）
func (l JSONList) MarshalJSON() ([]byte, error) {
	if l == nil {
		return []byte("[]"), nil
	}
	return json.Marshal([]map[string]interface{}(l))
}

// UnmarshalJSON 同时接受数组与「JSON 文本」两种入参（数据库原始形态）
func (l *JSONList) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "" || s == "null" {
		*l = JSONList{}
		return nil
	}
	if s[0] == '"' {
		var text string
		if err := json.Unmarshal(b, &text); err != nil {
			*l = JSONList{}
			return nil
		}
		l.decode(text)
		return nil
	}
	var raw []map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		*l = JSONList{}
		return nil
	}
	if raw == nil {
		raw = []map[string]interface{}{}
	}
	*l = JSONList(raw)
	return nil
}

func (l *JSONList) decode(s string) {
	s = strings.TrimSpace(s)
	if s == "" {
		*l = JSONList{}
		return
	}
	var raw []map[string]interface{}
	if err := json.Unmarshal([]byte(s), &raw); err != nil {
		*l = JSONList{}
		return
	}
	if raw == nil {
		raw = []map[string]interface{}{}
	}
	*l = JSONList(raw)
}

// EncodeJSONList 把请求体里的值（数组或 JSON 文本）转成可直接写库的文本。
// 供 pick + Updates(map) 这类绕过结构体的写入路径使用。
func EncodeJSONList(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		s = strings.TrimSpace(s)
		if s == "" || s == "null" {
			return ""
		}
		var probe []map[string]interface{}
		if json.Unmarshal([]byte(s), &probe) != nil {
			return ""
		}
		return s
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	var list JSONList
	if err := list.UnmarshalJSON(b); err != nil {
		return ""
	}
	val, err := list.Value()
	if err != nil {
		return ""
	}
	s, _ := val.(string)
	return s
}
