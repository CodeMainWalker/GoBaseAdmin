// Package svc 承载用户上下文组装与各类缓存的读写。
package svc

import (
	"github.com/CodeMainWalker/GoBaseAdmin/internal/cache"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/model"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/store"
)

// 缓存 TTL 与键前缀，沿用上游 SaiAdmin 6.x 的配置取值
const (
	UserCachePrefix = "saiadmin:user_cache:info_"
	UserCacheExpire = 4 * 60 * 60
	UserCacheDept   = "saiadmin:user_cache:dept_"
	UserCacheRole   = "saiadmin:user_cache:role_"
	UserCachePost   = "saiadmin:user_cache:post_"

	ButtonCachePrefix = "saiadmin:button_cache:user_"
	ButtonCacheExpire = 2 * 60 * 60
	ButtonCacheTag    = "saiadmin:button_cache"
	ButtonCacheRole   = "saiadmin:button_cache:role_"

	MenuCachePrefix = "saiadmin:menu_cache:user_"
	MenuCacheExpire = 7 * 24 * 60 * 60
	MenuCacheTag    = "saiadmin:menu_cache"

	DictCacheTag    = "saiadmin:dict_cache"
	DictCacheExpire = 365 * 24 * 60 * 60

	ConfigCachePrefix = "saiadmin:config_cache:config_"
	ConfigCacheTag    = "saiadmin:config_cache"
	ConfigCacheExpire = 365 * 24 * 60 * 60
)

// UserInfo 当前登录用户的完整上下文（缓存在 UserCachePrefix 键下）
type UserInfo struct {
	ID        int                    `json:"id"`
	Username  string                 `json:"username"`
	Password  string                 `json:"-"`
	Realname  string                 `json:"realname"`
	Gender    string                 `json:"gender"`
	Avatar    string                 `json:"avatar"`
	Email     string                 `json:"email"`
	Phone     string                 `json:"phone"`
	Signed    string                 `json:"signed"`
	Dashboard string                 `json:"dashboard"`
	DeptID    *uint64                `json:"dept_id"`
	IsSuper   int                    `json:"is_super"`
	Status    int                    `json:"status"`
	DeptList  *model.Dept            `json:"deptList"`
	RoleList  []model.Role           `json:"roleList"`
	PostList  []model.Post           `json:"postList"`
	Extra     map[string]interface{} `json:"-"`
}

// GetUserInfo 读取用户上下文（带缓存）。
// id=1 的超级管理员跳过缓存，避免权限变更后需要手动清缓存。
func GetUserInfo(uid int) *UserInfo {
	if uid <= 0 {
		return nil
	}
	if v := cache.C.Get(UserCachePrefix + itoa(uid)); v != nil {
		if info, ok := v.(*UserInfo); ok {
			return info
		}
	}
	info := buildUserInfo(uid)
	if info == nil {
		return nil
	}
	if uid != 1 {
		cache.C.SetTagged(UserCachePrefix+itoa(uid), info, UserCacheExpire,
			userTags(info))
	}
	return info
}

func userTags(info *UserInfo) []string {
	tags := []string{}
	if info.DeptList != nil {
		tags = append(tags, UserCacheDept+itoa(int(info.DeptList.ID)))
	}
	for _, r := range info.RoleList {
		tags = append(tags, UserCacheRole+itoa(int(r.ID)))
	}
	for _, p := range info.PostList {
		tags = append(tags, UserCachePost+itoa(int(p.ID)))
	}
	return tags
}

// buildUserInfo 从数据库组装用户上下文（部门/角色/岗位一并装入）
func buildUserInfo(uid int) *UserInfo {
	var u model.User
	if err := store.DB.Where("id = ?", uid).First(&u).Error; err != nil {
		return nil
	}

	info := &UserInfo{
		ID:        int(u.ID),
		Username:  u.Username,
		Password:  u.Password,
		Realname:  u.Realname,
		Gender:    u.Gender,
		Avatar:    u.Avatar,
		Email:     u.Email,
		Phone:     u.Phone,
		Signed:    u.Signed,
		Dashboard: u.Dashboard,
		DeptID:    u.DeptID,
		IsSuper:   u.IsSuper,
		Status:    u.Status,
		RoleList:  []model.Role{},
		PostList:  []model.Post{},
	}

	// 主部门
	if u.DeptID != nil && *u.DeptID > 0 {
		var d model.Dept
		if err := store.DB.Where("id = ?", *u.DeptID).First(&d).Error; err == nil {
			info.DeptList = &d
		}
	}

	// 角色
	store.DB.Raw(`SELECT r.* FROM sa_system_role r
		JOIN sa_system_user_role ur ON ur.role_id = r.id
		WHERE ur.user_id = ? AND r.delete_time IS NULL`, uid).Scan(&info.RoleList)

	// 岗位
	store.DB.Raw(`SELECT p.* FROM sa_system_post p
		JOIN sa_system_user_post up ON up.post_id = p.id
		WHERE up.user_id = ? AND p.delete_time IS NULL`, uid).Scan(&info.PostList)

	return info
}

// ClearUserCache 清理单个用户的全部相关缓存
func ClearUserCache(uid int) {
	cache.C.Delete(UserCachePrefix + itoa(uid))
	cache.C.Delete(ButtonCachePrefix + itoa(uid))
	cache.C.Delete(MenuCachePrefix + itoa(uid))
}

// GetUserAuth 读取用户的按钮权限 slug 列表（命中 ButtonCachePrefix 时直接返回）
func GetUserAuth(uid int) []string {
	if uid <= 0 {
		return []string{}
	}
	if v := cache.C.Get(ButtonCachePrefix + itoa(uid)); v != nil {
		if s, ok := v.([]string); ok {
			return s
		}
	}
	list := queryUserAuth(uid)
	tags := []string{ButtonCacheTag}
	for _, rid := range GetRoleIDs(uid) {
		tags = append(tags, ButtonCacheRole+itoa(rid))
	}
	cache.C.SetTagged(ButtonCachePrefix+itoa(uid), list, ButtonCacheExpire, tags)
	return list
}

func queryUserAuth(uid int) []string {
	roleIDs := GetRoleIDs(uid)
	if len(roleIDs) == 0 {
		return []string{}
	}
	var slugs []string
	store.DB.Raw(`SELECT DISTINCT m.slug FROM sa_system_menu m
		JOIN sa_system_role_menu rm ON rm.menu_id = m.id
		WHERE m.type = 3 AND m.status = 1 AND m.delete_time IS NULL
		  AND rm.role_id IN ?`, roleIDs).Scan(&slugs)
	out := []string{}
	for _, s := range slugs {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// GetRoleIDs 读取用户角色 ID 列表
func GetRoleIDs(uid int) []int {
	var ids []int
	store.DB.Raw(`SELECT role_id FROM sa_system_user_role WHERE user_id = ?`, uid).Scan(&ids)
	return ids
}

// ClearAuthByRole 清理某角色相关的权限缓存（对应 clearUserAuthByRoleId）
func ClearAuthByRole(roleIDs []int) {
	tags := []string{ButtonCacheTag}
	for _, id := range roleIDs {
		tags = append(tags, ButtonCacheRole+itoa(id))
	}
	cache.C.ClearTags(tags)
}

// ClearMenuCache 清理全部菜单缓存
func ClearMenuCache() {
	cache.C.ClearTag(MenuCacheTag)
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
