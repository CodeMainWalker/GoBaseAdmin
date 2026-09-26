package router

import (
	"github.com/gin-gonic/gin"

	"github.com/CodeMainWalker/GoBaseAdmin/internal/handler"
)

// RegisterSystemRoutes 注册 /core 下的系统管理接口（用户/角色/部门/岗位/菜单/字典/配置/日志）
func RegisterSystemRoutes(g *gin.RouterGroup) {
	user := handler.NewUserHandler()
	role := handler.NewRoleHandler()
	dept := handler.NewDeptHandler()
	post := handler.NewPostHandler()
	menuH := handler.NewMenuHandler()
	dictType := handler.NewDictTypeHandler()
	dictData := handler.NewDictDataHandler()
	cfg := handler.NewConfigHandler()
	cfgGroup := handler.NewConfigGroupHandler()
	mail := handler.NewMailHandler()
	logs := handler.NewLogHandler()

	// 用户管理
	FastRoute(g, CRUD{
		Name:  "user",
		Index: user.Index, Read: user.Read, Save: user.Save,
		Update: user.Update, Destroy: user.Destroy,
		IndexPerm:   Perm{"用户数据列表", "core:user:index"},
		ReadPerm:    Perm{"用户数据读取", "core:user:read"},
		SavePerm:    Perm{"用户数据保存", "core:user:save"},
		UpdatePerm:  Perm{"用户数据更新", "core:user:update"},
		DestroyPerm: Perm{"用户数据删除", "core:user:destroy"},
	})
	g.POST("/user/updateInfo", permMiddleware(Perm{"用户修改资料", ""}), user.UpdateInfo)
	g.POST("/user/modifyPassword", permMiddleware(Perm{"用户修改密码", ""}), user.ModifyPassword)
	g.POST("/user/clearCache", permMiddleware(Perm{"清理用户缓存", "core:user:cache"}), user.ClearCache)
	g.POST("/user/initUserPassword", permMiddleware(Perm{"修改用户密码", "core:user:password"}), user.InitUserPassword)
	g.POST("/user/setHomePage", permMiddleware(Perm{"设置用户首页", "core:user:home"}), user.SetHomePage)

	// 角色管理
	FastRoute(g, CRUD{
		Name:  "role",
		Index: role.Index, Read: role.Read, Save: role.Save,
		Update: role.Update, Destroy: role.Destroy,
		IndexPerm:   Perm{"角色数据列表", "core:role:index"},
		ReadPerm:    Perm{"角色数据读取", "core:role:read"},
		SavePerm:    Perm{"角色数据保存", "core:role:save"},
		UpdatePerm:  Perm{"角色数据更新", "core:role:update"},
		DestroyPerm: Perm{"角色数据删除", "core:role:destroy"},
	})
	g.GET("/role/accessRole", role.AccessRole)
	g.GET("/role/getMenuByRole", role.GetMenuByRole)
	g.POST("/role/menuPermission", permMiddleware(Perm{"角色菜单权限", "core:role:menu"}), role.MenuPermission)

	// 部门管理
	FastRoute(g, CRUD{
		Name:  "dept",
		Index: dept.Index, Read: dept.Read, Save: dept.Save,
		Update: dept.Update, Destroy: dept.Destroy,
		IndexPerm:   Perm{"部门数据列表", "core:dept:index"},
		ReadPerm:    Perm{"部门数据读取", "core:dept:read"},
		SavePerm:    Perm{"部门数据保存", "core:dept:save"},
		UpdatePerm:  Perm{"部门数据更新", "core:dept:update"},
		DestroyPerm: Perm{"部门数据删除", "core:dept:destroy"},
	})
	g.GET("/dept/accessDept", dept.AccessDept)

	// 岗位管理
	FastRoute(g, CRUD{
		Name:  "post",
		Index: post.Index, Read: post.Read, Save: post.Save,
		Update: post.Update, Destroy: post.Destroy,
		IndexPerm:   Perm{"岗位数据列表", "core:post:index"},
		ReadPerm:    Perm{"岗位数据读取", "core:post:read"},
		SavePerm:    Perm{"岗位数据保存", "core:post:save"},
		UpdatePerm:  Perm{"岗位数据更新", "core:post:update"},
		DestroyPerm: Perm{"岗位数据删除", "core:post:destroy"},
	})
	g.GET("/post/accessPost", post.AccessPost)

	// 菜单管理
	FastRoute(g, CRUD{
		Name:  "menu",
		Index: menuH.Index, Read: menuH.Read, Save: menuH.Save,
		Update: menuH.Update, Destroy: menuH.Destroy,
		IndexPerm:   Perm{"菜单数据列表", "core:menu:index"},
		ReadPerm:    Perm{"菜单数据读取", "core:menu:read"},
		SavePerm:    Perm{"菜单数据保存", "core:menu:save"},
		UpdatePerm:  Perm{"菜单数据更新", "core:menu:update"},
		DestroyPerm: Perm{"菜单数据删除", "core:menu:destroy"},
	})
	g.GET("/menu/accessMenu", menuH.AccessMenu)

	// 字典类型
	FastRoute(g, CRUD{
		Name:  "dictType",
		Index: dictType.Index, Read: dictType.Read, Save: dictType.Save,
		Update: dictType.Update, Destroy: dictType.Destroy,
		IndexPerm:   Perm{"字典类型列表", "core:dict:index"},
		ReadPerm:    Perm{"字典类型读取", "core:dict:index"},
		SavePerm:    Perm{"字典类型管理", "core:dict:edit"},
		UpdatePerm:  Perm{"字典类型管理", "core:dict:edit"},
		DestroyPerm: Perm{"字典类型管理", "core:dict:edit"},
	})

	// 字典数据
	FastRoute(g, CRUD{
		Name:  "dictData",
		Index: dictData.Index, Read: dictData.Read, Save: dictData.Save,
		Update: dictData.Update, Destroy: dictData.Destroy,
		IndexPerm:   Perm{"字典数据列表", "core:dict:index"},
		ReadPerm:    Perm{"字典数据读取", "core:dict:index"},
		SavePerm:    Perm{"字典数据管理", "core:dict:edit"},
		UpdatePerm:  Perm{"字典数据管理", "core:dict:edit"},
		DestroyPerm: Perm{"字典数据管理", "core:dict:edit"},
	})

	// 系统配置
	FastRoute(g, CRUD{
		Name:  "config",
		Index: cfg.Index, Read: cfg.Read, Save: cfg.Save,
		Update: cfg.Update, Destroy: cfg.Destroy,
		IndexPerm:   Perm{"系统设置列表", "core:config:index"},
		ReadPerm:    Perm{"系统设置列表", "core:config:index"},
		SavePerm:    Perm{"系统设置管理", "core:config:edit"},
		UpdatePerm:  Perm{"系统设置管理", "core:config:edit"},
		DestroyPerm: Perm{"系统设置管理", "core:config:edit"},
	})
	g.POST("/config/batchUpdate", permMiddleware(Perm{"系统设置修改", "core:config:update"}), cfg.BatchUpdate)

	// 配置分组
	FastRoute(g, CRUD{
		Name:  "configGroup",
		Index: cfgGroup.Index, Read: cfgGroup.Read, Save: cfgGroup.Save,
		Update: cfgGroup.Update, Destroy: cfgGroup.Destroy,
		IndexPerm:   Perm{"配置分组列表", "core:config:index"},
		ReadPerm:    Perm{"配置分组列表", "core:config:index"},
		SavePerm:    Perm{"配置分组管理", "core:config:edit"},
		UpdatePerm:  Perm{"配置分组管理", "core:config:edit"},
		DestroyPerm: Perm{"配置分组管理", "core:config:edit"},
	})

	// 邮件记录
	FastRoute(g, CRUD{
		Name:  "email",
		Index: mail.Index, Read: mail.Read, Destroy: mail.Destroy,
		IndexPerm:   Perm{"邮件记录列表", "core:email:index"},
		ReadPerm:    Perm{"邮件记录列表", "core:email:index"},
		DestroyPerm: Perm{"邮件记录删除", "core:email:destroy"},
	})

	// 日志
	g.GET("/logs/getLoginLogPageList", permMiddleware(Perm{"登录日志列表", "core:logs:login"}), logs.GetLoginLogPageList)
	g.DELETE("/logs/deleteLoginLog", permMiddleware(Perm{"登录日志删除", "core:logs:deleteLogin"}), logs.DeleteLoginLog)
	g.GET("/logs/getOperLogPageList", permMiddleware(Perm{"操作日志列表", "core:logs:Oper"}), logs.GetOperLogPageList)
	g.DELETE("/logs/deleteOperLog", permMiddleware(Perm{"操作日志删除", "core:logs:deleteOper"}), logs.DeleteOperLog)
}
