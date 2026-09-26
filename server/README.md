# GoBaseAdmin 后端（Go + Gin）

用 **Go 1.25 + Gin + GORM** 重写的后台管理服务端，接口契约与上游 SaiAdmin 6.x 完全一致，
命令行入口基于 `github.com/spf13/cobra`。

- **前端零改动**：`web` 直接复用，不需要改任何代码
- **数据库结构不变**：表结构、字段名、JSON 键名、权限标识都沿用上游
- **已排除范围**：代码生成（`tool/code`）、定时任务（`tool/crontab`）、插件市场
- **安装方式**：命令行安装（原网页安装向导 `/core/install` 已移除）

---

## 快速开始

```powershell
cd server
go build -o gobaseadmin.exe .

# 首次部署：建库 + 建表灌数据 + 生成 .env（含随机 JWT 密钥）
.\gobaseadmin.exe install --database gobaseadmin --username root --password 123456

# 启动服务（Ctrl+C 优雅退出）
.\gobaseadmin.exe serve
```

启动后监听 `http://127.0.0.1:8080`（`GO_SERVER_PORT` 可改）。
**无参数运行只打印帮助**，不会隐式启动服务。

## 命令一览

| 命令 | 说明 |
|---|---|
| `serve` | 启动 HTTP 服务；收到 Ctrl+C / SIGTERM 后 10 秒内优雅关闭 |
| `install` | 命令行安装：校验参数 → 建库 → 建表灌数据 → 写 `.env` |
| `secret` | 生成一对 JWT 密钥；`--write` 直接写入 `.env` |
| `version` | 打印版本（`--version` 等效） |
| `config` | 打印当前生效的配置（密码与密钥脱敏），用于确认到底读了哪个 `.env` |
| `--env-file <path>` | 全局参数：指定 `.env` 路径（等价于环境变量 `GO_ENV_FILE`） |

配置读取顺序（先命中者生效）：

1. `--env-file` / `GO_ENV_FILE` 指定的文件 —— 指定后**不再回退**，文件不存在直接报错
2. 当前工作目录的 `.env`（即 `server/.env`，由 `install` 生成）
3. 可执行文件同目录的 `.env`
4. 环境变量 / 内置默认值

可用的环境变量：

| 变量 | 说明 | 默认 |
|---|---|---|
| `DB_TYPE` / `DB_HOST` / `DB_PORT` | 数据库连接 | mysql / 127.0.0.1 / 3306 |
| `DB_NAME` / `DB_USER` / `DB_PASSWORD` | 库与账号 | — / — / — |
| `DB_PREFIX` / `DB_CHARSET` | 表前缀与字符集 | 空 / utf8mb4 |
| `CACHE_MODE` / `CAPTCHA_MODE` | 缓存与验证码方式 | file / cache |
| `REDIS_HOST` / `REDIS_PORT` / `REDIS_PASSWORD` / `REDIS_DB` | Redis 连接（预留） | 127.0.0.1 / 6379 / 空 / 0 |
| `GO_SERVER_PORT` | 服务监听端口 | `8080` |
| `JWT_ACCESS_SECRET` | access token 签名密钥（**必填**） | 无，缺失则拒绝启动 |
| `JWT_REFRESH_SECRET` | refresh token 签名密钥（**必填**） | 无，缺失则拒绝启动 |

### JWT 密钥

两个密钥都必须在 `.env` 中显式配置，**没有内置默认值**：缺失时 `serve` 直接报错并退出，
而不是偷偷用弱密钥把服务跑起来。

```powershell
# 生成并写入 .env（已有的 JWT_* 行会被就地替换，其它配置不动）
.\gobaseadmin.exe secret --write

# 只生成、自己决定写哪里
.\gobaseadmin.exe secret
```

`install` 会自动生成一对随机密钥（32 字节熵的 64 位十六进制串）写进 `.env`，
所以走安装流程的部署不需要手动执行 `secret`。
**access 与 refresh 必须使用不同的密钥**，这也是 `serve` 只校验「是否为空」之外要人工注意的点。

---

## 命令行安装（`gobaseadmin install`）

首次部署（数据库里还没有 `sa_` 表）时执行：

```powershell
cd server
.\gobaseadmin.exe install `
  --host 127.0.0.1 --port 3306 `
  --database gobaseadmin --username root --password 123456 `
  --data-type pure
