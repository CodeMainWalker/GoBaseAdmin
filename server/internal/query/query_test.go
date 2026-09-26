package query

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeRow struct {
	ID uint64 `json:"id"`
}

// 空结果必须序列化成 []，否则前端 data.map() 会抛错
func TestNonNilSliceEmpty(t *testing.T) {
	var rows []fakeRow
	b, err := json.Marshal(nonNilSlice(&rows))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "[]" {
		t.Fatalf("空切片序列化为 %s，期望 []", string(b))
	}
}

func TestNonNilSliceKeepsData(t *testing.T) {
	rows := []fakeRow{{ID: 1}}
	b, err := json.Marshal(nonNilSlice(&rows))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `[{"id":1}]` {
		t.Fatalf("序列化结果 %s 不正确", string(b))
	}
}

func TestNonNilSliceNilPointer(t *testing.T) {
	var rows *[]fakeRow
	b, err := json.Marshal(nonNilSlice(rows))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "[]" {
		t.Fatalf("nil 指针序列化为 %s，期望 []", string(b))
	}
}

// ParseParams 的解析与缺省值必须与上游 SaiAdmin 6.x 一致（saiType 缺省 list、page 1、limit 10）
func TestParseParams(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET",
		"/core/config/index?saiType=all&page=2&limit=5&orderField=sort&orderType=desc", nil)

	p := ParseParams(c)
	if p.SaiType != "all" || p.Page != 2 || p.Limit != 5 ||
		p.OrderField != "sort" || p.OrderType != "desc" {
		t.Fatalf("ParseParams 解析结果不正确: %+v", p)
	}
}

func TestParseParamsDefaults(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/core/config/index", nil)

	p := ParseParams(c)
	if p.SaiType != "list" || p.Page != 1 || p.Limit != 10 || p.OrderType != "" {
		t.Fatalf("ParseParams 缺省值不正确: %+v", p)
	}
	if p.OrderField != "" {
		t.Fatalf("orderField 缺省应为空，实际 %q", p.OrderField)
	}
}
