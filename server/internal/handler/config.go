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

// ---------- 系统配置 ----------

type ConfigHandler struct{}

func NewConfigHandler() *ConfigHandler { return &ConfigHandler{} }

func (h *ConfigHandler) Index(c *gin.Context) {
	p := query.ParseParams(c)
	db := store.DB.Model(&model.Config{}).Where("delete_time IS NULL")
	db = query.ApplySearch(db, map[string]interface{}{
		"group_id": c.Query("group_id"),
		"name":     c.Query("name"),
		"key":      c.Query("key"),
	})
	// 该接口默认按 sort desc 排序
	if p.OrderField == "" {
		p.OrderType = "desc"
	}
	db = p.Apply(db, "sort")
	var list []model.Config
	response.Success(c, p.Paginate(db, &list))
}

func (h *ConfigHandler) Read(c *gin.Context) {
	var m model.Config
	if err := store.DB.Where("id = ?", getIDParam(c)).First(&m).Error; err != nil {
		response.Fail(c, "数据不存在")
		return
	}
	response.Success(c, m)
}

func (h *ConfigHandler) Save(c *gin.Context) {
	data := bindBody(c)
	m := model.Config{}
	setStructFields(&m, data)
	if err := store.DB.Create(&m).Error; err != nil {
		response.Fail(c, "添加失败")
		return
	}
	cache.C.ClearTag(svc.ConfigCacheTag)
	response.Success(c, "添加成功")
}

func (h *ConfigHandler) Update(c *gin.Context) {
	data := bindBody(c)
	updates := pick(data, "group_id", "key", "value", "name", "input_type",
		"config_select_data", "sort", "remark")
	// config_select_data 走 map 更新，需要先把数组编码成 JSON 文本
	if v, ok := updates["config_select_data"]; ok {
		updates["config_select_data"] = model.EncodeJSONList(v)
	}
	if err := store.DB.Model(&model.Config{}).Where("id = ?", toStr(data["id"])).Updates(updates).Error; err != nil {
		response.Fail(c, "修改失败")
		return
	}
	cache.C.ClearTag(svc.ConfigCacheTag)
	response.Success(c, "修改成功")
}

func (h *ConfigHandler) Destroy(c *gin.Context) {
	ids := query.IDsFrom(getIDsParam(c))
	if len(ids) == 0 {
		response.Fail(c, "请选择要删除的数据")
		return
	}
	now := model.Now()
	store.DB.Model(&model.Config{}).Where("id IN ?", ids).Update("delete_time", now)
	cache.C.ClearTag(svc.ConfigCacheTag)
	response.Success(c, "删除成功")
}

// BatchUpdate POST /core/config/batchUpdate —— 按分组批量保存配置值
func (h *ConfigHandler) BatchUpdate(c *gin.Context) {
	data := bindBody(c)
	groupID := toUint(data["group_id"])
	if groupID == 0 {
		response.Fail(c, "参数错误")
		return
	}

	cfg, ok := data["config"]
	if !ok || cfg == nil {
		response.Fail(c, "参数错误")
		return
	}

	// config 形如 {"key": value, ...}
	if m, ok := cfg.(map[string]interface{}); ok {
		for k, v := range m {
			store.DB.Model(&model.Config{}).
				Where("group_id = ? AND `key` = ?", groupID, k).
				Update("value", toStr(v))
		}
	}

	cache.C.ClearTag(svc.ConfigCacheTag)
	response.Success(c, "操作成功")
}

// ---------- 配置分组 ----------

type ConfigGroupHandler struct{}

func NewConfigGroupHandler() *ConfigGroupHandler { return &ConfigGroupHandler{} }

func (h *ConfigGroupHandler) Index(c *gin.Context) {
	p := query.ParseParams(c)
	db := store.DB.Model(&model.ConfigGroup{}).Where("delete_time IS NULL")
	if name := c.Query("name"); name != "" {
		db = db.Where("name LIKE ?", "%"+name+"%")
	}
	db = p.Apply(db, "id")
	var list []model.ConfigGroup
	response.Success(c, p.Paginate(db, &list))
}

func (h *ConfigGroupHandler) Read(c *gin.Context) {
	var m model.ConfigGroup
	if err := store.DB.Where("id = ?", getIDParam(c)).First(&m).Error; err != nil {
		response.Fail(c, "数据不存在")
		return
	}
	response.Success(c, m)
}

func (h *ConfigGroupHandler) Save(c *gin.Context) {
	data := bindBody(c)
	m := model.ConfigGroup{}
	setStructFields(&m, data)
	if err := store.DB.Create(&m).Error; err != nil {
		response.Fail(c, "添加失败")
		return
	}
	cache.C.ClearTag(svc.ConfigCacheTag)
	response.Success(c, "添加成功")
}

func (h *ConfigGroupHandler) Update(c *gin.Context) {
	data := bindBody(c)
	updates := pick(data, "name", "code", "remark")
	if err := store.DB.Model(&model.ConfigGroup{}).Where("id = ?", toStr(data["id"])).Updates(updates).Error; err != nil {
		response.Fail(c, "修改失败")
		return
	}
	cache.C.ClearTag(svc.ConfigCacheTag)
	response.Success(c, "修改成功")
}

func (h *ConfigGroupHandler) Destroy(c *gin.Context) {
	ids := query.IDsFrom(getIDsParam(c))
	if len(ids) == 0 {
		response.Fail(c, "请选择要删除的数据")
		return
	}
	now := model.Now()
	store.DB.Model(&model.ConfigGroup{}).Where("id IN ?", ids).Update("delete_time", now)
	cache.C.ClearTag(svc.ConfigCacheTag)
	response.Success(c, "删除成功")
}

// ---------- 邮件记录 ----------

type MailHandler struct{}

func NewMailHandler() *MailHandler { return &MailHandler{} }

func (h *MailHandler) Index(c *gin.Context) {
	p := query.ParseParams(c)
	db := store.DB.Model(&model.Mail{}).Where("delete_time IS NULL")
	if email := c.Query("email"); email != "" {
		db = db.Where("email LIKE ?", "%"+email+"%")
	}
	db = p.Apply(db, "id")
	var list []model.Mail
	response.Success(c, p.Paginate(db, &list))
}

func (h *MailHandler) Read(c *gin.Context) {
	var m model.Mail
	if err := store.DB.Where("id = ?", getIDParam(c)).First(&m).Error; err != nil {
		response.Fail(c, "数据不存在")
		return
	}
	response.Success(c, m)
}

func (h *MailHandler) Destroy(c *gin.Context) {
	ids := query.IDsFrom(getIDsParam(c))
	if len(ids) == 0 {
		response.Fail(c, "参数错误，请检查")
		return
	}
	now := model.Now()
	store.DB.Model(&model.Mail{}).Where("id IN ?", ids).Update("delete_time", now)
	response.Success(c, "操作成功")
}
