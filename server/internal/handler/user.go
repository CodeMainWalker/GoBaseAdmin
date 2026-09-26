package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/CodeMainWalker/GoBaseAdmin/internal/auth"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/middleware"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/model"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/query"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/response"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/store"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/svc"
)

// UserHandler 用户管理
type UserHandler struct{}

func NewUserHandler() *UserHandler { return &UserHandler{} }

// Index GET /core/user/index
func (h *UserHandler) Index(c *gin.Context) {
	info := middleware.GetAdminInfo(c)
	p := query.ParseParams(c)

	db := store.DB.Model(&model.User{}).Where("delete_time IS NULL")

	if kw := c.Query("keyword"); kw != "" {
		db = query.LikeAny(db, []string{"username", "realname", "phone"}, kw)
	}
	db = query.ApplySearch(db, map[string]interface{}{
		"username": c.Query("username"),
		"phone":    c.Query("phone"),
		"email":    c.Query("email"),
		"status":   c.Query("status"),
		"dept_id":  c.Query("dept_id"),
	})

	// 数据权限：非超管仅可见自己部门及子部门的用户
	if info != nil && info.ID != 1 && info.DeptList != nil {
		ids := deptSubtreeIDs(info.DeptList)
		db = db.Where("dept_id IN ?", ids)
	}

	db = p.Apply(db, "id")
	var list []model.User
	// 需要逐行附加部门信息，故固定返回分页对象（该接口不会被前端以 saiType=all 调用）
	page := p.PageResult(db, &list)
	page.Data = usersWithDept(list)
	response.Success(c, page)
}

// usersWithDept 附加部门信息（对应 with(['depts'])）
func usersWithDept(list []model.User) []gin.H {
	out := make([]gin.H, 0, len(list))
	deptCache := map[uint64]*model.Dept{}
	for _, u := range list {
		var dept interface{}
		if u.DeptID != nil && *u.DeptID > 0 {
			d, ok := deptCache[*u.DeptID]
			if !ok {
				var dd model.Dept
				if err := store.DB.Where("id = ?", *u.DeptID).First(&dd).Error; err == nil {
					d = &dd
					deptCache[*u.DeptID] = d
				}
			}
			if d != nil {
				dept = d
			}
		}
		item := gin.H{}
		item["id"] = u.ID
		item["username"] = u.Username
		item["realname"] = u.Realname
		item["gender"] = u.Gender
		item["avatar"] = u.Avatar
		item["email"] = u.Email
		item["phone"] = u.Phone
		item["signed"] = u.Signed
		item["dashboard"] = u.Dashboard
		item["dept_id"] = u.DeptID
		item["is_super"] = u.IsSuper
		item["status"] = u.Status
		item["remark"] = u.Remark
		item["login_time"] = u.LoginTime
		item["login_ip"] = u.LoginIP
		item["create_time"] = u.CreateTime
		item["update_time"] = u.UpdateTime
		item["depts"] = dept
		out = append(out, item)
	}
	return out
}

// Read GET /core/user/read?id=
func (h *UserHandler) Read(c *gin.Context) {
	id := getIDParam(c)
	data, err := loadUserDetail(id)
	if err != nil {
		response.Fail(c, "未查找到信息")
		return
	}
	info := middleware.GetAdminInfo(c)
	if info != nil && info.ID != 1 && info.DeptList != nil {
		if !deptProtect(info.DeptList, toUint(data["dept_id"])) {
			response.Fail(c, "没有权限操作该部门数据")
			return
		}
	}
	response.Success(c, data)
}

// loadUserDetail 读取用户详情（含角色/岗位/部门列表）
func loadUserDetail(id string) (gin.H, error) {
	var u model.User
	if err := store.DB.Where("id = ?", id).First(&u).Error; err != nil {
		return nil, err
	}

	var roles []model.Role
	store.DB.Raw(`SELECT r.* FROM sa_system_role r
		JOIN sa_system_user_role ur ON ur.role_id = r.id
		WHERE ur.user_id = ? AND r.delete_time IS NULL`, u.ID).Scan(&roles)

	var posts []model.Post
	store.DB.Raw(`SELECT p.* FROM sa_system_post p
		JOIN sa_system_user_post up ON up.post_id = p.id
		WHERE up.user_id = ? AND p.delete_time IS NULL`, u.ID).Scan(&posts)

	var depts []model.Dept
	if u.DeptID != nil && *u.DeptID > 0 {
		var d model.Dept
		if err := store.DB.Where("id = ?", *u.DeptID).First(&d).Error; err == nil {
			depts = append(depts, d)
		}
	}

	return gin.H{
		"id": u.ID, "username": u.Username, "realname": u.Realname,
		"gender": u.Gender, "avatar": u.Avatar, "email": u.Email,
		"phone": u.Phone, "signed": u.Signed, "dashboard": u.Dashboard,
		"dept_id": u.DeptID, "is_super": u.IsSuper, "status": u.Status,
		"remark": u.Remark, "login_time": u.LoginTime, "login_ip": u.LoginIP,
		"create_time": u.CreateTime, "update_time": u.UpdateTime,
		"roleList": roles, "postList": posts, "deptList": depts,
	}, nil
}

