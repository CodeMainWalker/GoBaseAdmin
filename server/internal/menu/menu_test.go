package menu

import (
	"encoding/json"
	"testing"
)

// TestMakeArtdMenusTopLevel 验证 type∈{1,2,4} 的树构造与字段映射。
// 关键点：children 键必须稳定存在，且 type=1 的 component 固定为 /index/index。
func TestMakeArtdMenusTopLevel(t *testing.T) {
	rows := []Row{
		{"id": 1, "parent_id": 0, "name": "仪表盘", "code": "Dashboard", "type": 1,
			"path": "/dashboard", "component": nil, "icon": "ri:pie-chart-line",
			"is_iframe": 2, "is_keep_alive": 2, "is_hidden": 2, "is_fixed_tab": 2, "is_full_page": 2},
		{"id": 2, "parent_id": 1, "name": "工作台", "code": "Console", "type": 2,
			"path": "console", "component": "/dashboard/console", "icon": "ri:home-smile-2-line",
			"is_iframe": 2, "is_keep_alive": 2, "is_hidden": 2, "is_fixed_tab": 2, "is_full_page": 2},
		{"id": 19, "parent_id": 0, "name": "官方文档", "code": "Document", "type": 4,
			"path": "", "component": "", "icon": "ri:file-copy-2-fill", "link_url": "https://saithink.top",
			"is_iframe": 2, "is_keep_alive": 2, "is_hidden": 2, "is_fixed_tab": 2, "is_full_page": 2},
	}

	tree := MakeArtdMenus(rows)
	if len(tree) != 2 {
		t.Fatalf("期望 2 个根节点，实际 %d", len(tree))
	}
	if len(tree[0].Children) != 1 {
		t.Fatalf("期望根节点 1 有 1 个子节点，实际 %d", len(tree[0].Children))
	}
	if tree[0].Component != "/index/index" {
		t.Errorf("type=1 的 component 应为 /index/index，实际 %q", tree[0].Component)
	}
	if tree[0].Children[0].Component != "/dashboard/console" {
		t.Errorf("type=2 的 component 应取自 component 列，实际 %q", tree[0].Children[0].Component)
	}

	// 外链：path 必须改写，meta.link 必须带上
	link := tree[1]
	if link.Path != "/outside/Iframe" {
		t.Errorf("type=4 的 path 应为 /outside/Iframe，实际 %q", link.Path)
	}
	if link.Meta.Link != "https://saithink.top" {
		t.Errorf("type=4 的 meta.link 应等于 link_url，实际 %q", link.Meta.Link)
	}

	b, _ := json.Marshal(tree[0])
	t.Logf("root[0] json: %s", string(b))
}

// TestMakeTreeOrder 验证通用树的父子挂接
func TestMakeTreeOrder(t *testing.T) {
	rows := []Row{
		{"id": 1, "parent_id": 0, "name": "a"},
		{"id": 2, "parent_id": 1, "name": "b"},
		{"id": 3, "parent_id": 0, "name": "c"},
	}
	tree := MakeTree(rows, "id", "parent_id")
	if len(tree) != 2 {
		t.Fatalf("期望 2 个根节点，实际 %d", len(tree))
	}
	kids, ok := tree[0]["children"].([]Row)
	if !ok || len(kids) != 1 {
		t.Fatalf("期望根节点 1 有 1 个子节点")
	}
}
