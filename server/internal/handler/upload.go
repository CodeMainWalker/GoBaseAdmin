package handler

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CodeMainWalker/GoBaseAdmin/internal/menu"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/model"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/query"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/response"
)

// uploadConfig 上传配置（取自 sa_system_config 的 upload_config 分组）
type uploadConfig struct {
	AllowFile  []string
	AllowImage []string
	MaxSize    int64
	Root       string // public/storage/
	Domain     string // http://127.0.0.1:8787
	URI        string // /storage/
	Mode       int
}

// loadUploadConfig 读取上传配置（带缓存，缓存时长沿用上游的一年 TTL）
func loadUploadConfig() uploadConfig {
	cfg := uploadConfig{
		AllowFile:  []string{"txt", "doc", "docx", "xls", "xlsx", "ppt", "pptx", "rar", "zip", "7z", "gz", "pdf", "wps", "md", "jpg", "png", "jpeg", "mp4", "pem", "crt"},
		AllowImage: []string{"jpg", "jpeg", "png", "gif", "svg", "bmp"},
		MaxSize:    52428800,
		Root:       "public/storage/",
		Domain:     "http://127.0.0.1:8787",
		URI:        "/storage/",
		Mode:       1,
	}

	values := configGroup("upload_config")
	if v, ok := values["upload_allow_file"]; ok && v != "" {
		cfg.AllowFile = splitCSV(v)
	}
	if v, ok := values["upload_allow_image"]; ok && v != "" {
		cfg.AllowImage = splitCSV(v)
	}
	if v, ok := values["upload_size"]; ok && v != "" {
		if n := toInt(v); n > 0 {
			cfg.MaxSize = int64(n)
		}
	}
	if v, ok := values["local_root"]; ok && v != "" {
		cfg.Root = v
	}
	if v, ok := values["local_domain"]; ok && v != "" {
		cfg.Domain = v
	}
	if v, ok := values["local_uri"]; ok && v != "" {
		cfg.URI = v
	}
	if v, ok := values["upload_mode"]; ok && v != "" {
		cfg.Mode = toInt(v)
	}
	return cfg
}

