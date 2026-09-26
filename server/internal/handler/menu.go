package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/CodeMainWalker/GoBaseAdmin/internal/menu"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/middleware"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/model"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/query"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/response"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/store"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/svc"
)

// MenuHandler 菜单管理
type MenuHandler struct{}

func NewMenuHandler() *MenuHandler { return &MenuHandler{} }

// Index GET /core/menu/index —— 菜单列表/树
func (h *MenuHandler) Index(c *gin.Context) {
	response.Success(c, h.menuRows(c))
}

// menuRows 查询菜单，tree=true 时返回 id/value/label/parent_id/type 精简树
func (h *MenuHandler) menuRows(c *gin.Context) []menu.Row {
	isTree := c.Query("tree") == "true"

	db := store.DB.Table("sa_system_menu m").Where("m.delete_time IS NULL")
	if name := c.Query("name"); name != "" {
		db = db.Where("m.name LIKE ?", "%"+name+"%")
	}
	if status := c.Query("status"); status != "" {
		db = db.Where("m.status = ?", status)
	}
	if t := c.Query("type"); t != "" {
		db = db.Where("m.type = ?", t)
	}
	if pid := c.Query("parent_id"); pid != "" {
		db = db.Where("m.parent_id = ?", pid)
	}

	var raw []map[string]interface{}
	if isTree {
		db.Select("m.id, m.id AS value, m.name AS label, m.parent_id, m.type").
			Order("m.sort DESC").Scan(&raw)
	} else {
		db.Select("m.*").Order("m.sort DESC").Scan(&raw)
	}

	rows := normalizeRawRows(raw, "create_time", "update_time", "delete_time")
	return menu.MakeTree(rows, "id", "parent_id")
}

// AccessMenu GET /core/menu/accessMenu —— 当前用户可分配的菜单（树，供角色授权用）
func (h *MenuHandler) AccessMenu(c *gin.Context) {
	info := middleware.GetAdminInfo(c)
	if info == nil {
		response.Success(c, []interface{}{})
		return
	}

	var raw []map[string]interface{}
	if info.ID == 1 {
		store.DB.Table("sa_system_menu m").Where("m.delete_time IS NULL AND m.status = 1").
			Select("m.id, m.id AS value, m.name AS label, m.parent_id, m.type").
			Order("m.sort DESC").Scan(&raw)
	} else {
		roleIDs := svc.GetRoleIDs(info.ID)
		if len(roleIDs) == 0 {
			response.Success(c, []interface{}{})
			return
		}
		store.DB.Table("sa_system_menu m").
			Where("m.delete_time IS NULL AND m.status = 1 AND m.id IN (SELECT menu_id FROM sa_system_role_menu WHERE role_id IN ?)", roleIDs).
			Select("m.id, m.id AS value, m.name AS label, m.parent_id, m.type").
			Order("m.sort DESC").Scan(&raw)
	}

	rows := normalizeRawRows(raw, "create_time", "update_time", "delete_time")
	response.Success(c, menu.MakeTree(rows, "id", "parent_id"))
}

// Read GET /core/menu/read?id=
func (h *MenuHandler) Read(c *gin.Context) {
	var m model.Menu
	if err := store.DB.Where("id = ?", getIDParam(c)).First(&m).Error; err != nil {
		response.Fail(c, "数据不存在")
		return
	}
	response.Success(c, m)
}

// Save POST /core/menu/save
func (h *MenuHandler) Save(c *gin.Context) {
	data := bindBody(c)
	m := model.Menu{}
	setStructFields(&m, data)
	if err := store.DB.Create(&m).Error; err != nil {
		response.Fail(c, "添加失败")
		return
	}
	svc.ClearMenuCache()
	response.Success(c, "添加成功")
}

// Update PUT /core/menu/update
func (h *MenuHandler) Update(c *gin.Context) {
	data := bindBody(c)
	id := toUint(data["id"])
	if toUint(data["parent_id"]) == id {
		response.Fail(c, "不能设置父级为自身")
		return
	}
	updates := pick(data,
		"parent_id", "name", "code", "slug", "type", "path", "component",
		"method", "icon", "sort", "link_url", "is_iframe", "is_keep_alive",
		"is_hidden", "is_fixed_tab", "is_full_page", "status", "remark")
	if err := store.DB.Model(&model.Menu{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		response.Fail(c, "修改失败")
		return
	}
	// 菜单结构变化会影响菜单树与权限集合
	svc.ClearMenuCache()
	svc.ClearAuthByRole(allRoleIDs())
	response.Success(c, "修改成功")
}

// Destroy DELETE /core/menu/destroy
func (h *MenuHandler) Destroy(c *gin.Context) {
	ids := query.IDsFrom(getIDsParam(c))
	if len(ids) == 0 {
		response.Fail(c, "参数错误，请检查")
		return
	}
	var childCount int64
	store.DB.Model(&model.Menu{}).Where("parent_id IN ? AND delete_time IS NULL", ids).Count(&childCount)
	if childCount > 0 {
		response.Fail(c, "该菜单下存在子菜单，请先删除子菜单")
		return
	}
	now := model.Now()
	store.DB.Model(&model.Menu{}).Where("id IN ?", ids).Update("delete_time", now)
	store.DB.Where("menu_id IN ?", ids).Delete(&model.RoleMenu{})
	svc.ClearMenuCache()
	svc.ClearAuthByRole(allRoleIDs())
	response.Success(c, "操作成功")
}

func allRoleIDs() []int {
	var ids []int
	store.DB.Raw(`SELECT id FROM sa_system_role WHERE delete_time IS NULL`).Scan(&ids)
	return ids
}