// Save POST /core/user/save
func (h *UserHandler) Save(c *gin.Context) {
	data := bindBody(c)
	info := middleware.GetAdminInfo(c)

	if toStr(data["username"]) == "" {
		response.Fail(c, "账号不能为空")
		return
	}
	if toStr(data["password"]) == "" {
		response.Fail(c, "密码不能为空")
		return
	}
	var cnt int64
	store.DB.Model(&model.User{}).Where("username = ? AND delete_time IS NULL", toStr(data["username"])).Count(&cnt)
	if cnt > 0 {
		response.Fail(c, "账号已存在")
		return
	}
	// 唯一索引 uk_username 不区分软删除：先物理清除同名的已删除残留行，
	// 否则插入会因唯一键冲突失败（且删掉会导致孤立关联数据）。
	store.DB.Exec("DELETE FROM sa_system_user WHERE username = ? AND delete_time IS NOT NULL", toStr(data["username"]))

	roleIDs := query.IDsFrom(data["role_ids"])
	postIDs := query.IDsFrom(data["post_ids"])

	if info != nil && info.ID != 1 && info.DeptList != nil {
		if !deptProtect(info.DeptList, toUint(data["dept_id"])) {
			response.Fail(c, "没有权限操作该部门数据")
			return
		}
		if !roleProtect(info.RoleList, roleIDs) {
			response.Fail(c, "没有权限操作该角色数据")
			return
		}
	}

	hashed, err := auth.HashPassword(toStr(data["password"]))
	if err != nil {
		response.Fail(c, "密码加密失败")
		return
	}

	u := model.User{}
	setStructFields(&u, data)
	u.Password = hashed
	if v, ok := data["dept_id"]; ok {
		d := toUint(v)
		u.DeptID = &d
	}

	if err := store.DB.Create(&u).Error; err != nil {
		response.Fail(c, "添加失败")
		return
	}

	replaceUserRoles(uint64(u.ID), roleIDs)
	replaceUserPosts(uint64(u.ID), postIDs)
	response.Success(c, "添加成功")
}