// configGroup 读取配置分组，返回 key→value
func configGroup(code string) map[string]string {
	type row struct {
		Key   string `gorm:"column:key"`
		Value string `gorm:"column:value"`
	}
	var rows []row
	storeRaw(`SELECT c.`+"`key`"+` AS `+"`key`"+`, c.value FROM sa_system_config c
		JOIN sa_system_config_group g ON g.id = c.group_id
		WHERE g.code = ? AND c.delete_time IS NULL AND g.delete_time IS NULL`, code).Scan(&rows)

	out := map[string]string{}
	for _, r := range rows {
		out[r.Key] = r.Value
	}
	return out
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func containsStr(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// formatBytes 把字节数格式化为 "1.00 MB" 形式（与上游 SaiAdmin 6.x 一致）
func formatBytes(b int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	f := float64(b)
	i := 0
	for f > 1024 && i < len(units)-1 {
		f /= 1024
		i++
	}
	return fmt.Sprintf("%.2f %s", f, units[i])
}

// saveUploadedFile 落盘并返回存储信息。
// 命名策略与上游 SaiAdmin 6.x 的本地适配器一致：sha1(内容) + '.' + 扩展名，按 Ymd 分目录。
func saveUploadedFile(fh *multipart.FileHeader, cfg uploadConfig) (map[string]string, error) {
	src, err := fh.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	h := sha1.New()
	if _, err := io.Copy(h, src); err != nil {
		return nil, err
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(fh.Filename)), ".")
	sum := hex.EncodeToString(h.Sum(nil))
	objectName := sum + "." + ext

	dirName := time.Now().Format("20060102")
	root := strings.TrimRight(cfg.Root, "/\\")
	dir := filepath.Join(root, dirName)
	if err := os.MkdirAll(dir, 0o777); err != nil {
		return nil, err
	}

	dstPath := filepath.Join(dir, objectName)
	dst, err := os.Create(dstPath)
	if err != nil {
		return nil, err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return nil, err
	}

	// URL = domain + uri(去尾斜杠) + '/' + Ymd + '/' + 文件名
	url := strings.TrimRight(cfg.Domain, "/") +
		strings.TrimRight(cfg.URI, "/") + "/" + dirName + "/" + objectName

	return map[string]string{
		"origin_name":  fh.Filename,
		"object_name":  objectName,
		"hash":         sum,
		"storage_path": filepath.ToSlash(dstPath),
		"url":          url,
		"ext":          ext,
	}, nil
}

// insertAttachment 写入附件表并返回响应体（与上游 SaiAdmin 6.x 一致：返回体不含 id）
func insertAttachment(meta map[string]string, categoryID, mode int, mime string, size int64) gin.H {
	info := gin.H{
		"storage_mode": mode,
		"category_id":  categoryID,
		"origin_name":  meta["origin_name"],
		"object_name":  meta["object_name"],
		"hash":         meta["hash"],
		"mime_type":    mime,
		"storage_path": meta["storage_path"],
		"suffix":       meta["ext"],
		"size_byte":    size,
		"size_info":    formatBytes(size),
		"url":          meta["url"],
	}
	rec := model.Attachment{
		CategoryID:  categoryID,
		StorageMode: mode,
		OriginName:  meta["origin_name"],
		ObjectName:  meta["object_name"],
		Hash:        meta["hash"],
		MimeType:    mime,
		StoragePath: meta["storage_path"],
		Suffix:      meta["ext"],
		SizeByte:    size,
		SizeInfo:    formatBytes(size),
		URL:         meta["url"],
	}
	_ = storeDB().Create(&rec).Error
	return info
}

// ---------- 上传接口 ----------

type UploadHandler struct{}

func NewUploadHandler() *UploadHandler { return &UploadHandler{} }

// UploadImage POST /core/system/uploadImage
func (h *UploadHandler) UploadImage(c *gin.Context) {
	h.upload(c, "image")
}

// UploadFile POST /core/system/uploadFile
func (h *UploadHandler) UploadFile(c *gin.Context) {
	h.upload(c, "file")
}

func (h *UploadHandler) upload(c *gin.Context, kind string) {
	cfg := loadUploadConfig()

	fh, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, "请选择上传文件")
		return
	}
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(fh.Filename)), ".")
	// 白名单比较：上游按扩展名精确匹配（大小写敏感），这里已统一为小写以便宽容处理
	allow := cfg.AllowFile
	if kind == "image" {
		allow = cfg.AllowImage
	}
	if !containsStr(allow, ext) {
		response.Fail(c, "不支持该格式的文件上传")
		return
	}
	if fh.Size > cfg.MaxSize {
		response.Fail(c, "文件大小超过限制")
		return
	}

	mode := cfg.Mode
	if c.Query("mode") == "local" || c.PostForm("mode") == "local" {
		mode = 1
	}

	meta, err := saveUploadedFile(fh, cfg)
	if err != nil {
		response.Fail(c, "上传失败："+err.Error())
		return
	}
	categoryID := toInt(c.Query("category_id"))
	if categoryID == 0 {
		categoryID = toInt(c.PostForm("category_id"))
	}
	if categoryID == 0 {
		categoryID = 1
	}

	// MIME 检测：优先信任客户端声明，但其常为 application/octet-stream；
	// 上游由扩展名推导，这里同样以扩展名为主、客户端声明为辅，保证与
	// getResourceList 的 mime_type 过滤条件（image/jpeg 等）一致。
	mimeType := detectMime(meta["ext"], fh.Header.Get("Content-Type"))

	response.Success(c, insertAttachment(meta, categoryID, mode, mimeType, fh.Size))
}

// detectMime 解析 MIME：客户端值有效则采用，否则按扩展名映射。
func detectMime(ext, declared string) string {
	if declared != "" && declared != "application/octet-stream" {
		return declared
	}
	if m, ok := extMime[strings.ToLower(ext)]; ok {
		return m
	}
	if declared != "" {
		return declared
	}
	return "application/octet-stream"
}

// extMime 常见扩展名到 MIME 的映射（与上游 SaiAdmin 6.x 的由后缀推导方式一致）
var extMime = map[string]string{
	"jpg": "image/jpeg", "jpeg": "image/jpeg", "png": "image/png",
	"gif": "image/gif", "webp": "image/webp", "bmp": "image/bmp",
	"svg": "image/svg+xml", "ico": "image/x-icon",
	"txt": "text/plain", "md": "text/markdown", "csv": "text/csv",
	"pdf": "application/pdf", "zip": "application/zip",
	"doc":  "application/msword",
	"docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	"xls":  "application/vnd.ms-excel",
	"xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	"ppt":  "application/vnd.ms-powerpoint",
	"pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	"mp4":  "video/mp4", "json": "application/json",
}

