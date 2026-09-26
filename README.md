# GoBaseAdmin

<p>
  <img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License">
  <img src="https://img.shields.io/badge/Go-1.25-00ADD8.svg" alt="Go">
  <img src="https://img.shields.io/badge/Gin-1.12-008ECF.svg" alt="Gin">
  <img src="https://img.shields.io/badge/Vue-3.5-42B883.svg" alt="Vue">
</p>

GoBaseAdmin 是一个基于 **Go + Gin + GORM** 的后台管理系统，前端使用 **Vue 3 + TypeScript + Vite + Element Plus**。

本项目的服务端是对开源项目 [SaiAdmin 6.x](https://github.com/saithink/saiadmin6.x)（PHP / Webman 实现）的
**Go 语言重写版**：数据库表结构、接口契约（响应信封、分页、JWT、权限标识）与原版保持一致，
因此前端页面无需任何改动即可直接使用。重写过程中去掉了与原版强绑定、维护成本较高的部分
（代码生成、定时任务、插件市场），保留了一套完整、可直接用于生产后台的权限与运维体系。

> 想了解服务端的实现细节与接口契约，请看 [`server/README.md`](server/README.md)；
> 前端说明见 [`web/README.md`](web/README.md)。

## 技术栈

| 层次 | 选型 |
|---|---|
| 后端语言 | Go 1.25 |
| Web 框架 | Gin 1.12 |
| ORM / 数据库 | GORM + MySQL 5.7 / 8.0 |
| 命令行 | spf13/cobra |
| 认证 | JWT（HS256）+ bcrypt |
| 前端 | Vue 3.5 / TypeScript / Vite 7 / Element Plus / Pinia / Tailwind CSS |

## 功能模块

### 仪表盘

| 模块 | 说明 |
|---|---|
| 工作台 | 数据概览、登录统计图表、快捷入口、项目信息 |

### 系统管理

| 模块 | 说明 |
|---|---|
| 用户管理 | 用户增删改查、重置密码、清理缓存、设置工作台首页 |
| 部门管理 | 组织架构树、部门负责人、数据权限范围 |
| 角色管理 | 角色增删改查、菜单权限分配、角色层级保护 |
| 岗位管理 | 岗位信息维护（Excel 导入导出未实现） |
| 菜单管理 | 目录 / 菜单 / 按钮权限 / 外链配置 |
| 系统配置 | 分组配置、站点信息、上传配置等动态参数 |

### 运维管理

| 模块 | 说明 |
|---|---|
| 缓存管理 | 缓存键查看与一键清理 |
| 数据字典 | 字典类型与字典数据维护 |
| 附件管理 | 文件上传、分类管理、资源移动、回收站 |
| 数据表维护 | 表结构查看、表优化、碎片整理、数据源与回收站 |
| 登录日志 | 登录记录查询与清理 |
| 操作日志 | 请求记录、脱敏、IP 归属地 |
| 邮件日志 | 邮件发送记录 |
| 服务监控 | CPU / 内存 / 磁盘 / 负载、Go 运行时信息、缓存与协程数 |

### 未包含的功能

以下功能在原版中存在，本项目**有意不实现**，对应的前端页面、菜单与数据表也已移除：

- **代码生成器**：按数据表生成 CRUD 代码
- **定时任务**：Crontab 任务管理与执行日志
- **插件市场**：插件安装与计费相关功能

## 快速开始

### 环境要求

- Go 1.25 或更高版本
- MySQL 5.7 / 8.0（当前仅支持 MySQL）
- Node.js >= 20.19.0、pnpm >= 8.8.0

### 1. 启动后端

```powershell
cd server
go build -o gobaseadmin.exe .

# 首次部署：自动建库 + 建表灌数据 + 生成 .env（含随机 JWT 密钥）
.\gobaseadmin.exe install --database gobaseadmin --username root --password 123456

# 启动服务，默认监听 http://127.0.0.1:8080
.\gobaseadmin.exe serve
```

安装完成后默认后台账号为 `admin / 123456`，请登录后立即修改密码。

命令行一览：

| 命令 | 说明 |
|---|---|
| `gobaseadmin serve` | 启动 HTTP 服务（Ctrl+C 优雅退出） |
| `gobaseadmin install` | 命令行安装：建库 → 建表灌数据 → 写 `.env` |
| `gobaseadmin secret` | 生成 JWT 密钥，`--write` 直接写入 `.env` |
| `gobaseadmin config` | 打印当前生效的配置（密码与密钥脱敏） |
| `gobaseadmin version` | 打印版本 |

> 无参数运行只打印帮助，不会隐式启动服务。

### 2. 启动前端

```powershell
cd web
pnpm install
pnpm dev
```

前端开发服务器默认在 `3006` 端口，通过 `web/.env.development` 的
`VITE_API_PROXY_URL` 指向后端（默认已是 `http://127.0.0.1:8080`）。
生产构建使用 `pnpm build`。

### 3. 配置说明

后端配置全部来自 `server/.env`，由 `install` 生成，也可以手工维护：

| 配置项 | 说明 | 默认值 |
|---|---|---|
| `DB_TYPE` / `DB_HOST` / `DB_PORT` | 数据库连接 | mysql / 127.0.0.1 / 3306 |
| `DB_NAME` / `DB_USER` / `DB_PASSWORD` | 库名与账号 | — |
| `DB_PREFIX` / `DB_CHARSET` | 表前缀与字符集 | 空 / utf8mb4 |
| `GO_SERVER_PORT` | 服务监听端口 | `8080` |
| `JWT_ACCESS_SECRET` | access token 签名密钥 | **必填**，无默认值 |
| `JWT_REFRESH_SECRET` | refresh token 签名密钥 | **必填**，无默认值 |
| `CACHE_MODE` / `CAPTCHA_MODE` | 缓存与验证码方式 | file / cache |

**JWT 密钥是必填项**：没有配置时 `serve` 会直接报错退出，并提示生成命令；
`install` 会自动生成一对不同的随机密钥写入 `.env`。需要轮换密钥时执行：

```powershell
cd server
.\gobaseadmin.exe secret --write   # 就地替换 .env 中的 JWT_* 两行
```

## 目录结构

```
.
├── server/            # Go 后端（Gin + GORM + cobra）
│   ├── cmd/           # 命令行入口：serve / install / secret / version / config
│   ├── internal/      # 配置、安装、认证、权限、菜单、控制器等
│   └── public/        # 上传文件落盘目录
└── web/               # Vue 3 前端
    ├── src/           # 页面、组件、状态管理、请求封装
    └── public/        # 静态资源
```

## 与上游的差异

- **语言与框架**：服务端由 PHP / Webman 改为 Go / Gin，前端保持 Vue 3 不变。
- **数据库兼容**：表结构与字段完全沿用上游，可复用上游已有数据；
  库中历史密码为 `$2y$` bcrypt 哈希，Go 可直接校验，**无需迁移密码**。
- **安装方式**：网页安装向导已移除，改为 `gobaseadmin install` 命令行安装。
- **仅支持 MySQL**：原版支持的 PostgreSQL 未实现。
- **未实现的功能**：代码生成、定时任务、插件市场（见上文）。
- **已修复的缺陷**：菜单权限保存时上游的 `limit(100)` 截断问题。

## 协议

本项目基于 [MIT 协议](LICENSE) 开源。

服务端与前端界面均重写/改造自上游开源项目 **SaiAdmin 6.x**（Copyright (c) 2026 saithink，MIT 协议），
前端 UI 模板来自 **art-design-pro**（MIT 协议）。原始版权声明一并保留在 [LICENSE](LICENSE) 中。