// Update PUT /core/user/update
func (h *UserHandler) Update(c *gin.Context) {
	data := bindBody(c)
	info := middleware.GetAdminInfo(c)

	id := toStr(data["id"])
	if id == "" {
		response.Fail(c, "参数错误")
		return
	}
	var u model.User
	if err := store.DB.Where("id = ?", id).First(&u).Error; err != nil {
		response.Fail(c, "数据不存在")
		return
	}

	if info != nil && info.ID != 1 && info.DeptList != nil {
		if !deptSubtreeContains(info.DeptList, u.DeptID) {
			response.Fail(c, "没有权限操作该数据")
			return
		}
		if !deptProtect(info.DeptList, toUint(data["dept_id"])) {
			response.Fail(c, "没有权限操作该部门数据")
			return
		}
	}

	roleIDs := query.IDsFrom(data["role_ids"])
	postIDs := query.IDsFrom(data["post_ids"])
	if info != nil && info.ID != 1 {
		if !roleProtect(info.RoleList, roleIDs) {
			response.Fail(c, "没有权限操作该角色数据")
			return
		}
	}

	delete(data, "password")
	updates := pick(data,
		"username", "realname", "gender", "avatar", "email", "phone",
		"signed", "dashboard", "dept_id", "status", "remark")
	if v, ok := updates["dept_id"]; ok {
		updates["dept_id"] = toUint(v)
	}
	if err := store.DB.Model(&model.User{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		response.Fail(c, "修改失败")
		return
	}

	replaceUserRoles(uint64(u.ID), roleIDs)
	replaceUserPosts(uint64(u.ID), postIDs)
	svc.ClearUserCache(int(u.ID))
	response.Success(c, "修改成功")
}

// Destroy DELETE /core/user/destroy
func (h *UserHandler) Destroy(c *gin.Context) {
	ids := query.IDsFrom(getIDsParam(c))
	info := middleware.GetAdminInfo(c)

	if len(ids) == 0 {
		response.Fail(c, "参数错误，请检查")
		return
	}
	if len(ids) > 1 {
		response.Fail(c, "禁止批量删除操作")
		return
	}
	id := ids[0]
	if id == 1 {
		response.Fail(c, "超级管理员禁止删除")
		return
	}

	var u model.User
	if err := store.DB.Where("id = ?", id).First(&u).Error; err != nil {
		response.Fail(c, "没有权限操作该数据")
		return
	}
	if info != nil && info.ID != 1 && info.DeptList != nil {
		if !deptSubtreeContains(info.DeptList, u.DeptID) {
			response.Fail(c, "没有权限操作该数据")
			return
		}
	}

	svc.ClearUserCache(int(id))
	now := model.Now()
	store.DB.Model(&model.User{}).Where("id = ?", id).Update("delete_time", now)
	store.DB.Where("user_id = ?", id).Delete(&model.UserRole{})
	store.DB.Where("user_id = ?", id).Delete(&model.UserPost{})
	response.Success(c, "操作成功")
}

// ClearCache POST /core/user/clearCache
func (h *UserHandler) ClearCache(c *gin.Context) {
	id := toInt(getIDParam(c))
	if id == 0 {
		id = toInt(bindBody(c)["id"])
	}
	svc.ClearUserCache(id)
	response.Success(c, "操作成功")
}

// InitUserPassword POST /core/user/initUserPassword
func (h *UserHandler) InitUserPassword(c *gin.Context) {
	data := bindBody(c)
	id := toInt(data["id"])
	password := toStr(data["password"])
	if id == 1 {
		response.Fail(c, "超级管理员不允许重置密码")
		return
	}
	hashed, err := auth.HashPassword(password)
	if err != nil {
		response.Fail(c, "密码加密失败")
		return
	}
	store.DB.Model(&model.User{}).Where("id = ?", id).Update("password", hashed)
	svc.ClearUserCache(id)
	response.Success(c, "操作成功")
}

// SetHomePage POST /core/user/setHomePage
func (h *UserHandler) SetHomePage(c *gin.Context) {
	data := bindBody(c)
	id := toInt(data["id"])
	dashboard := toStr(data["dashboard"])
	store.DB.Model(&model.User{}).Where("id = ?", id).Update("dashboard", dashboard)
	svc.ClearUserCache(id)
	response.Success(c, "操作成功")
}

// UpdateInfo POST /core/user/updateInfo —— 修改个人资料
func (h *UserHandler) UpdateInfo(c *gin.Context) {
	data := bindBody(c)
	uid := middleware.GetAdminID(c)
	updates := pick(data, "realname", "gender", "phone", "email", "avatar", "signed")
	store.DB.Model(&model.User{}).Where("id = ?", uid).Updates(updates)
	svc.ClearUserCache(uid)
	response.Success(c, "操作成功")
}

// ModifyPassword POST /core/user/modifyPassword
func (h *UserHandler) ModifyPassword(c *gin.Context) {
	data := bindBody(c)
	uid := middleware.GetAdminID(c)
	oldPw := toStr(data["oldPassword"])
	newPw := toStr(data["newPassword"])

	var u model.User
	if err := store.DB.Where("id = ?", uid).First(&u).Error; err != nil {
		response.Fail(c, "用户不存在")
		return
	}
	if !auth.CheckPassword(u.Password, oldPw) {
		response.Fail(c, "原密码错误")
		return
	}
	hashed, err := auth.HashPassword(newPw)
	if err != nil {
		response.Fail(c, "密码加密失败")
		return
	}
	store.DB.Model(&model.User{}).Where("id = ?", uid).Update("password", hashed)
	svc.ClearUserCache(uid)
	response.Success(c, "修改成功")
}

// ---------- 关系维护 ----------

func replaceUserRoles(uid uint64, roleIDs []uint64) {
	store.DB.Where("user_id = ?", uid).Delete(&model.UserRole{})
	for _, rid := range roleIDs {
		if rid == 0 {
			continue
		}
		store.DB.Create(&model.UserRole{UserID: uid, RoleID: rid})
	}
}

func replaceUserPosts(uid uint64, postIDs []uint64) {
	store.DB.Where("user_id = ?", uid).Delete(&model.UserPost{})
	for _, pid := range postIDs {
		if pid == 0 {
			continue
		}
		store.DB.Create(&model.UserPost{UserID: uid, PostID: pid})
	}
}

// ---------- 数据权限保护 ----------

// deptSubtreeIDs 返回部门自身及全部子部门 ID
func deptSubtreeIDs(d *model.Dept) []uint64 {
	ids := []uint64{uint64(d.ID)}
	prefix := d.Level + itoa(int(d.ID)) + ","
	var sub []uint64
	store.DB.Raw(`SELECT id FROM sa_system_dept
		WHERE level LIKE ? AND delete_time IS NULL`, prefix+"%").Scan(&sub)
	return append(ids, sub...)
}

func deptSubtreeContains(d *model.Dept, deptID *uint64) bool {
	if deptID == nil {
		return false
	}
	for _, id := range deptSubtreeIDs(d) {
		if id == *deptID {
			return true
		}
	}
	return false
}

// deptProtect 判断目标部门是否在当前用户可操作范围内
func deptProtect(d *model.Dept, deptID uint64) bool {
	if deptID == 0 {
		return true
	}
	for _, id := range deptSubtreeIDs(d) {
		if id == deptID {
			return true
		}
	}
	return false
}

// roleProtect 越权保护：目标角色级别必须低于自身最高级别
func roleProtect(roleList []model.Role, roleIDs []uint64) bool {
	if len(roleList) == 0 {
		return true
	}
	maxLevel := 0
	for _, r := range roleList {
		if r.Level > maxLevel {
			maxLevel = r.Level
		}
	}
	if len(roleIDs) == 0 {
		return true
	}
	var currentMax *int
	store.DB.Raw(`SELECT MAX(level) FROM sa_system_role WHERE id IN ?`, roleIDs).Scan(&currentMax)
	if currentMax == nil {
		return true
	}
	return *currentMax < maxLevel
}