// ChunkUpload POST /core/system/chunkUpload
// 参数（body）：index / hash / total / ext / name / type / size
func (h *UploadHandler) ChunkUpload(c *gin.Context) {
	cfg := loadUploadConfig()
	_ = c.Request.ParseMultipartForm(64 << 20)

	index := toInt(c.PostForm("index"))
	hash := c.PostForm("hash")
	total := toInt(c.PostForm("total"))
	ext := c.PostForm("ext")
	name := c.PostForm("name")
	mimeType := c.PostForm("type")
	size := int64(toInt(c.PostForm("size")))

	if hash == "" || total <= 0 || ext == "" {
		response.Fail(c, "参数错误")
		return
	}
	// 防路径穿越
	if strings.ContainsAny(hash, `/\..`) {
		response.Fail(c, "参数错误")
		return
	}
	if !containsStr(cfg.AllowFile, ext) {
		response.Fail(c, "不支持该格式的文件上传")
		return
	}

	chunkDir := filepath.Join(strings.TrimRight(cfg.Root, "/\\"), "chunk")
	if err := os.MkdirAll(chunkDir, 0o777); err != nil {
		response.Fail(c, "创建目录失败")
		return
	}

	// index==0：先去重，再看是否需要续传
	if index == 0 {
		var existing model.Attachment
		if err := storeDB().Where("hash = ? AND delete_time IS NULL", hash).First(&existing).Error; err == nil {
			response.Success(c, gin.H{
				"storage_mode": existing.StorageMode, "category_id": existing.CategoryID,
				"origin_name": existing.OriginName, "object_name": existing.ObjectName,
				"hash": existing.Hash, "mime_type": existing.MimeType,
				"storage_path": existing.StoragePath, "suffix": existing.Suffix,
				"size_byte": existing.SizeByte, "size_info": existing.SizeInfo,
				"url": existing.URL, "id": existing.ID,
			})
			return
		}
		// 检查已存在的分片
		for i := 0; i < total; i++ {
			p := filepath.Join(chunkDir, fmt.Sprintf("%s_%d_%d.chunk", hash, total, i))
			if !fileExists(p) {
				if i == 0 {
					break // 尚无分片，走正常写入
				}
				response.Success(c, gin.H{"chunk": i, "status": "resume"})
				return
			}
		}
		// 全部存在则直接合并
		if allChunksExist(chunkDir, hash, total) {
			response.Success(c, mergeChunks(chunkDir, hash, total, ext, name, mimeType, size, cfg))
			return
		}
	}

	// 写入当前分片
	fh, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, "请选择上传文件")
		return
	}
	chunkPath := filepath.Join(chunkDir, fmt.Sprintf("%s_%d_%d.chunk", hash, total, index))
	if err := c.SaveUploadedFile(fh, chunkPath); err != nil {
		response.Fail(c, "分片保存失败")
		return
	}

	// index+1 == total 时触发合并
	if index+1 == total {
		response.Success(c, mergeChunks(chunkDir, hash, total, ext, name, mimeType, size, cfg))
		return
	}
	response.Success(c, gin.H{"chunk": index, "status": "success"})
}