```

| 参数 | 默认值 | 说明 |
|---|---|---|
| `--database` | 必填 | 库名，只允许字母、数字、下划线；不存在时自动 `CREATE DATABASE` |
| `--host` / `--port` | `127.0.0.1` / `3306` | 数据库地址与端口 |
| `--username` / `--password` | `root` / 空 | 数据库账号；密码会明文出现在命令行，生产环境请改用已有的 `.env` |
| `--db-type` | `mysql` | 当前仅支持 MySQL |
| `--data-type` | `pure` | `pure` = 仅基础数据 + 管理员；`demo` = 另含演示数据（部门/岗位/用户等） |

成功输出：

```
安装成功
  数据库: 127.0.0.1:3306/gobaseadmin
  配置文件: <当前工作目录>\.env（已随机生成 JWT 密钥）
  后台账号: admin / 123456
下一步: 执行 gobaseadmin serve 启动服务
```

`.env` 写在**当前工作目录**（即 `config.LocalEnvPath()`），所以请在 `server/` 下执行；
也可用 `--env-file` 指定别处。安装过程会整包执行内置 SQL dump，因此安装期间自动关掉
逐条 SQL 日志（`SAI_SQL_LOG=off`），避免几百行噪音。

安装失败的判定与提示文案保持与上游 SaiAdmin 6.x 一致（失败时退出码为 1）：

| 场景 | 表现 |
|---|---|
| `.env` 已存在 | `管理后台已经安装！如需重新安装，请删除根目录env配置文件并重启` |
| 库中已有 `sa_system_menu` | `数据库已经安装，请勿重复安装` |
| 工作目录不可写 | `权限认证失败` |
| 库名含非法字符 | `数据库名只能包含字母、数字和下划线` |
| 地址或用户名为空 | `数据库地址和用户名不能为空` |
| 认证 / 连接类错误 | 数据库用户名或密码错误、`Connection refused. 请确认数据库IP端口是否正确，数据库已经启动`、连接超时、无建库权限 |

未安装期间（`store.Ready()` 为 false，且探测不到 `sa_system_menu`），所有请求由
`middleware.InstallGuard` 统一拦截，返回
`{"code":400,"message":"系统尚未安装，请先执行 gobaseadmin install 完成安装，完成后重启服务"}`。
网页安装向导连同它的 302 跳转、静态资源与免登录白名单都已移除，不再有任何例外路径。
该中间件必须是**最早注册的全局中间件** —— 未安装时 `store.DB` 为 `nil`，
任何先执行且依赖数据库的中间件都会空指针 panic。同理，`serve` 在数据库连不上时**不退出**。

实现上的取舍：

- **仅支持 MySQL**。`--db-type` 传其它值直接返回「当前仅支持 MySQL 数据库」。
- 不依赖迁移框架，直接执行内置 SQL dump
  （`internal/install/sql/saiadmin-pure.sql` / `saiadmin-demo.sql`，含建表与初始数据，
  包括 `admin` 账号）。dump 已确认不含 `DELIMITER`／存储过程／触发器／`DEFINER`，
  可整包执行（依赖 DSN 的 `multiStatements=true`）。
- 配置文件只有一个 `.env`（上游会生成多个配置文件，Go 版合并为一处）。
- `install` 与 `serve` 是两个独立命令，安装结束后直接 `serve` 即可，不需要热重载。
- 安装数据**不含「工具」与「插件市场」**：`sa_tool_*` 四张表、
  `工具/代码生成/定时任务/插件市场` 菜单（id 80–88）及其按钮均已从 dump 中移除。
  `pure` = 18 张表 / 76 条菜单；`demo` = 21 张（多 3 张 `sa_article*`）/ 76 条菜单。
- 端到端验收脚本（需真实 MySQL，未设 `SAI_E2E=1` 时自动 skip）：
  - `go test ./internal/install -run TestInstallEndToEnd -v` —— `pure` 版初始数据
  - `go test ./internal/install -run TestInstallDemoEndToEnd -v` —— `demo` 版初始数据
    （默认库名 `SAI_TEST_DB_NAME_DEMO`，与上面区分开）

  两者都用 `SAI_TEST_DB_HOST/PORT/USER/PASSWORD` 连库，会**清空并最终删除**指定的临时库，
  所以别指向有数据的库；账号没有建库权限且库也不存在时会自动 skip 并打印需要的授权 SQL。
  想在测试/搬数据时关掉逐条 SQL 日志，设 `SAI_SQL_LOG=off`。

已在 MySQL 5.7 上实测通过：

| 验证项 | 结果 |
|---|---|
| 无参数执行 | 只打印帮助，退出码 0；未知子命令退出码 1 |
| 缺少 JWT 密钥时 `serve` | 报错退出并提示 `gobaseadmin secret --write`（退出码 1） |
| `install`（库不存在） | 自动 `CREATE DATABASE` → 建表灌数据 → 写 `.env`，输出「安装成功」，退出码 0 |
| `.env` 已存在时再 `install` | `管理后台已经安装！如需重新安装，请删除根目录env配置文件并重启`，退出码 1 |
| 删掉 `.env` 再 `install` | `数据库已经安装，请勿重复安装`，退出码 1 |
| `install --data-type demo` | 21 张表（多 3 张 `sa_article*`）/ 76 条菜单 / `sa_article` 8 行，退出码 0 |
| 用安装生成的 `.env` 起服务 | `admin/123456` 登录成功，`roles=["super_admin"]`、`buttons=["*"]`，4 个顶级菜单 |
| `GET /core/install` | 已移除，返回 404 |
| Ctrl+C（`CTRL_BREAK`） | 打印「收到退出信号，正在关闭服务…」「服务已退出」，端口释放 |

---

## 必须遵守的接口契约

这些是与上游 SaiAdmin 6.x 逐条核对出来的行为，改动会直接导致前端白屏或功能失效：

### 1. 响应信封 —— HTTP 状态码恒为 200

**所有**响应（含错误、未登录、服务器异常）HTTP 状态码都是 `200`，
业务状态放在 body 的 `code` 字段。前端 axios 拦截器只判断 `body.code === 200`。

```jsonc
// 成功
{"code": 200, "message": "success", "data": {...}}
// 失败（注意：没有 data 字段）
{"code": 400, "message": "权限不足，无法访问或操作"}
// 未登录
{"code": 401, "message": "您的登录凭证错误或者已过期，请重新登录"}
// 未捕获异常
{"code": 500, "message": "Server internal error", "type": "failed"}
```

> 若返回真正的 HTTP 401/403/500，前端不会触发既有的登出逻辑，行为将与上游不一致。

### 2. 分页 / 全量

- **请求**：`page` / `limit` / `orderField` / `orderType`（默认 `ASC`）/ `saiType`（`list` | `all`）
- **`saiType=list`（默认）**：响应字段名沿用上游 `Paginator::toArray()`

```jsonc
{"total": 8, "per_page": 10, "current_page": 1, "last_page": 1, "data": [], "has_more": false}
```

- **`saiType=all`：直接返回数组**（对应上游 `BaseLogic::getList()` 的
  `$query->select()->toArray()`），**不是**分页对象。前端会把它当数组用，
  例如 `system/config` 的 `api.configList({saiType:'all'}).then(data => data.map(...))`；
  若这里返回分页对象，页面会因 `data.map is not a function` 而一片空白。
- 两条分支的空结果都必须是 `[]` 而不是 `null`（见 `query.nonNilSlice`）。

### 3. JSON 列

`sa_system_config.config_select_data` 在库里是 JSON 文本，**接口对外必须是数组**：

```jsonc
{"key": "upload_mode", "input_type": "select",
 "config_select_data": [{"label": "本地上传", "value": "1"}, {"label": "阿里云OSS", "value": "2"}]}
