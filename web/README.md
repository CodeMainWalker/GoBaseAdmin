# GoBaseAdmin 前端

GoBaseAdmin 管理后台前端项目，采用 Vue 3 + TypeScript + Vite 开发，后端服务由 Go(Gin) 提供。

项目仓库：<https://github.com/CodeMainWalker/GoBaseAdmin>

## 技术栈

- Vue 3.5 + TypeScript 5.6
- Vite 7
- Element Plus 2.11
- Pinia 3 + Vue Router 4 + vue-i18n 9
- Tailwind CSS 4 + Sass
- ECharts 6

## 环境要求

- Node.js >= 20.19.0
- pnpm >= 8.8.0

## 本地开发

```bash
pnpm install
pnpm dev
```

开发服务器默认端口为 `3006`（可在 `.env` 中通过 `VITE_PORT` 修改）。

后端接口地址通过 `.env.development` 中的 `VITE_API_PROXY_URL` 配置，默认指向 `http://127.0.0.1:8080`；前端请求基础路径为 `/api`，开发环境由 Vite 代理转发。

## 构建与预览

```bash
pnpm build   # 先执行 vue-tsc --noEmit 类型检查，再打包到 dist/
pnpm serve   # 本地预览构建产物
```

## 说明

- 界面模板基于开源项目 [art-design-pro](https://github.com/Daymychen/art-design-pro)（MIT），其版权声明与
  `@author Art Design Pro Team` 注释均已保留，原始许可证见 [web/LICENSE](./LICENSE)。
- 本项目遵循 MIT 许可证，详见仓库根目录的 [LICENSE](../LICENSE)。
