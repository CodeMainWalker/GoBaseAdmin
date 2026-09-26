package router

import (
	"github.com/gin-gonic/gin"
)

// CRUD 描述一个业务模块的处理器集合
type CRUD struct {
	// 资源名，如 "user" → 注册到 /core/user/*
	Name string
	// 各动作的处理器，nil 表示不注册该路由
	Index   gin.HandlerFunc
	Read    gin.HandlerFunc
	Save    gin.HandlerFunc
	Update  gin.HandlerFunc
	Destroy gin.HandlerFunc
	Import  gin.HandlerFunc
	Export  gin.HandlerFunc

	// 权限元信息
	IndexPerm   Perm
	ReadPerm    Perm
	SavePerm    Perm
	UpdatePerm  Perm
	DestroyPerm Perm
	ImportPerm  Perm
	ExportPerm  Perm
}

// FastRoute 注册一个业务模块的标准资源路由（与上游 SaiAdmin 6.x 的
// 快速路由注册方式一致）：按 index/read/save/update/destroy/import/export 注册。
//
// 方法约定（必须与前端 api 层完全一致）：
//
//	GET    /{name}/index
//	GET    /{name}/read?id=
//	POST   /{name}/save
//	PUT    /{name}/update
//	DELETE /{name}/destroy
func FastRoute(g *gin.RouterGroup, c CRUD) {
	name := "/" + c.Name

	if c.Index != nil {
		g.GET(name+"/index", permMiddleware(c.IndexPerm), c.Index)
	}
	if c.Save != nil {
		g.POST(name+"/save", permMiddleware(c.SavePerm), c.Save)
	}
	if c.Update != nil {
		g.PUT(name+"/update", permMiddleware(c.UpdatePerm), c.Update)
	}
	if c.Read != nil {
		g.GET(name+"/read", permMiddleware(c.ReadPerm), c.Read)
	}
	if c.Destroy != nil {
		g.DELETE(name+"/destroy", permMiddleware(c.DestroyPerm), c.Destroy)
	}
	if c.Import != nil {
		g.POST(name+"/import", permMiddleware(c.ImportPerm), c.Import)
	}
	if c.Export != nil {
		g.POST(name+"/export", permMiddleware(c.ExportPerm), c.Export)
	}
}