```

原因：Element Plus 的 `el-select` / `el-radio-group` 的 `options` 只接受 `Array`，
配置项编辑弹窗还会对它 `v-for` / `push`。空值为 `[]`，非法内容也退化为 `[]`（不报错）。
写库时数组会被编码回 JSON 文本，空数组写空串（与库中既有数据一致）。
实现见 `internal/model/jsonfields.go`（`JSONList`：`Scan` / `Value` / `MarshalJSON` / `UnmarshalJSON`）。

### 4. 菜单树（前端动态路由的数据源）

由 `internal/menu/menu.go` 的 `MakeArtdMenus` 生成，规则：

| 菜单 `type` | 含义 | `component` 取值 |
|---|---|---|
| 1 | 目录 | 固定 `/index/index` |
| 2 | 菜单 | 取该行 `component` 列 |
| 4 | 外链 | `path` 强制改写为 `/outside/Iframe`，`meta.link` = `link_url` |

节点字段：`{id, parent_id, name, path, component, meta{title, icon, isIframe, keepAlive, isHide, fixedTab, isFullPage, link?}, children?}`
其中 `name` 取 `code` 列，`meta.title` 取 `name` 列。

### 5. JWT

- 算法 **HS256**，密钥来自配置项 `JWT_ACCESS_SECRET` / `JWT_REFRESH_SECRET`（access 与 refresh 必须不同）。
- 业务声明**全部嵌套在 `extend` 内**，没有顶层 `id`/`username`：

```jsonc
{"iss":"webman.tinywan.cn","aud":"webman.tinywan.cn","iat":...,"nbf":...,"exp":...,
 "extend":{"access_exp":28800,"id":1,"username":"admin","type":"pc","plat":"saiadmin"}}