func allChunksExist(dir, hash string, total int) bool {
	for i := 0; i < total; i++ {
		if !fileExists(filepath.Join(dir, fmt.Sprintf("%s_%d_%d.chunk", hash, total, i))) {
			return false
		}
	}
	return true
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// mergeChunks 合并分片并入库。
// 合并结果位于 public/storage/chunk/，因此 URL 含 chunk/ 段（沿用上游的存放约定）。
func mergeChunks(dir, hash string, total int, ext, name, mimeType string, size int64, cfg uploadConfig) gin.H {
	objectName := hash + "." + ext
	outPath := filepath.Join(dir, objectName)

	out, err := os.Create(outPath)
	if err != nil {
		return gin.H{"status": "error", "message": "合并失败"}
	}
	for i := 0; i < total; i++ {
		p := filepath.Join(dir, fmt.Sprintf("%s_%d_%d.chunk", hash, total, i))
		data, err := os.ReadFile(p)
		if err != nil {
			out.Close()
			return gin.H{"status": "error", "message": "切片文件查找失败，请重新上传"}
		}
		if _, err := out.Write(data); err != nil {
			out.Close()
			return gin.H{"status": "error", "message": "合并失败"}
		}
		_ = os.Remove(p)
	}
	out.Close()

	url := strings.TrimRight(cfg.Domain, "/") + strings.TrimRight(cfg.URI, "/") + "/chunk/" + objectName

	info := gin.H{
		"storage_mode": 1,
		"category_id":  1,
		"origin_name":  name,
		"object_name":  objectName,
		"hash":         hash,
		"mime_type":    mimeType,
		"storage_path": filepath.ToSlash(outPath),
		"suffix":       ext,
		"size_byte":    size,
		"size_info":    formatBytes(size),
		"url":          url,
	}
	rec := model.Attachment{
		CategoryID: 1, StorageMode: 1, OriginName: name, ObjectName: objectName,
		Hash: hash, MimeType: mimeType, StoragePath: filepath.ToSlash(outPath),
		Suffix: ext, SizeByte: size, SizeInfo: formatBytes(size), URL: url,
	}
	_ = storeDB().Create(&rec).Error
	return info
}

// SaveNetworkImage POST /core/system/saveNetworkImage
func (h *UploadHandler) SaveNetworkImage(c *gin.Context) {
	cfg := loadUploadConfig()
	url := c.PostForm("url")
	if url == "" {
		url = c.Query("url")
	}
	if url == "" {
		response.Fail(c, "参数错误")
		return
	}

	resp, err := http.Get(url)
	if err != nil {
		response.Fail(c, "获取文件资源失败")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		response.Fail(c, "获取文件资源失败")
		return
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, cfg.MaxSize+1))
	if err != nil {
		response.Fail(c, "获取文件资源失败")
		return
	}
	if int64(len(data)) > cfg.MaxSize {
		response.Fail(c, "文件大小超过限制")
		return
	}

	// 扩展名取自 URL 后缀，仅允许 jpg/jpeg/png/gif（与上游 SaiAdmin 6.x 一致）
	base := filepath.Base(strings.Split(url, "?")[0])
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(base)), ".")
	if !containsStr([]string{"jpg", "jpeg", "png", "gif"}, ext) {
		response.Fail(c, "文件格式错误")
		return
	}

	// MD5 去重（与上游 SaiAdmin 6.x 一致）
	sum := md5Hex(data)
	var existing model.Attachment
	if err := storeDB().Where("hash = ? AND delete_time IS NULL", sum).First(&existing).Error; err == nil {
		response.SuccessMsg(c, gin.H{
			"storage_mode": existing.StorageMode, "category_id": existing.CategoryID,
			"origin_name": existing.OriginName, "object_name": existing.ObjectName,
			"hash": existing.Hash, "mime_type": existing.MimeType,
			"storage_path": existing.StoragePath, "suffix": existing.Suffix,
			"size_byte": existing.SizeByte, "size_info": existing.SizeInfo,
			"url": existing.URL,
		}, "操作成功")
		return
	}

	dirName := time.Now().Format("20060102")
	root := strings.TrimRight(cfg.Root, "/\\")
	dir := filepath.Join(root, dirName)
	if err := os.MkdirAll(dir, 0o777); err != nil {
		response.Fail(c, "创建目录失败")
		return
	}
	// 命名：秒级时间戳(8 位十六进制) + 随机数(4 位十六进制) + 扩展名，
	// 沿用上游 SaiAdmin 6.x 生成的对象名形态。
	objectName := fmt.Sprintf("%08x%04x.%s", time.Now().Unix(), randUint16(), ext)
	dstPath := filepath.Join(dir, objectName)
	if err := os.WriteFile(dstPath, data, 0o666); err != nil {
		response.Fail(c, "写入文件失败")
		return
	}

	mime := detectMime(ext, "")
	urlOut := strings.TrimRight(cfg.Domain, "/") + strings.TrimRight(cfg.URI, "/") + "/" + dirName + "/" + objectName

	categoryID := toInt(c.PostForm("category_id"))
	if categoryID == 0 {
		categoryID = 1
	}

	meta := map[string]string{
		"origin_name": base, "object_name": objectName, "hash": sum,
		"storage_path": filepath.ToSlash(dstPath), "url": urlOut, "ext": ext,
	}
	response.SuccessMsg(c, insertAttachment(meta, categoryID, 1, mime, int64(len(data))), "操作成功")
}

// ResourceCategory GET /core/system/getResourceCategory
func (h *UploadHandler) ResourceCategory(c *gin.Context) {
	var raw []map[string]interface{}
	storeRaw(`SELECT id, parent_id, level, category_name, sort, status, remark
		FROM sa_system_category WHERE delete_time IS NULL ORDER BY sort DESC`).Scan(&raw)
	rows := normalizeRawRows(raw, "create_time", "update_time", "delete_time")
	response.Success(c, menu.MakeTree(rows, "id", "parent_id"))
}

// ResourceList GET /core/system/getResourceList —— 仅图片类型
func (h *UploadHandler) ResourceList(c *gin.Context) {
	p := query.ParseParams(c)
	db := storeDB().Model(&model.Attachment{}).Where("delete_time IS NULL").
		Where("mime_type IN ?", []string{"image/jpeg", "image/png", "image/gif", "image/webp"})
	if v := c.Query("origin_name"); v != "" {
		db = db.Where("origin_name LIKE ?", "%"+v+"%")
	}
	if v := c.Query("category_id"); v != "" {
		db = db.Where("category_id = ?", v)
	}
	db = p.Apply(db, "id")
	var list []model.Attachment
	response.Success(c, p.Paginate(db, &list))
}
