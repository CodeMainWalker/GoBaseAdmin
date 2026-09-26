package handler

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CodeMainWalker/GoBaseAdmin/internal/cache"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/menu"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/middleware"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/model"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/query"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/response"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/store"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/svc"
)

// SystemHandler 系统通用控制器
type SystemHandler struct{}

func NewSystemHandler() *SystemHandler { return &SystemHandler{} }

// UserInfo GET /core/system/user
// id==1 → buttons=['*'], roles=['super_admin']；否则取真实权限集合
func (h *SystemHandler) UserInfo(c *gin.Context) {
	info := middleware.GetAdminInfo(c)
	if info == nil {
		response.Fail(c, "用户信息读取失败，无法访问或操作")
		return
	}

	out := gin.H{
		"id":         info.ID,
		"username":   info.Username,
		"dashboard":  info.Dashboard,
		"avatar":     info.Avatar,
		"email":      info.Email,
		"phone":      info.Phone,
		"gender":     info.Gender,
		"signed":     info.Signed,
		"realname":   info.Realname,
		"department": info.DeptList,
	}

	if info.ID == 1 {
		out["buttons"] = []string{"*"}
		out["roles"] = []string{"super_admin"}
	} else {
		out["buttons"] = svc.GetUserAuth(info.ID)
		roleCodes := []string{}
		for _, r := range info.RoleList {
			roleCodes = append(roleCodes, r.Code)
		}
		out["roles"] = roleCodes
	}
	response.Success(c, out)
}

// Menu GET /core/system/menu —— 前端动态路由数据源
func (h *SystemHandler) Menu(c *gin.Context) {
	uid := middleware.GetAdminID(c)
	response.Success(c, getUserMenu(uid))
}

// getUserMenu 获取用户菜单树（带缓存）
func getUserMenu(uid int) interface{} {
	if uid <= 0 {
		return []interface{}{}
	}
	key := svc.MenuCachePrefix + itoa(uid)
	if v := cache.C.Get(key); v != nil {
		return v
	}

	var data interface{}
	if uid == 1 {
		data = allMenus()
	} else {
		data = menusByRole(svc.GetRoleIDs(uid))
	}

	cache.C.SetTagged(key, data, svc.MenuCacheExpire, []string{svc.MenuCacheTag})
	return data
}

// allMenus 全部可用菜单（status=1，type ∈ {1,2,4}）
func allMenus() interface{} {
	rows := queryMenus("m.status = 1 AND m.type IN (1,2,4)", nil)
	return menu.MakeArtdMenus(rows)
}

// menusByRole 按角色取菜单
func menusByRole(roleIDs []int) interface{} {
	if len(roleIDs) == 0 {
		return []interface{}{}
	}
	rows := queryMenus(
		"m.status = 1 AND m.type IN (1,2,4) AND m.id IN (SELECT menu_id FROM sa_system_role_menu WHERE role_id IN ?)",
		roleIDs,
	)
	return menu.MakeArtdMenus(rows)
}

// queryMenus 查询菜单并转为 menu.Row。
// 注意：args 为 nil 时**不能**传给 Raw，否则 GORM 会报
// 「expected 0 arguments, got 1」并静默返回空结果集。
func queryMenus(where string, args interface{}) []menu.Row {
	sql := "SELECT m.id, m.parent_id, m.name, m.code, m.type, m.path, m.component, " +
		"m.icon, m.link_url, m.is_iframe, m.is_keep_alive, m.is_hidden, m.is_fixed_tab, m.is_full_page " +
		"FROM sa_system_menu m WHERE m.delete_time IS NULL AND " + where +
		" ORDER BY m.sort DESC"
	var raw []map[string]interface{}
	if args == nil {
		store.DB.Raw(sql).Scan(&raw)
	} else {
		store.DB.Raw(sql, args).Scan(&raw)
	}

	rows := normalizeRawRows(raw, "create_time", "update_time", "delete_time")
	return rows
}

// DictAll GET /core/system/dictAll —— 全部字典，形如 {code: [{label,value,color,...}]}
func (h *SystemHandler) DictAll(c *gin.Context) {
	response.Success(c, dictAll())
}

func dictAll() map[string]interface{} {
	if v := cache.C.Get(svc.DictCacheTag); v != nil {
		if m, ok := v.(map[string]interface{}); ok {
			return m
		}
	}

	result := map[string]interface{}{}

	// 字典类型：仅 status=1
	var types []model.DictType
	store.DB.Where("status = 1").Find(&types)

	for _, t := range types {
		var items []model.DictData
		store.DB.Where("type_id = ? AND status = 1", t.ID).Order("sort ASC").Find(&items)
		list := make([]gin.H, 0, len(items))
		for _, it := range items {
			list = append(list, gin.H{
				"id":    it.ID,
				"label": it.Label,
				"value": it.Value,
				"color": it.Color,
				"code":  it.Code,
			})
		}
		result[t.Code] = list
	}

	cache.C.SetTagged(svc.DictCacheTag, result, svc.DictCacheExpire, []string{svc.DictCacheTag})
	return result
}

