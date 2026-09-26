package model

import (
	"encoding/json"
	"testing"
)

// 库里的 JSON 文本要解码成数组
func TestJSONListScanText(t *testing.T) {
	var l JSONList
	if err := l.Scan(`[{"label":"本地上传","value":"1"}]`); err != nil {
		t.Fatal(err)
	}
	if len(l) != 1 || l[0]["label"] != "本地上传" {
		t.Fatalf("解码结果不正确: %#v", l)
	}
	// 带空白/换行的形态（邮件服务的 radio 数据就是这种）
	var l2 JSONList
	if err := l2.Scan("[\r\n    {\"label\":\"ssl\",\"value\":\"ssl\"}\r\n]"); err != nil {
		t.Fatal(err)
	}
	if len(l2) != 1 || l2[0]["value"] != "ssl" {
		t.Fatalf("带空白的内容解码失败: %#v", l2)
	}
}

// 空值 / 非法内容都退化为空列表，绝不能返回 null
func TestJSONListScanEmptyAndBroken(t *testing.T) {
	for _, v := range []interface{}{nil, "", []byte(""), "not-json", []byte("{}")} {
		var l JSONList
		if err := l.Scan(v); err != nil {
			t.Fatal(err)
		}
		b, err := json.Marshal(l)
		if err != nil {
			t.Fatal(err)
		}
		if string(b) != "[]" {
			t.Fatalf("输入 %#v 序列化为 %s，期望 []", v, string(b))
		}
	}
}

// nil 值直接序列化也要是 []
func TestJSONListMarshalNil(t *testing.T) {
	var l JSONList
	b, err := json.Marshal(l)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "[]" {
		t.Fatalf("nil JSONList 序列化为 %s，期望 []", string(b))
	}
}

// 写库：空列表写空串，非空写 JSON 文本
func TestJSONListValue(t *testing.T) {
	var empty JSONList
	v, err := empty.Value()
	if err != nil {
		t.Fatal(err)
	}
	if s, _ := v.(string); s != "" {
		t.Fatalf("空列表应写空串，实际 %q", s)
	}

	l := JSONList{{"label": "a", "value": "1"}}
	v, err = l.Value()
	if err != nil {
		t.Fatal(err)
	}
	s, _ := v.(string)
	var back JSONList
	if err := json.Unmarshal([]byte(s), &back); err != nil || len(back) != 1 {
		t.Fatalf("写库文本无法往返: %q (%v)", s, err)
	}
}

// 请求体两种形态都要能接受：数组、JSON 文本
func TestJSONListUnmarshalJSON(t *testing.T) {
	var fromArray JSONList
	if err := json.Unmarshal([]byte(`[{"label":"a","value":"1"}]`), &fromArray); err != nil {
		t.Fatal(err)
	}
	if len(fromArray) != 1 {
		t.Fatalf("数组形态解析失败: %#v", fromArray)
	}

	var fromString JSONList
	if err := json.Unmarshal([]byte(`"[{\"label\":\"b\",\"value\":\"2\"}]"`), &fromString); err != nil {
		t.Fatal(err)
	}
	if len(fromString) != 1 || fromString[0]["label"] != "b" {
		t.Fatalf("JSON 文本形态解析失败: %#v", fromString)
	}
}

// map 更新路径用的编码函数
func TestEncodeJSONList(t *testing.T) {
	got := EncodeJSONList([]interface{}{
		map[string]interface{}{"label": "a", "value": "1"},
	})
	if got != `[{"label":"a","value":"1"}]` {
		t.Fatalf("数组编码结果 %q 不正确", got)
	}
	if got := EncodeJSONList(`[{"label":"x","value":"9"}]`); got != `[{"label":"x","value":"9"}]` {
		t.Fatalf("JSON 文本应原样保留，实际 %q", got)
	}
	if got := EncodeJSONList(nil); got != "" {
		t.Fatalf("nil 应编码为空串，实际 %q", got)
	}
	if got := EncodeJSONList([]interface{}{}); got != "" {
		t.Fatalf("空数组应编码为空串，实际 %q", got)
	}
}
