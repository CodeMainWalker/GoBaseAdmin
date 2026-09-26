package model

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// DateTime 以 'Y-m-d H:i:s' 文本形式读写 datetime 列。
// 前端日期组件与该格式强绑定，因此不能用 Go 默认的 RFC3339 序列化
// （这也是数据库中的历史时间格式）。
type DateTime struct {
	time.Time
	Valid bool
}

const timeLayout = "2006-01-02 15:04:05"

// NewDateTime 由 time.Time 构造
func NewDateTime(t time.Time) DateTime {
	return DateTime{Time: t, Valid: true}
}

// MarshalJSON 输出 'Y-m-d H:i:s'；零值输出 null
func (t DateTime) MarshalJSON() ([]byte, error) {
	if !t.Valid {
		return []byte("null"), nil
	}
	return []byte(`"` + t.Time.Format(timeLayout) + `"`), nil
}

// Value 写入数据库
func (t DateTime) Value() (driver.Value, error) {
	if !t.Valid {
		return nil, nil
	}
	return t.Time.Format(timeLayout), nil
}

// Scan 从数据库读取
func (t *DateTime) Scan(v interface{}) error {
	if v == nil {
		t.Valid = false
		return nil
	}
	switch val := v.(type) {
	case time.Time:
		t.Time = val
		t.Valid = true
		return nil
	case []byte:
		return t.parse(string(val))
	case string:
		return t.parse(val)
	}
	return fmt.Errorf("DateTime: 无法解析 %T", v)
}

func (t *DateTime) parse(s string) error {
	if s == "" || s == "0000-00-00 00:00:00" {
		t.Valid = false
		return nil
	}
	layouts := []string{timeLayout, "2006-01-02T15:04:05Z07:00", "2006-01-02"}
	for _, l := range layouts {
		if parsed, err := time.ParseInLocation(l, s, time.Local); err == nil {
			t.Time = parsed
			t.Valid = true
			return nil
		}
	}
	t.Valid = false
	return nil
}

// Now 当前时间的 DateTime
func Now() DateTime { return NewDateTime(time.Now()) }

// DateTimePtr 返回可空时间的辅助构造
func DateTimePtr(t time.Time) *DateTime {
	p := NewDateTime(t)
	return &p
}
