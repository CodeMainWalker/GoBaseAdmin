// Package menu 负责菜单树与前端路由的构造，行为与上游 SaiAdmin 6.x 一致。
// 这是前端路由的唯一契约，字段名与嵌套结构必须逐字保持一致。
package menu

// Row 通用数据行（键为列名，值为原始列值）
type Row map[string]interface{}

// asInt 将数据库取回的任意数值形态归一为 int
func asInt(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case int8:
		return int(n)
	case int16:
		return int(n)
	case int32:
		return int(n)
	case int64:
		return int(n)
	case uint:
		return int(n)
	case uint8:
		return int(n)
	case uint16:
		return int(n)
	case uint32:
		return int(n)
	case uint64:
		return int(n)
	case float64:
		return int(n)
	case float32:
		return int(n)
	case []byte:
		return atoiSafe(string(n))
	case string:
		return atoiSafe(n)
	}
	return 0
}

func atoiSafe(s string) int {
	n := 0
	neg := false
	for i, ch := range s {
		if i == 0 && ch == '-' {
			neg = true
			continue
		}
		if ch < '0' || ch > '9' {
			return n
		}
		n = n*10 + int(ch-'0')
	}
	if neg {
		return -n
	}
	return n
}

func asString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	case []byte:
		return string(s)
	}
	return ""
}

func keyOf(v interface{}) string {
	switch n := v.(type) {
	case string:
		return n
	case []byte:
		return string(n)
	}
	return itoa(asInt(v))
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// MakeTree 构造通用父子树（与上游 SaiAdmin 6.x 的 makeTree 一致）。
// 输出顺序与上游一致：先按输入顺序建索引，再按索引顺序挂接，
// 因此根节点之间的相对顺序取决于传入顺序（上游的关联数组同样保持插入序）。
func MakeTree(rows []Row, keyName, pidName string) []Row {
	list := make(map[string]Row, len(rows))
	order := make([]string, 0, len(rows))
	for _, r := range rows {
		k := keyOf(r[keyName])
		list[k] = r
		order = append(order, k)
	}

	// children 累积
	children := make(map[string][]Row)
	tree := make([]Row, 0)
	for _, k := range order {
		item := list[k]
		pid := keyOf(item[pidName])
		parent, ok := list[pid]
		if ok && pid != k {
			_ = parent
			children[pid] = append(children[pid], item)
		} else {
			tree = append(tree, item)
		}
	}
	attachChildren(tree, children)
	return tree
}

func attachChildren(nodes []Row, children map[string][]Row) {
	for _, n := range nodes {
		k := keyOf(n["id"])
		if cs, ok := children[k]; ok && len(cs) > 0 {
			attachChildren(cs, children)
			n["children"] = cs
		}
	}
}

// ArtdMenuNode 前端路由节点。使用有序结构体保证 JSON 键顺序稳定，
// 且 children 仅在存在子节点时输出（与上游 SaiAdmin 6.x 的引用挂接行为一致）。
type ArtdMenuNode struct {
	ID        interface{}     `json:"id"`
	ParentID  interface{}     `json:"parent_id"`
	Name      string          `json:"name"`
	Path      string          `json:"path"`
	Component string          `json:"component"`
	Meta      ArtdMenuMeta    `json:"meta"`
	Children  []*ArtdMenuNode `json:"children,omitempty"`
}

// ArtdMenuMeta 前端 meta 结构
type ArtdMenuMeta struct {
	Title      string `json:"title"`
	Icon       string `json:"icon"`
	IsIframe   bool   `json:"isIframe"`
	KeepAlive  bool   `json:"keepAlive"`
	IsHide     bool   `json:"isHide"`
	FixedTab   bool   `json:"fixedTab"`
	IsFullPage bool   `json:"isFullPage"`
	Link       string `json:"link,omitempty"`
}

// MakeArtdMenus 生成前端路由树（与上游 SaiAdmin 6.x 的 makeArtdMenus 一致）。
//
// 逐条对齐上游逻辑：
//   - type=1(目录) → component = '/index/index'
//   - type=2(菜单) → component = 该行的 component 字段
//   - type=4(外链) → path 强制为 '/outside/Iframe'，meta.link = link_url
//   - name  取 code 列
//   - path  取 path 列
//   - meta 各布尔位由对应的 tinyint == 1 决定
//
// 仅 type ∈ {1,2,4} 参与构建（调用方已过滤，这里再兜底一次）。
func MakeArtdMenus(rows []Row) []*ArtdMenuNode {
	list := make(map[string]*ArtdMenuNode, len(rows))
	order := make([]string, 0, len(rows))

	for _, r := range rows {
		t := asInt(r["type"])
		if t != 1 && t != 2 && t != 4 {
			continue
		}
		component := ""
		if t == 1 {
			component = "/index/index"
		} else if t == 2 {
			component = asString(r["component"])
		}

		node := &ArtdMenuNode{
			ID:        normalID(r["id"]),
			ParentID:  normalID(r["parent_id"]),
			Name:      asString(r["code"]),
			Path:      asString(r["path"]),
			Component: component,
			Meta: ArtdMenuMeta{
				Title:      asString(r["name"]),
				Icon:       asString(r["icon"]),
				IsIframe:   asInt(r["is_iframe"]) == 1,
				KeepAlive:  asInt(r["is_keep_alive"]) == 1,
				IsHide:     asInt(r["is_hidden"]) == 1,
				FixedTab:   asInt(r["is_fixed_tab"]) == 1,
				IsFullPage: asInt(r["is_full_page"]) == 1,
			},
		}
		if t == 4 {
			node.Path = "/outside/Iframe"
			node.Meta.Link = asString(r["link_url"])
		}

		k := keyOf(r["id"])
		list[k] = node
		order = append(order, k)
	}

	tree := make([]*ArtdMenuNode, 0)
	for _, k := range order {
		node := list[k]
		pid := keyOf(node.ParentID)
		if p, ok := list[pid]; ok && pid != k {
			p.Children = append(p.Children, node)
		} else {
			tree = append(tree, node)
		}
	}
	return tree
}

// normalID 保持 ID 的数值形态，避免 int/uint64 混用导致前端类型判断差异
func normalID(v interface{}) interface{} {
	switch n := v.(type) {
	case nil:
		return 0
	case string:
		return atoiSafe(n)
	case []byte:
		return atoiSafe(string(n))
	}
	return asInt(v)
}