```

- `iss`/`aud` 沿用上游的固定字面量 `webman.tinywan.cn`，改动会使已签发的令牌立即失效。
- access 有效期 **28800 秒（8 小时）** —— 与上游契约一致（上游由配置项
  `saiadmin.access_exp` 决定，Go 版固定为常量 `auth.AccessExp`）。
- `exp`/`nbf` 允许 **60 秒**时钟偏差。
- `Authorization` 头解析比常规严格：恰好一个空格、恰好两个字面 `Bearer`
  （大小写敏感），且拒绝字面量 `undefined`。
- refresh token 会签发但**前端不使用**（上游也没有 refresh 接口）。

### 6. 密码

库中现存的是上游产出的 **`$2y$` bcrypt** 哈希。
Go 的 `golang.org/x/crypto/bcrypt` 可直接校验 `$2y$`，
因此**无需迁移任何密码数据**，老用户可直接登录。
新建/改密时会把 Go 产出的 `$2a$` 前缀改写为 `$2y$` 以保持一致。

### 7. 权限

- 上游用 `#[Permission(title, slug)]` 注解 + 反射；Go 改为**路由注册时显式声明**：

```go
FastRoute(g, CRUD{
    Name: "user",
    Index: user.Index,
    IndexPerm: Perm{"用户数据列表", "core:user:index"},  // title 用于操作日志，slug 用于鉴权
})
```

- `slug` 为空的接口**仅校验登录**，不做权限判断（与上游一致）。
- 用户 `id == 1` 为超级管理员，跳过全部权限与数据权限检查。
- **授权必须在路由级中间件做**（见 `permMiddleware`）。Gin 的 group 中间件
  先于路由级中间件执行，若在 group 级读取 slug，此时 slug 尚未写入，
  会导致校验被整体跳过 —— 这是一个已修复的真实缺陷。

### 8. 操作日志

- **GET 请求不记录**，只记录 POST/PUT/DELETE 等（与上游一致）。
- `service_name` 取路由上声明的权限 `title`，缺省为 `未命名业务`。
- `request_data` 对 `password` / `oldPassword` / `newPassword` / `confirmPassword` 脱敏为 `******`。
- `router` 含查询串；`ip_location` 对内外网做区分，其余为 `未知`
  （不引入 ip2region 离线库数据文件）。

### 9. 其他易错点

| 场景 | 处理 |
|---|---|
| DELETE 携带 body | 前端 `delete()` 把参数放在 JSON body，Gin 默认不解析，需显式读取 `ids` |
| 软删除与唯一索引 | `uk_username` / `uk_slug` **不区分软删除**。创建前会先物理清除同名/同标识的已删除残留行，否则插入必然失败 |
| 时间格式 | 统一 `Y-m-d H:i:s` 字符串（见 `model.DateTime`），不能用 Go 默认的 RFC3339 |
| `level` 前缀树 | 部门/分类用 `level` 字符串（如 `0,1,`）表示层级，子节点用 `LIKE 'level' '<prefix>%'` 查询 |
| 角色越权保护 | 只能操作 `level` **低于**自身的角色（`>=` 自身即拒绝） |
| 数据权限 | 非超管仅能操作自己部门及子部门的用户（`deptSubtreeIDs`） |
| 用户批量删除 | 明确拒绝，返回 `禁止批量删除操作` |
| 超级管理员 | `id=1` 不可删除、不可重置密码 |
| 服务监控 | 主机指标（磁盘/CPU/负载）取自系统原生接口；`memory.php_version` / `cpu.go_version` 沿用上游键名（历史命名不改），值都是 Go 运行时版本 |

---

## 目录结构

