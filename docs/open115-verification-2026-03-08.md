# Open115 集成验收记录（2026-03-08）

## 已完成的真实验证

- `go test ./...` 通过
- 使用真实 `access_token / refresh_token` 完成 `POST /api/v1/open115/test`
- 成功列出 115 根目录与子目录
- 使用测试目录 `/tmp/immichto115-e2e` 触发真实备份
- 远端目录 `/open115-e2e-test/library/album1/hello.txt` 上传成功
- 远端目录 `/open115-e2e-test/backups/db.json` 上传成功
- `manifest.db` 已生成

## 增量验证结果

### 第一次备份
- 扫描 2
- 上传 2
- 跳过 0

### 第二次备份（无修改）
- 扫描 2
- 上传 0
- 跳过 2

### 第三次备份（修改 1 个文件）
- 扫描 2
- 上传 1
- 跳过 1

## 当前判断

Open115 模式已经具备：
- token 模式接入
- 目录浏览
- 单文件上传
- 目录自动创建
- manifest 增量索引
- copy 增量备份主流程

## 仍建议继续补强

- `sync` 模式
- 更完整的 stop/cancel 行为
- 大文件/分片上传专项验证
- 前端完整 UI 自测
