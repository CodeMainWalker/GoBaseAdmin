package handler

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/CodeMainWalker/GoBaseAdmin/internal/menu"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/middleware"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/model"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/query"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/response"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/store"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/svc"
)

// DeptHandler 部门管理
type DeptHandler struct{}

func NewDeptHandler() *DeptHandler { return &DeptHandler{} }

// Index GET /core/dept/index —— 树形结构（含部门领导）
func (h *DeptHandler) Index(c *gin.Context) {
	response.Success(c, deptTree(c, false))
}

// deptTree 构造部门树。
//
// 两种形态：
//   - 完整形态（tree=false）→ 返回 d.*，并附带 leader 关联，供部门管理列表用
//   - 精简形态（tree=true） → 仅 id/value/label/parent_id，供下拉/树选择器用
//
// accessOnly 为 true 时（accessDept 接口）**始终**走精简形态：
// 上游 SaiAdmin 6.x 的 accessDept 接口字段是固定写死的，
// 不看 tree 参数；而前端调用 accessDept() 时不带任何参数，
// 若这里按 tree 判断就会返回 d.*（只有 name、没有 label），
// 导致 el-tree / el-tree-select 的 label 取不到值而显示空白。
func deptTree(c *gin.Context, accessOnly bool) []menu.Row {
	info := middleware.GetAdminInfo(c)
	// 对 accessDept 而言形态是固定的，不受 tree 参数影响
	isTree := c.Query("tree") == "true" || accessOnly

	db := store.DB.Table("sa_system_dept d").Where("d.delete_time IS NULL")
	if name := c.Query("name"); name != "" {
		db = db.Where("d.name LIKE ?", "%"+name+"%")
	}
	if status := c.Query("status"); status != "" {
		db = db.Where("d.status = ?", status)
	}
	// 数据权限
	if accessOnly && info != nil && info.ID != 1 && info.DeptList != nil {
		ids := deptSubtreeIDs(info.DeptList)
		db = db.Where("d.id IN ?", ids)
	}

	var raw []map[string]interface{}
	if isTree {
		db.Select("d.id, d.id AS value, d.name AS label, d.parent_id").Order("d.sort DESC").Scan(&raw)
	} else {
		db.Select("d.*").Order("d.sort DESC").Scan(&raw)
	}

	rows := normalizeRawRows(raw, "create_time", "update_time", "delete_time")

	// 仅完整形态才需要 leader 关联（精简形态前端只读 label/value）
	if !isTree && len(rows) > 0 {
		attachDeptLeaders(rows)
	}
	return menu.MakeTree(rows, "id", "parent_id")
}

// attachDeptLeaders 批量填充 leader 字段，避免 N+1 查询。
// 上游 SaiAdmin 6.x 的部门树接口会在每行附带 leader 关联对象，此处语义相同。
func attachDeptLeaders(rows []menu.Row) {
	// 收集所有 leader_id
	ids := make([]uint64, 0, len(rows))
	seen := map[uint64]bool{}
	for _, r := range rows {
		lid := toUint(r["leader_id"])
		if lid > 0 && !seen[lid] {
			seen[lid] = true
			ids = append(ids, lid)
		}
	}
	if len(ids) == 0 {
		return
	}

	var users []model.User
	store.DB.Where("id IN ?", ids).Find(&users)

	byID := map[uint64]*model.User{}
	for i := range users {
		byID[uint64(users[i].ID)] = &users[i]
	}

	for _, r := range rows {
		lid := toUint(r["leader_id"])
		if u, ok := byID[lid]; ok {
			// 与上游 SaiAdmin 6.x 一致：输出用户对象，但不含 password 等敏感字段
			r["leader"] = gin.H{
				"id":       u.ID,
				"username": u.Username,
				"realname": u.Realname,
				"avatar":   u.Avatar,
				"email":    u.Email,
				"phone":    u.Phone,
				"gender":   u.Gender,
				"status":   u.Status,
				"dept_id":  u.DeptID,
			}
		} else {
			r["leader"] = nil
		}
	}
}

// deptRows 供内部复用
func deptRows(c *gin.Context) []menu.Row { return deptTree(c, false) }

// AccessDept GET /core/dept/accessDept —— 可操作部门（树）
func (h *DeptHandler) AccessDept(c *gin.Context) {
	response.Success(c, deptTree(c, true))
}

// Read GET /core/dept/read?id=
func (h *DeptHandler) Read(c *gin.Context) {
	var d model.Dept
	if err := store.DB.Where("id = ?", getIDParam(c)).First(&d).Error; err != nil {
		response.Fail(c, "数据不存在")
		return
	}
	response.Success(c, d)
}

// Save POST /core/dept/save
func (h *DeptHandler) Save(c *gin.Context) {
	data := bindBody(c)
	data = h.handleLevel(data)
	d := model.Dept{}
	setStructFields(&d, data)
	if err := store.DB.Create(&d).Error; err != nil {
		response.Fail(c, "添加失败")
		return
	}
	response.Success(c, "添加成功")
}