```
server/
├── main.go          # 仅 11 行：os.Exit(cmd.Execute())
├── cmd/             # cobra 命令：root / serve / install / secret / version / config
├── internal/
│   ├── config/      # 配置加载（--env-file → .env → exe 同目录 → 环境变量）
│   ├── install/     # 命令行安装（内嵌 SQL dump）
│   ├── auth/        # JWT 签发/校验、bcrypt
│   ├── captcha/     # 图形验证码（纯标准库绘制，无第三方字体依赖）
│   ├── cache/       # 进程内 TTL 缓存（对应 CACHE_MODE=file）
│   ├── menu/        # makeTree / makeArtdMenus（前端路由契约核心）
│   ├── model/       # 18 张表的 GORM 模型 + DateTime 类型
│   ├── query/       # 分页与搜索（对应 BaseLogic）
│   ├── store/       # 数据库连接
│   ├── svc/         # 用户上下文与权限缓存
│   ├── sysinfo/     # 主机指标采集（磁盘 / CPU / 负载，分平台实现）
│   ├── middleware/  # 登录校验、授权、操作日志、CORS、异常兜底
│   ├── handler/     # 各控制器
│   └── router/      # 路由与权限元信息注册
└── public/storage/  # 上传落盘目录（运行时自动创建）
```

---

## 已知取舍

1. **缓存为单进程内存实现**。若需多实例部署，请将 `internal/cache` 换成 Redis
   （接口已抽象，键名沿用 `saiadmin:*` 前缀，便于与既有数据共存）。
2. **服务监控**的主机指标（磁盘、CPU、负载）取自操作系统原生接口，见下方「服务监控指标」；
   `memory.*` 与 `goroutines` 取自 Go runtime（进程口径，不是整机内存），字段名沿用上游契约以免前端报错。
3. **Excel 导入导出**（岗位）未实现，对应路由未注册；前端该功能会提示请求失败。
4. **插件市场**未实现：对应前端页面、菜单与数据表均已删除。
5. **数据表维护**的优化/碎片整理使用 MySQL 语法（`OPTIMIZE TABLE` / `ALTER TABLE ... ENGINE=InnoDB`），
   不支持 PostgreSQL。
6. 修复了上游 `saveMenuPermission` 中 `limit(100)` 导致菜单权限超过 100 条被静默截断的缺陷。

---

## 服务监控指标（`GET /core/server/monitor`）

主机指标由 `internal/sysinfo` 按平台采集，全部是真实数值，采集失败时返回占位符
（`disk.path = "-"`、`cpu.used = "-"`、`load.available = false`），不会把失败伪装成 0：

| 字段 | 含义 | 数据来源 |
|---|---|---|
| `disk.total/used/free/usage/path` | 当前工作目录所在卷 / 文件系统的容量（字节） | Windows：`GetDiskFreeSpaceEx`；Linux/macOS/FreeBSD：`statfs` |
| `cpu.usage`（`used`/`idle` 为其文本形式） | CPU 使用率百分比 | Windows：`GetSystemTimes` 两次采样差值；Linux：`/proc/stat` 两次采样差值 |
| `load.one/five/fifteen` | 1 / 5 / 15 分钟平均负载 | Linux：`/proc/loadavg`；macOS/BSD：`sysctl vm.loadavg` |
| `load.available` | 平台是否提供 load average | Windows 为 `false`：Windows 没有 load average 概念（等价指标是处理器队列长度，需要 PDH 计数器），页面上以 CPU 使用率代替 |
| `memory.total/used/free/usage`、`goroutines` | Go 进程内存与协程数 | Go runtime（`runtime.ReadMemStats` / `NumGoroutine`） |

- CPU 使用率是**两次采样之间的差值**：首次请求返回自开机以来的平均值，
  之后每次请求返回距上次请求区间内的真实占用，所以前端页面按固定间隔（5 秒）自动刷新。
- 平台支持情况：`internal/sysinfo` 用 build tag 分文件实现，已在
  `windows/amd64`、`windows/arm64`、`linux/amd64`、`linux/arm64`、`linux/386`、
  `darwin/amd64`、`darwin/arm64`、`freebsd/amd64`、`dragonfly/amd64`、`openbsd/amd64`、
  `netbsd/amd64`、`solaris/amd64`、`aix/ppc64`、`js/wasm`、`plan9/amd64` 上通过编译；
  openbsd / netbsd 的 `statfs` 字段名与其它平台不同，磁盘指标在这两个平台会显示占位符。
- 自动化验证：`go test ./internal/sysinfo -v`
  （磁盘容量为正、CPU 忙等 300ms 后能测出非零使用率、不支持负载的平台 `Available=false`）。
