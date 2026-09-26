package model

// 本文件按 db/saiadmin-pure.sql 的表结构定义全部模型。
// 字段名、JSON 键名均与上游 SaiAdmin 6.x 一致，以保证前端零改动。
// 注意：软删除用 delete_time（Base.DeleteTime），查询需显式加 ScopeNotDeleted。

// ---------- 用户 ----------

type User struct {
	Base
	Username  string    `gorm:"column:username" json:"username"`
	Password  string    `gorm:"column:password" json:"-"`
	Realname  string    `gorm:"column:realname" json:"realname"`
	Gender    string    `gorm:"column:gender" json:"gender"`
	Avatar    string    `gorm:"column:avatar" json:"avatar"`
	Email     string    `gorm:"column:email" json:"email"`
	Phone     string    `gorm:"column:phone" json:"phone"`
	Signed    string    `gorm:"column:signed" json:"signed"`
	Dashboard string    `gorm:"column:dashboard" json:"dashboard"`
	DeptID    *uint64   `gorm:"column:dept_id" json:"dept_id"`
	IsSuper   int       `gorm:"column:is_super" json:"is_super"`
	Status    int       `gorm:"column:status" json:"status"`
	Remark    string    `gorm:"column:remark" json:"remark"`
	LoginTime *DateTime `gorm:"column:login_time" json:"login_time"`
	LoginIP   string    `gorm:"column:login_ip" json:"login_ip"`
}

func (User) TableName() string { return "sa_system_user" }

// ---------- 角色 ----------

type Role struct {
	Base
	Name      string `gorm:"column:name" json:"name"`
	Code      string `gorm:"column:code" json:"code"`
	Level     int    `gorm:"column:level" json:"level"`
	DataScope int    `gorm:"column:data_scope" json:"data_scope"`
	Remark    string `gorm:"column:remark" json:"remark"`
	Sort      int    `gorm:"column:sort" json:"sort"`
	Status    int    `gorm:"column:status" json:"status"`
}

func (Role) TableName() string { return "sa_system_role" }

// ---------- 部门 ----------

type Dept struct {
	Base
	ParentID uint64  `gorm:"column:parent_id" json:"parent_id"`
	Name     string  `gorm:"column:name" json:"name"`
	Code     string  `gorm:"column:code" json:"code"`
	LeaderID *uint64 `gorm:"column:leader_id" json:"leader_id"`
	Level    string  `gorm:"column:level" json:"level"`
	Sort     int     `gorm:"column:sort" json:"sort"`
	Status   int     `gorm:"column:status" json:"status"`
	Remark   string  `gorm:"column:remark" json:"remark"`
}

func (Dept) TableName() string { return "sa_system_dept" }

// ---------- 岗位 ----------

type Post struct {
	Base
	Name   string `gorm:"column:name" json:"name"`
	Code   string `gorm:"column:code" json:"code"`
	Sort   int    `gorm:"column:sort" json:"sort"`
	Status int    `gorm:"column:status" json:"status"`
	Remark string `gorm:"column:remark" json:"remark"`
}

func (Post) TableName() string { return "sa_system_post" }

// ---------- 菜单 ----------

type Menu struct {
	Base
	ParentID    uint64 `gorm:"column:parent_id" json:"parent_id"`
	Name        string `gorm:"column:name" json:"name"`
	Code        string `gorm:"column:code" json:"code"`
	Slug        string `gorm:"column:slug" json:"slug"`
	Type        int    `gorm:"column:type" json:"type"` // 1目录 2菜单 3按钮 4外链
	Path        string `gorm:"column:path" json:"path"`
	Component   string `gorm:"column:component" json:"component"`
	Method      string `gorm:"column:method" json:"method"`
	Icon        string `gorm:"column:icon" json:"icon"`
	Sort        int    `gorm:"column:sort" json:"sort"`
	LinkURL     string `gorm:"column:link_url" json:"link_url"`
	IsIframe    int    `gorm:"column:is_iframe" json:"is_iframe"`
	IsKeepAlive int    `gorm:"column:is_keep_alive" json:"is_keep_alive"`
	IsHidden    int    `gorm:"column:is_hidden" json:"is_hidden"`
	IsFixedTab  int    `gorm:"column:is_fixed_tab" json:"is_fixed_tab"`
	IsFullPage  int    `gorm:"column:is_full_page" json:"is_full_page"`
	GenerateID  int    `gorm:"column:generate_id" json:"generate_id"`
	GenerateKey string `gorm:"column:generate_key" json:"generate_key"`
	Status      int    `gorm:"column:status" json:"status"`
	Remark      string `gorm:"column:remark" json:"remark"`
	// 注意：sa_system_menu 表**没有** level 列（与 dept 不同），
	// 上游会为此多输出一个 level 字段，Go 版不输出，对前端无影响。
}

func (Menu) TableName() string { return "sa_system_menu" }

// ---------- 字典 ----------

type DictType struct {
	Base
	Name   string `gorm:"column:name" json:"name"`
	Code   string `gorm:"column:code" json:"code"`
	Status int    `gorm:"column:status" json:"status"`
	Remark string `gorm:"column:remark" json:"remark"`
}

func (DictType) TableName() string { return "sa_system_dict_type" }

type DictData struct {
	Base
	TypeID uint   `gorm:"column:type_id" json:"type_id"`
	Label  string `gorm:"column:label" json:"label"`
	Value  string `gorm:"column:value" json:"value"`
	Color  string `gorm:"column:color" json:"color"`
	Code   string `gorm:"column:code" json:"code"`
	Sort   int    `gorm:"column:sort" json:"sort"`
	Status int    `gorm:"column:status" json:"status"`
	Remark string `gorm:"column:remark" json:"remark"`
}