// Update PUT /core/dept/update
func (h *DeptHandler) Update(c *gin.Context) {
	data := bindBody(c)
	id := toUint(data["id"])

	var old model.Dept
	if err := store.DB.Where("id = ?", id).First(&old).Error; err != nil {
		response.Fail(c, "数据不存在")
		return
	}
	if toUint(data["parent_id"]) == id {
		response.Fail(c, "上级部门和当前部门不能相同")
		return
	}

	data = h.handleLevel(data)
	newLevel := toStr(data["level"]) + itoa(int(id)) + ","

	// 不能把上级设为自己的子部门
	oldSelfLevel := old.Level + itoa(int(id)) + ","
	for _, part := range strings.Split(toStr(data["level"]), ",") {
		if part == itoa(int(id)) {
			response.Fail(c, "不能将上级部门设置为当前部门的子部门")
			return
		}
	}

	// 迁移子部门的 level 前缀
	updates := pick(data, "parent_id", "name", "code", "leader_id", "level", "sort", "status", "remark")
	if err := store.DB.Model(&model.Dept{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		response.Fail(c, "修改失败")
		return
	}
	store.DB.Exec(`UPDATE sa_system_dept
		SET level = REPLACE(level, ?, ?)
		WHERE level LIKE ? AND delete_time IS NULL`,
		oldSelfLevel, newLevel, oldSelfLevel+"%")

	clearDeptUsers(int(id))
	response.Success(c, "修改成功")
}

// Destroy DELETE /core/dept/destroy
func (h *DeptHandler) Destroy(c *gin.Context) {
	ids := query.IDsFrom(getIDsParam(c))
	if len(ids) == 0 {
		response.Fail(c, "参数错误，请检查")
		return
	}
	var childCount int64
	store.DB.Model(&model.Dept{}).Where("parent_id IN ? AND delete_time IS NULL", ids).Count(&childCount)
	if childCount > 0 {
		response.Fail(c, "该部门下存在子部门，请先删除子部门")
		return
	}
	var userCount int64
	store.DB.Model(&model.User{}).Where("dept_id IN ? AND delete_time IS NULL", ids).Count(&userCount)
	if userCount > 0 {
		response.Fail(c, "该部门下存在用户，请先删除或者转移用户")
		return
	}
	now := model.Now()
	store.DB.Model(&model.Dept{}).Where("id IN ?", ids).Update("delete_time", now)
	for _, id := range ids {
		clearDeptUsers(int(id))
	}
	response.Success(c, "操作成功")
}

// handleLevel 计算 level 前缀串（父级 level + 父级 id + ','）
func (h *DeptHandler) handleLevel(data map[string]interface{}) map[string]interface{} {
	parentID := toUint(data["parent_id"])
	if parentID == 0 {
		data["level"] = "0"
		data["parent_id"] = 0
		return data
	}
	var parent model.Dept
	if err := store.DB.Where("id = ?", parentID).First(&parent).Error; err == nil {
		data["level"] = parent.Level + itoa(int(parent.ID)) + ","
	}
	return data
}

// clearDeptUsers 清理部门下用户的缓存
func clearDeptUsers(deptID int) {
	var uids []int
	store.DB.Raw(`SELECT id FROM sa_system_user WHERE dept_id = ? AND delete_time IS NULL`, deptID).Scan(&uids)
	for _, uid := range uids {
		svc.ClearUserCache(uid)
	}
}

// ---------- 岗位 ----------

// PostHandler 岗位管理
type PostHandler struct{}

func NewPostHandler() *PostHandler { return &PostHandler{} }

// Index GET /core/post/index
func (h *PostHandler) Index(c *gin.Context) {
	p := query.ParseParams(c)
	db := store.DB.Model(&model.Post{}).Where("delete_time IS NULL")
	if name := c.Query("name"); name != "" {
		db = db.Where("name LIKE ?", "%"+name+"%")
	}
	db = query.ApplySearch(db, map[string]interface{}{
		"code":   c.Query("code"),
		"status": c.Query("status"),
	})
	db = p.Apply(db, "sort")
	var list []model.Post
	response.Success(c, p.Paginate(db, &list))
}

// Read GET /core/post/read?id=
func (h *PostHandler) Read(c *gin.Context) {
	var m model.Post
	if err := store.DB.Where("id = ?", getIDParam(c)).First(&m).Error; err != nil {
		response.Fail(c, "数据不存在")
		return
	}
	response.Success(c, m)
}

// Save POST /core/post/save
func (h *PostHandler) Save(c *gin.Context) {
	data := bindBody(c)
	m := model.Post{}
	setStructFields(&m, data)
	if err := store.DB.Create(&m).Error; err != nil {
		response.Fail(c, "添加失败")
		return
	}
	response.Success(c, "添加成功")
}

// Update PUT /core/post/update
func (h *PostHandler) Update(c *gin.Context) {
	data := bindBody(c)
	updates := pick(data, "name", "code", "sort", "status", "remark")
	if err := store.DB.Model(&model.Post{}).Where("id = ?", toStr(data["id"])).Updates(updates).Error; err != nil {
		response.Fail(c, "修改失败")
		return
	}
	clearPostUsers(toInt(data["id"]))
	response.Success(c, "修改成功")
}

// Destroy DELETE /core/post/destroy
func (h *PostHandler) Destroy(c *gin.Context) {
	ids := query.IDsFrom(getIDsParam(c))
	if len(ids) == 0 {
		response.Fail(c, "参数错误，请检查")
		return
	}
	now := model.Now()
	store.DB.Model(&model.Post{}).Where("id IN ?", ids).Update("delete_time", now)
	store.DB.Where("post_id IN ?", ids).Delete(&model.UserPost{})
	for _, id := range ids {
		clearPostUsers(int(id))
	}
	response.Success(c, "操作成功")
}

// AccessPost GET /core/post/accessPost —— 可分配的岗位列表
func (h *PostHandler) AccessPost(c *gin.Context) {
	var list []model.Post
	store.DB.Where("delete_time IS NULL").Order("sort DESC").Find(&list)
	response.Success(c, list)
}

func clearPostUsers(postID int) {
	var uids []int
	store.DB.Raw(`SELECT user_id FROM sa_system_user_post WHERE post_id = ?`, postID).Scan(&uids)
	for _, uid := range uids {
		svc.ClearUserCache(uid)
	}
}
