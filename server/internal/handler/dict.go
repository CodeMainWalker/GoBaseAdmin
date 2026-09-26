package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/CodeMainWalker/GoBaseAdmin/internal/cache"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/model"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/query"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/response"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/store"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/svc"
)

// ---------- 字典类型 ----------

type DictTypeHandler struct{}

func NewDictTypeHandler() *DictTypeHandler { return &DictTypeHandler{} }

func (h *DictTypeHandler) Index(c *gin.Context) {
	p := query.ParseParams(c)
	db := store.DB.Model(&model.DictType{}).Where("delete_time IS NULL")
	if name := c.Query("name"); name != "" {
		db = db.Where("name LIKE ?", "%"+name+"%")
	}
	if code := c.Query("code"); code != "" {
		db = db.Where("code LIKE ?", "%"+code+"%")
	}
	db = query.ApplySearch(db, map[string]interface{}{"status": c.Query("status")})
	db = p.Apply(db, "id")
	var list []model.DictType
	response.Success(c, p.Paginate(db, &list))
}

func (h *DictTypeHandler) Read(c *gin.Context) {
	var m model.DictType
	if err := store.DB.Where("id = ?", getIDParam(c)).First(&m).Error; err != nil {
		response.Fail(c, "数据不存在")
		return
	}
	response.Success(c, m)
}

func (h *DictTypeHandler) Save(c *gin.Context) {
	data := bindBody(c)
	m := model.DictType{}
	setStructFields(&m, data)
	if err := store.DB.Create(&m).Error; err != nil {
		response.Fail(c, "添加失败")
		return
	}
	cache.C.ClearTag(svc.DictCacheTag)
	response.Success(c, "添加成功")
}

func (h *DictTypeHandler) Update(c *gin.Context) {
	data := bindBody(c)
	updates := pick(data, "name", "code", "status", "remark")
	if err := store.DB.Model(&model.DictType{}).Where("id = ?", toStr(data["id"])).Updates(updates).Error; err != nil {
		response.Fail(c, "修改失败")
		return
	}
	cache.C.ClearTag(svc.DictCacheTag)
	response.Success(c, "修改成功")
}

func (h *DictTypeHandler) Destroy(c *gin.Context) {
	ids := query.IDsFrom(getIDsParam(c))
	if len(ids) == 0 {
		response.Fail(c, "参数错误，请检查")
		return
	}
	now := model.Now()
	store.DB.Model(&model.DictType{}).Where("id IN ?", ids).Update("delete_time", now)
	store.DB.Model(&model.DictData{}).Where("type_id IN ?", ids).Update("delete_time", now)
	cache.C.ClearTag(svc.DictCacheTag)
	response.Success(c, "操作成功")
}

// ---------- 字典数据 ----------

type DictDataHandler struct{}

func NewDictDataHandler() *DictDataHandler { return &DictDataHandler{} }

func (h *DictDataHandler) Index(c *gin.Context) {
	p := query.ParseParams(c)
	db := store.DB.Model(&model.DictData{}).Where("delete_time IS NULL")
	if label := c.Query("label"); label != "" {
		db = db.Where("label LIKE ?", "%"+label+"%")
	}
	db = query.ApplySearch(db, map[string]interface{}{
		"type_id": c.Query("type_id"),
		"status":  c.Query("status"),
	})
	db = p.Apply(db, "sort")
	var list []model.DictData
	response.Success(c, p.Paginate(db, &list))
}

func (h *DictDataHandler) Read(c *gin.Context) {
	var m model.DictData
	if err := store.DB.Where("id = ?", getIDParam(c)).First(&m).Error; err != nil {
		response.Fail(c, "数据不存在")
		return
	}
	response.Success(c, m)
}

func (h *DictDataHandler) Save(c *gin.Context) {
	data := bindBody(c)
	m := model.DictData{}
	setStructFields(&m, data)
	if m.Value == "" {
		response.Fail(c, "字典键值不能为空")
		return
	}
	if err := store.DB.Create(&m).Error; err != nil {
		response.Fail(c, "添加失败")
		return
	}
	cache.C.ClearTag(svc.DictCacheTag)
	response.Success(c, "添加成功")
}

func (h *DictDataHandler) Update(c *gin.Context) {
	data := bindBody(c)
	updates := pick(data, "type_id", "label", "value", "color", "code", "sort", "status", "remark")
	if err := store.DB.Model(&model.DictData{}).Where("id = ?", toStr(data["id"])).Updates(updates).Error; err != nil {
		response.Fail(c, "修改失败")
		return
	}
	cache.C.ClearTag(svc.DictCacheTag)
	response.Success(c, "修改成功")
}

func (h *DictDataHandler) Destroy(c *gin.Context) {
	ids := query.IDsFrom(getIDsParam(c))
	if len(ids) == 0 {
		response.Fail(c, "参数错误，请检查")
		return
	}
	now := model.Now()
	store.DB.Model(&model.DictData{}).Where("id IN ?", ids).Update("delete_time", now)
	cache.C.ClearTag(svc.DictCacheTag)
	response.Success(c, "操作成功")
}