// Statistics GET /core/system/statistics
func (h *SystemHandler) Statistics(c *gin.Context) {
	var userCount, attachCount, loginCount, operCount int64
	store.DB.Model(&model.User{}).Where("delete_time IS NULL").Count(&userCount)
	store.DB.Model(&model.Attachment{}).Where("delete_time IS NULL").Count(&attachCount)
	store.DB.Model(&model.LoginLog{}).Where("delete_time IS NULL").Count(&loginCount)
	store.DB.Model(&model.OperLog{}).Where("delete_time IS NULL").Count(&operCount)

	response.Success(c, gin.H{
		"user":    userCount,
		"attach":  attachCount,
		"login":   loginCount,
		"operate": operCount,
	})
}

// LoginChart GET /core/system/loginChart —— 近 10 天每日登录次数
func (h *SystemHandler) LoginChart(c *gin.Context) {
	const days = 10
	type row struct {
		LoginDate  string `gorm:"column:login_date"`
		LoginCount int    `gorm:"column:login_count"`
	}
	var rows []row
	start := time.Now().AddDate(0, 0, -(days - 1)).Format("2006-01-02 00:00:00")
	end := time.Now().AddDate(0, 0, 1).Format("2006-01-02 00:00:00")
	store.DB.Raw(`SELECT CAST(login_time AS DATE) AS login_date, COUNT(*) AS login_count
		FROM sa_system_login_log
		WHERE login_time >= ? AND login_time < ? AND delete_time IS NULL
		GROUP BY CAST(login_time AS DATE)`, start, end).Scan(&rows)

	counts := map[string]int{}
	for _, r := range rows {
		d := r.LoginDate
		if len(d) > 10 {
			d = d[:10]
		}
		counts[d] = r.LoginCount
	}

	dates := make([]string, 0, days)
	values := make([]int, 0, days)
	for i := days - 1; i >= 0; i-- {
		d := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		dates = append(dates, d)
		values = append(values, counts[d])
	}

	response.Success(c, gin.H{"login_count": values, "login_date": dates})
}

// LoginBarChart GET /core/system/loginBarChart —— 本年度每月登录次数
func (h *SystemHandler) LoginBarChart(c *gin.Context) {
	type row struct {
		LoginMonth int `gorm:"column:login_month"`
		LoginCount int `gorm:"column:login_count"`
	}
	var rows []row
	year := time.Now().Year()
	start := fmt.Sprintf("%d-01-01 00:00:00", year)
	end := fmt.Sprintf("%d-01-01 00:00:00", year+1)
	store.DB.Raw(`SELECT EXTRACT(MONTH FROM login_time) AS login_month, COUNT(*) AS login_count
		FROM sa_system_login_log
		WHERE login_time >= ? AND login_time < ? AND delete_time IS NULL
		GROUP BY EXTRACT(MONTH FROM login_time)`, start, end).Scan(&rows)

	counts := map[int]int{}
	for _, r := range rows {
		counts[r.LoginMonth] = r.LoginCount
	}

	months := make([]string, 0, 12)
	values := make([]int, 0, 12)
	for m := 1; m <= 12; m++ {
		months = append(months, fmt.Sprintf("%02d月", m))
		values = append(values, counts[m])
	}

	response.Success(c, gin.H{"login_count": values, "login_month": months})
}

// ClearAllCache GET /core/system/clearAllCache
func (h *SystemHandler) ClearAllCache(c *gin.Context) {
	uid := middleware.GetAdminID(c)
	svc.ClearUserCache(uid)
	response.SuccessMsg(c, []interface{}{}, "清除缓存成功!")
}

// GetLoginLogList GET /core/system/getLoginLogList —— 当前用户的登录日志
func (h *SystemHandler) GetLoginLogList(c *gin.Context) {
	username := middleware.GetAdminName(c)
	p := query.ParseParams(c)
	db := store.DB.Model(&model.LoginLog{}).Where("delete_time IS NULL").Where("username = ?", username)
	db = p.Apply(db, "id")
	var list []model.LoginLog
	response.Success(c, p.Paginate(db, &list))
}

// GetOperationLogList GET /core/system/getOperationLogList —— 当前用户的操作日志
func (h *SystemHandler) GetOperationLogList(c *gin.Context) {
	username := middleware.GetAdminName(c)
	p := query.ParseParams(c)
	db := store.DB.Model(&model.OperLog{}).Where("delete_time IS NULL").Where("username = ?", username)
	db = p.Apply(db, "id")
	var list []model.OperLog
	response.Success(c, p.Paginate(db, &list))
}

// GetUserList GET /core/system/getUserList —— 精简用户列表（选择器用）
func (h *SystemHandler) GetUserList(c *gin.Context) {
	p := query.ParseParams(c)
	db := store.DB.Model(&model.User{}).Where("delete_time IS NULL")
	if kw := c.Query("keyword"); kw != "" {
		db = query.LikeAny(db, []string{"username", "realname", "phone"}, kw)
	}
	if did := c.Query("dept_id"); did != "" {
		db = db.Where("dept_id = ?", did)
	}
	db = p.Apply(db, "id")
	var list []model.User
	// 需要逐行裁剪字段，故固定返回分页对象（该接口不会被前端以 saiType=all 调用）
	page := p.PageResult(db, &list)

	// 仅输出必要字段，隐藏 password
	items := make([]gin.H, 0, len(list))
	for _, u := range list {
		items = append(items, gin.H{
			"id": u.ID, "username": u.Username, "realname": u.Realname,
			"avatar": u.Avatar, "phone": u.Phone, "email": u.Email,
		})
	}
	page.Data = items
	response.Success(c, page)
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