func (DictData) TableName() string { return "sa_system_dict_data" }

// ---------- 配置 ----------

type ConfigGroup struct {
	Base
	Name   string `gorm:"column:name" json:"name"`
	Code   string `gorm:"column:code" json:"code"`
	Remark string `gorm:"column:remark" json:"remark"`
}

func (ConfigGroup) TableName() string { return "sa_system_config_group" }

type Config struct {
	Base
	GroupID   *uint  `gorm:"column:group_id" json:"group_id"`
	Key       string `gorm:"column:key" json:"key"`
	Value     string `gorm:"column:value" json:"value"`
	Name      string `gorm:"column:name" json:"name"`
	InputType string `gorm:"column:input_type" json:"input_type"`
	Sort      int    `gorm:"column:sort" json:"sort"`
	Remark    string `gorm:"column:remark" json:"remark"`
	// 库里是 JSON 文本，对外是数组（select / radio 的 options）
	ConfigSelectData JSONList `gorm:"column:config_select_data" json:"config_select_data"`
}

func (Config) TableName() string { return "sa_system_config" }

// ---------- 附件 ----------

type Attachment struct {
	Base
	CategoryID  int    `gorm:"column:category_id" json:"category_id"`
	StorageMode int    `gorm:"column:storage_mode" json:"storage_mode"`
	OriginName  string `gorm:"column:origin_name" json:"origin_name"`
	ObjectName  string `gorm:"column:object_name" json:"object_name"`
	Hash        string `gorm:"column:hash" json:"hash"`
	MimeType    string `gorm:"column:mime_type" json:"mime_type"`
	StoragePath string `gorm:"column:storage_path" json:"storage_path"`
	Suffix      string `gorm:"column:suffix" json:"suffix"`
	SizeByte    int64  `gorm:"column:size_byte" json:"size_byte"`
	SizeInfo    string `gorm:"column:size_info" json:"size_info"`
	URL         string `gorm:"column:url" json:"url"`
	Remark      string `gorm:"column:remark" json:"remark"`
}

func (Attachment) TableName() string { return "sa_system_attachment" }

type Category struct {
	Base
	ParentID     uint64 `gorm:"column:parent_id" json:"parent_id"`
	Level        string `gorm:"column:level" json:"level"`
	CategoryName string `gorm:"column:category_name" json:"category_name"`
	Sort         int    `gorm:"column:sort" json:"sort"`
	Status       int    `gorm:"column:status" json:"status"`
	Remark       string `gorm:"column:remark" json:"remark"`
}

func (Category) TableName() string { return "sa_system_category" }

// ---------- 邮件 ----------

type Mail struct {
	Base
	Gateway  string `gorm:"column:gateway" json:"gateway"`   // 网关
	From     string `gorm:"column:from" json:"from"`         // 发送人
	Email    string `gorm:"column:email" json:"email"`       // 接收人
	Code     string `gorm:"column:code" json:"code"`         // 验证码
	Content  string `gorm:"column:content" json:"content"`   // 邮箱内容
	Status   string `gorm:"column:status" json:"status"`     // 发送状态
	Response string `gorm:"column:response" json:"response"` // 返回结果
}

func (Mail) TableName() string { return "sa_system_mail" }

// ---------- 日志 ----------

type LoginLog struct {
	Base
	Username   string    `gorm:"column:username" json:"username"`
	IP         string    `gorm:"column:ip" json:"ip"`
	IPLocation string    `gorm:"column:ip_location" json:"ip_location"`
	OS         string    `gorm:"column:os" json:"os"`
	Browser    string    `gorm:"column:browser" json:"browser"`
	Status     int       `gorm:"column:status" json:"status"`
	Message    string    `gorm:"column:message" json:"message"`
	LoginTime  *DateTime `gorm:"column:login_time" json:"login_time"`
}

func (LoginLog) TableName() string { return "sa_system_login_log" }

type OperLog struct {
	Base
	Username    string `gorm:"column:username" json:"username"`
	App         string `gorm:"column:app" json:"app"`
	Method      string `gorm:"column:method" json:"method"`
	Router      string `gorm:"column:router" json:"router"`
	ServiceName string `gorm:"column:service_name" json:"service_name"`
	IP          string `gorm:"column:ip" json:"ip"`
	IPLocation  string `gorm:"column:ip_location" json:"ip_location"`
	RequestData string `gorm:"column:request_data" json:"request_data"`
	Remark      string `gorm:"column:remark" json:"remark"`
}

func (OperLog) TableName() string { return "sa_system_oper_log" }

// ---------- 关联表（无软删除、无时间戳） ----------

type UserRole struct {
	ID     uint64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID uint64 `gorm:"column:user_id" json:"user_id"`
	RoleID uint64 `gorm:"column:role_id" json:"role_id"`
}

func (UserRole) TableName() string { return "sa_system_user_role" }

type UserPost struct {
	ID     uint64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID uint64 `gorm:"column:user_id" json:"user_id"`
	PostID uint64 `gorm:"column:post_id" json:"post_id"`
}

func (UserPost) TableName() string { return "sa_system_user_post" }

type RoleMenu struct {
	ID     uint64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	RoleID uint64 `gorm:"column:role_id" json:"role_id"`
	MenuID uint64 `gorm:"column:menu_id" json:"menu_id"`
}

func (RoleMenu) TableName() string { return "sa_system_role_menu" }

type RoleDept struct {
	ID     uint64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	RoleID uint64 `gorm:"column:role_id" json:"role_id"`
	DeptID uint64 `gorm:"column:dept_id" json:"dept_id"`
}

func (RoleDept) TableName() string { return "sa_system_role_dept" }
