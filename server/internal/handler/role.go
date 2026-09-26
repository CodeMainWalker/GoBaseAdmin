package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/CodeMainWalker/GoBaseAdmin/internal/middleware"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/model"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/query"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/response"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/store"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/svc"
)

// RoleHandler 角色管理
type RoleHandler struct{}

func NewRoleHandler() *RoleHandler { return &RoleHandler{} }

// Index GET /core/role/index
func (h *RoleHandler) Index(c *gin.Context) {
	info := middleware.GetAdminInfo(c)
	p := query.ParseParams(c)
	db := store.DB.Model(&model.Role{}).Where("delete_time IS NULL")
	db = query.ApplySearch(db, map[string]interface{}{
		"name":   c.Query("name"),
		"code":   c.Query("code"),
		"status": c.Query("status"),
	})
	// 非超管仅可见可操作（级别低于自身）的角色
	if info != nil && info.ID != 1 {
		db = db.Where("level < ?", maxRoleLevel(info.RoleList))
	}
	db = p.Apply(db, "sort")
	var list []model.Role
	response.Success(c, p.Paginate(db, &list))
}

// Read GET /core/role/read?id=
func (h *RoleHandler) Read(c *gin.Context) {
	var r model.Role
	if err := store.DB.Where("id = ?", getIDParam(c)).First(&r).Error; err != nil {
		response.Fail(c, "数据不存在")
		return
	}
	response.Success(c, r)
}

// Save POST /core/role/save
func (h *RoleHandler) Save(c *gin.Context) {
	data := bindBody(c)
	info := middleware.GetAdminInfo(c)

	if info != nil && info.ID != 1 {
		if toInt(data["level"]) >= maxRoleLevel(info.RoleList) {
			response.Fail(c, "不能操作比当前账户职级高的角色")
			return
		}
	}
	var cnt int64
	store.DB.Model(&model.Role{}).Where("code = ? AND delete_time IS NULL", toStr(data["code"])).Count(&cnt)
	if cnt > 0 {
		response.Fail(c, "角色标识已存在")
		return
	}
	// 唯一索引 uk_slug 不区分软删除：若存在同 code 的已删除记录，
	// 直接插入会因唯一键冲突而失败。这里先物理清除该残留行。
	store.DB.Exec("DELETE FROM sa_system_role WHERE code = ? AND delete_time IS NOT NULL", toStr(data["code"]))

	r := model.Role{}
	setStructFields(&r, data)
	if err := store.DB.Create(&r).Error; err != nil {
		response.Fail(c, "添加失败："+err.Error())
		return
	}
	response.Success(c, "添加成功")
}

// Update PUT /core/role/update
func (h *RoleHandler) Update(c *gin.Context) {
	data := bindBody(c)
	info := middleware.GetAdminInfo(c)
	id := toStr(data["id"])

	var r model.Role
	if err := store.DB.Where("id = ?", id).First(&r).Error; err != nil {
		response.Fail(c, "数据不存在")
		return
	}
	if info != nil && info.ID != 1 {
		if toInt(data["level"]) >= maxRoleLevel(info.RoleList) {
			response.Fail(c, "不能操作比当前账户职级高的角色")
			return
		}
	}
	updates := pick(data, "name", "code", "level", "data_scope", "remark", "sort", "status")
	if err := store.DB.Model(&model.Role{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		response.Fail(c, "修改失败")
		return
	}
	// 角色变更需刷新相关缓存
	roleID := int(r.ID)
	svc.ClearAuthByRole([]int{roleID})
	svc.ClearMenuCache()
	clearUsersOfRole(roleID)
	response.Success(c, "修改成功")
}

// Destroy DELETE /core/role/destroy
func (h *RoleHandler) Destroy(c *gin.Context) {
	ids := query.IDsFrom(getIDsParam(c))
	info := middleware.GetAdminInfo(c)
	if len(ids) == 0 {
		response.Fail(c, "参数错误，请检查")
		return
	}
	if info != nil && info.ID != 1 {
		maxLevel := maxRoleLevel(info.RoleList)
		var cnt int64
		store.DB.Model(&model.Role{}).Where("level >= ? AND id IN ?", maxLevel, ids).Count(&cnt)
		if cnt > 0 {
			response.Fail(c, "不能操作比当前账户职级高的角色")
			return
		}
	}
	now := model.Now()
	store.DB.Model(&model.Role{}).Where("id IN ?", ids).Update("delete_time", now)
	intIDs := make([]int, 0, len(ids))
	for _, id := range ids {
		intIDs = append(intIDs, int(id))
		clearUsersOfRole(int(id))
	}
	svc.ClearAuthByRole(intIDs)
	svc.ClearMenuCache()
	response.Success(c, "操作成功")
}

// AccessRole GET /core/role/accessRole —— 可分配的角色（级别低于自身）
func (h *RoleHandler) AccessRole(c *gin.Context) {
	info := middleware.GetAdminInfo(c)
	db := store.DB.Model(&model.Role{}).Where("delete_time IS NULL")
	if info != nil && info.ID != 1 {
		db = db.Where("level < ?", maxRoleLevel(info.RoleList))
	}
	var list []model.Role
	db.Order("sort DESC").Find(&list)
	response.Success(c, list)
}

// GetMenuByRole GET /core/role/getMenuByRole?id= —— 角色已分配的菜单
func (h *RoleHandler) GetMenuByRole(c *gin.Context) {
	id := getIDParam(c)
	var menus []model.Menu
	store.DB.Raw(`SELECT m.* FROM sa_system_menu m
		JOIN sa_system_role_menu rm ON rm.menu_id = m.id
		WHERE rm.role_id = ? AND m.delete_time IS NULL`, id).Scan(&menus)
	response.Success(c, gin.H{"id": toInt(id), "menus": menus})
}

// MenuPermission POST /core/role/menuPermission —— 保存角色菜单权限
//
// 注意：上游 SaiAdmin 6.x 写菜单关联时带了 100 条上限，菜单超过 100 条会静默截断，
// 导致权限保存不全。Go 版**不复制该缺陷**，全量写入。
func (h *RoleHandler) MenuPermission(c *gin.Context) {
	data := bindBody(c)
	roleID := toUint(data["id"])
	menuIDs := query.IDsFrom(data["menu_ids"])

	store.DB.Where("role_id = ?", roleID).Delete(&model.RoleMenu{})
	if len(menuIDs) > 0 {
		rows := make([]model.RoleMenu, 0, len(menuIDs))
		for _, mid := range menuIDs {
			if mid == 0 {
				continue
			}
			rows = append(rows, model.RoleMenu{RoleID: roleID, MenuID: mid})
		}
		if len(rows) > 0 {
			store.DB.Create(&rows)
		}
	}

	svc.ClearAuthByRole([]int{int(roleID)})
	svc.ClearMenuCache()
	clearUsersOfRole(int(roleID))
	response.Success(c, "操作成功")
}

// clearUsersOfRole 清理拥有该角色的用户缓存
func clearUsersOfRole(roleID int) {
	var uids []int
	store.DB.Raw(`SELECT user_id FROM sa_system_user_role WHERE role_id = ?`, roleID).Scan(&uids)
	for _, uid := range uids {
		svc.ClearUserCache(uid)
	}
}

// maxRoleLevel 当前用户角色的最高级别
func maxRoleLevel(roles []model.Role) int {
	max := 0
	for _, r := range roles {
		if r.Level > max {
			max = r.Level
		}
	}
	return max
}
