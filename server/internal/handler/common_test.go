package handler

import (
	"testing"

	"github.com/CodeMainWalker/GoBaseAdmin/internal/model"
)

// 配置项新增（Save 走 setStructFields + assign）必须能绑定数组形态的
// config_select_data —— 前端配置弹窗提交的就是数组。
func TestSetStructFieldsJSONList(t *testing.T) {
	var m model.Config
	setStructFields(&m, map[string]interface{}{
		"key":        "upload_mode",
		"name":       "上传模式",
		"input_type": "select",
		"sort":       float64(99),
		"config_select_data": []interface{}{
			map[string]interface{}{"label": "本地上传", "value": "1"},
		},
	})
	if m.Key != "upload_mode" || m.InputType != "select" || m.Sort != 99 {
		t.Fatalf("普通字段绑定失败: %+v", m)
	}
	if len(m.ConfigSelectData) != 1 || m.ConfigSelectData[0]["label"] != "本地上传" {
		t.Fatalf("config_select_data 数组绑定失败: %#v", m.ConfigSelectData)
	}
}

// 也要兼容数据库里的 JSON 文本形态
func TestSetStructFieldsJSONListText(t *testing.T) {
	var m model.Config
	setStructFields(&m, map[string]interface{}{
		"config_select_data": `[{"label":"七牛云","value":"3"}]`,
	})
	if len(m.ConfigSelectData) != 1 || m.ConfigSelectData[0]["value"] != "3" {
		t.Fatalf("config_select_data 文本绑定失败: %#v", m.ConfigSelectData)
	}
}

// 空数组不能写坏字段
func TestSetStructFieldsJSONListEmpty(t *testing.T) {
	var m model.Config
	setStructFields(&m, map[string]interface{}{
		"config_select_data": []interface{}{},
	})
	if len(m.ConfigSelectData) != 0 {
		t.Fatalf("空数组应得到空列表: %#v", m.ConfigSelectData)
	}
}
