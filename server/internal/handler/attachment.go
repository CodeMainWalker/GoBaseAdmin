package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/CodeMainWalker/GoBaseAdmin/internal/menu"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/model"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/query"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/response"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/store"
)

// AttachmentHandler 附件管理
type AttachmentHandler struct{}

func NewAttachmentHandler() *AttachmentHandler { return &AttachmentHandler{} }

// Index GET /core/attachment/index
func (h *AttachmentHandler) Index(c *gin.Context) {
	p := query.ParseParams(c)
	db := store.DB.Model(&model.Attachment{}).Where("delete_time IS NULL")
	if v := c.Query("origin_name"); v != "" {
		db = db.Where("origin_name LIKE ?", "%"+v+"%")
	}
	if v := c.Query("category_id"); v != "" {
		db = db.Where("category_id = ?", v)
	}
	if v := c.Query("mime_type"); v != "" {
		db = db.Where("mime_type LIKE ?", v+"%")
	}
	db = p.Apply(db, "id")
	var list []model.Attachment
	response.Success(c, p.Paginate(db, &list))
}

// Update PUT /core/attachment/update —— 仅允许改分类与备注
func (h *AttachmentHandler) Update(c *gin.Context) {
	data := bindBody(c)
	updates := pick(data, "category_id", "remark", "origin_name")
	if err := store.DB.Model(&model.Attachment{}).Where("id = ?", toStr(data["id"])).Updates(updates).Error; err != nil {
		response.Fail(c, "修改失败")
		return
	}
	response.Success(c, "修改成功")
}

// Destroy DELETE /core/attachment/destroy
func (h *AttachmentHandler) Destroy(c *gin.Context) {
	ids := query.IDsFrom(getIDsParam(c))
	if len(ids) == 0 {
		response.Fail(c, "参数错误，请检查")
		return
	}
	now := model.Now()
	store.DB.Model(&model.Attachment{}).Where("id IN ?", ids).Update("delete_time", now)
	response.Success(c, "操作成功")
}

// Move POST /core/attachment/move —— 批量移动附件到指定分类
func (h *AttachmentHandler) Move(c *gin.Context) {
	data := bindBody(c)
	categoryID := toInt(data["category_id"])
	ids := query.IDsFrom(data["ids"])
	if len(ids) == 0 {
		response.Fail(c, "参数错误，请检查")
		return
	}
	store.DB.Model(&model.Attachment{}).Where("id IN ?", ids).Update("category_id", categoryID)
	response.Success(c, "操作成功")
}

// CategoryHandler 附件分类
type CategoryHandler struct{}

func NewCategoryHandler() *CategoryHandler { return &CategoryHandler{} }

// Index GET /core/category/index —— 树形结构
func (h *CategoryHandler) Index(c *gin.Context) {
	isTree := c.Query("tree") == "true"

	db := store.DB.Table("sa_system_category").Where("delete_time IS NULL")
	if v := c.Query("category_name"); v != "" {
		db = db.Where("category_name LIKE ?", "%"+v+"%")
	}

	var raw []map[string]interface{}
	if isTree {
		db.Select("id, id AS value, category_name AS label, parent_id").Order("sort DESC").Scan(&raw)
	} else {
		db.Select("*").Order("sort DESC").Scan(&raw)
	}

	rows := normalizeRawRows(raw, "create_time", "update_time", "delete_time")
	response.Success(c, menu.MakeTree(rows, "id", "parent_id"))
}

// Read GET /core/category/read?id=
func (h *CategoryHandler) Read(c *gin.Context) {
	var m model.Category
	if err := store.DB.Where("id = ?", getIDParam(c)).First(&m).Error; err != nil {
		response.Fail(c, "数据不存在")
		return
	}
	response.Success(c, m)
}

// Save POST /core/category/save
func (h *CategoryHandler) Save(c *gin.Context) {
	data := bindBody(c)
	data = categoryLevel(data)
	m := model.Category{}
	setStructFields(&m, data)
	if err := store.DB.Create(&m).Error; err != nil {
		response.Fail(c, "添加失败")
		return
	}
	response.Success(c, "添加成功")
}

// Update PUT /core/category/update
func (h *CategoryHandler) Update(c *gin.Context) {
	data := bindBody(c)
	id := toUint(data["id"])
	if toUint(data["parent_id"]) == id {
		response.Fail(c, "上级分类和当前分类不能相同")
		return
	}
	data = categoryLevel(data)
	updates := pick(data, "parent_id", "level", "category_name", "sort", "status", "remark")
	if err := store.DB.Model(&model.Category{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		response.Fail(c, "修改失败")
		return
	}
	response.Success(c, "修改成功")
}

// Destroy DELETE /core/category/destroy
func (h *CategoryHandler) Destroy(c *gin.Context) {
	ids := query.IDsFrom(getIDsParam(c))
	if len(ids) == 0 {
		response.Fail(c, "参数错误，请检查")
		return
	}
	var children int64
	store.DB.Model(&model.Category{}).Where("parent_id IN ? AND delete_time IS NULL", ids).Count(&children)
	if children > 0 {
		response.Fail(c, "该分类下存在子分类，请先删除子分类")
		return
	}
	now := model.Now()
	store.DB.Model(&model.Category{}).Where("id IN ?", ids).Update("delete_time", now)
	response.Success(c, "操作成功")
}

func categoryLevel(data map[string]interface{}) map[string]interface{} {
	parentID := toUint(data["parent_id"])
	if parentID == 0 {
		data["level"] = "0"
		data["parent_id"] = 0
		return data
	}
	var parent model.Category
	if err := store.DB.Where("id = ?", parentID).First(&parent).Error; err == nil {
		data["level"] = parent.Level + itoa(int(parent.ID)) + ","
	}
	return data
}
