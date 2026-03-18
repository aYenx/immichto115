# ImmichTo115 前端

这里是 ImmichTo115 的前端目录，基于 Vue 3 + Vite + TypeScript，主要负责：

- 首次启动的 Setup Wizard
- Dashboard、Settings、Photo Upload、Restore Explorer 等页面
- 与后端 API / WebSocket 的交互

## 常用命令

```bash
cd web
npm ci
npm run dev
npm run typecheck
npm run build
```

## 本地开发

- `npm run dev` 会启动 Vite 开发服务器
- `/api` 会代理到 `http://localhost:8096`
- `/ws` 会代理到 `ws://localhost:8096`

后端本地启动方式、单文件构建方式和完整项目说明，请查看根目录 [README.md](../README.md#开发说明)。
