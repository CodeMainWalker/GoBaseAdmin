package router

import (
	"github.com/gin-gonic/gin"

	"github.com/CodeMainWalker/GoBaseAdmin/internal/handler"
)

// RegisterSafeguardRoutes 注册运维管理接口（附件/分类/数据库/服务监控）
func RegisterSafeguardRoutes(g *gin.RouterGroup) {
	att := handler.NewAttachmentHandler()
	cat := handler.NewCategoryHandler()
	srv := handler.NewServerHandler()
	dbH := handler.NewDatabaseHandler()

	// 附件管理
	FastRoute(g, CRUD{
		Name:  "attachment",
		Index: att.Index, Update: att.Update, Destroy: att.Destroy,
		IndexPerm:   Perm{"附件数据列表", "core:attachment:index"},
		UpdatePerm:  Perm{"附件管理", "core:attachment:edit"},
		DestroyPerm: Perm{"附件管理", "core:attachment:edit"},
	})
	g.POST("/attachment/move", permMiddleware(Perm{"附件管理", "core:attachment:edit"}), att.Move)

	// 附件分类
	FastRoute(g, CRUD{
		Name:  "category",
		Index: cat.Index, Read: cat.Read, Save: cat.Save,
		Update: cat.Update, Destroy: cat.Destroy,
		IndexPerm:   Perm{"附件分类列表", "core:attachment:index"},
		ReadPerm:    Perm{"附件分类列表", "core:attachment:index"},
		SavePerm:    Perm{"附件分类管理", "core:attachment:edit"},
		UpdatePerm:  Perm{"附件分类管理", "core:attachment:edit"},
		DestroyPerm: Perm{"附件分类管理", "core:attachment:edit"},
	})

	// 服务监控
	g.GET("/server/monitor", permMiddleware(Perm{"服务监控", "core:server:monitor"}), srv.Monitor)
	g.GET("/server/cache", permMiddleware(Perm{"缓存信息", "core:server:cache"}), srv.Cache)
	g.POST("/server/clear", permMiddleware(Perm{"缓存清理", "core:server:clear"}), srv.Clear)

	// 数据表维护
	g.GET("/database/index", permMiddleware(Perm{"数据表列表", "core:database:index"}), dbH.Index)
	g.GET("/database/recycle", permMiddleware(Perm{"回收站数据", "core:recycle:index"}), dbH.Recycle)
	g.DELETE("/database/delete", permMiddleware(Perm{"回收站管理", "core:recycle:edit"}), dbH.Delete)
	g.POST("/database/recovery", permMiddleware(Perm{"回收站管理", "core:recycle:edit"}), dbH.Recovery)
	g.GET("/database/dataSource", permMiddleware(Perm{"数据表列表", "core:database:index"}), dbH.DataSource)
	g.GET("/database/detailed", permMiddleware(Perm{"数据表维护", "core:database:edit"}), dbH.Detailed)
	g.POST("/database/optimize", permMiddleware(Perm{"数据表维护", "core:database:edit"}), dbH.Optimize)
	g.POST("/database/fragment", permMiddleware(Perm{"数据表维护", "core:database:edit"}), dbH.Fragment)
}

// RegisterUploadRoutes 注册上传接口
func RegisterUploadRoutes(g *gin.RouterGroup) {
	up := handler.NewUploadHandler()

	g.GET("/system/getResourceCategory", permMiddleware(Perm{"附件列表读取", "core:system:resource"}), up.ResourceCategory)
	g.GET("/system/getResourceList", permMiddleware(Perm{"附件列表读取", "core:system:resource"}), up.ResourceList)
	g.POST("/system/uploadImage", permMiddleware(Perm{"上传图片", "core:system:uploadImage"}), up.UploadImage)
	g.POST("/system/uploadFile", permMiddleware(Perm{"上传文件", "core:system:uploadFile"}), up.UploadFile)
	g.POST("/system/chunkUpload", permMiddleware(Perm{"上传文件", "core:system:chunkUpload"}), up.ChunkUpload)
	g.POST("/system/saveNetworkImage", permMiddleware(Perm{"上传网络图片", "core:system:uploadImage"}), up.SaveNetworkImage)
}
