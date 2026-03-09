# Open115 当前版本使用说明 / 发布前检查清单

> 适用版本：包含 `provider=open115`、`copy/sync`、`manifest.db`、`open115_encrypt.temp/stream`、`stream debug` 能力的当前分支。

---

## 1. 推荐使用姿势

### 当前默认推荐

如果你要稳定可用，推荐优先使用：

- `provider: open115`
- `open115_encrypt.enabled: false`

如果你要加密：

### 推荐优先级
1. `open115_encrypt.mode = temp`
   - 已做真实加密上传验证
   - 更稳
2. `open115_encrypt.mode = stream`
   - 已做 debug 闭环验证
   - 仍建议作为实验模式使用

---

## 2. 获取 token

### 推荐方式
在 UI 中点击：

- **获取 Token（OpenList）**

然后在打开的页面中：
1. 选择 `115 Network Disk Verification`
2. 勾选 `Use parameters provided by OpenList`
3. 留空 `Client ID / Secret`
4. 获取：
   - `access_token`
   - `refresh_token`

填回项目即可。

---

## 3. 最小配置示例

### 3.1 Open115 明文上传

```yaml
provider: open115

open115:
  enabled: true
  access_token: your_access_token
  refresh_token: your_refresh_token
  root_id: "0"

backup:
  library_dir: /data/library
  backups_dir: /data/backups
  remote_dir: /immich-backup
  mode: copy
  manifest_path: ./config/manifest.db
  allow_remote_delete: false
```

### 3.2 Open115 加密上传（推荐 temp）

```yaml
provider: open115

open115:
  enabled: true
  access_token: your_access_token
  refresh_token: your_refresh_token
  root_id: "0"

open115_encrypt:
  enabled: true
  password: your-password
  salt: your-salt
  mode: temp
  filename_mode: plain
  algorithm: aes256gcm-v1
  temp_dir: /tmp/immichto115-open115-encrypt
  min_free_space_mb: 1024

backup:
  library_dir: /data/library
  backups_dir: /data/backups
  remote_dir: /immich-backup
  mode: copy
  manifest_path: ./config/manifest.db
  allow_remote_delete: false
```

### 3.3 Open115 加密上传（实验 stream）

```yaml
open115_encrypt:
  enabled: true
  password: your-password
  salt: your-salt
  mode: stream
  filename_mode: plain
  algorithm: aes256gcm-v1
  temp_dir: /tmp/immichto115-open115-encrypt
  min_free_space_mb: 1024
```

> [!WARNING]
> `stream` 模式目前更适合实验验证；正式场景建议优先 `temp`。

---

## 4. 首次上线检查顺序

### 第一步：基础测试

```bash
curl -X POST http://127.0.0.1:8096/api/v1/open115/test
```

预期：

```json
{"message":"115 Open 连接成功","success":true}
```

### 第二步：列根目录

```bash
curl 'http://127.0.0.1:8096/api/v1/open115/ls?path=%2F'
```

如果这一步失败，不要直接跑备份，先重新获取 token。

### 第三步：小目录验证

先找一个很小的测试目录：
- 1~3 个小文件

先跑一轮：
- 明文
- 或 `temp` 加密

确认远端有文件后，再切真实目录。

---

## 5. 增量 copy 验收标准

### 第一次备份
预期：
- 扫描 N
- 上传 N
- 跳过 0

### 第二次备份（无变化）
预期：
- 扫描 N
- 上传 0
- 跳过 N

### 修改一个文件后
预期：
- 扫描 N
- 上传 1
- 跳过 N-1

---

## 6. sync 模式使用建议

如果你使用：

```yaml
backup:
  mode: sync
```

请确认：

```yaml
allow_remote_delete: true
```

才会真的删除远端多余文件。

### 推荐策略
- 先用 `copy` 跑稳定
- 再切 `sync`
- 第一次启用 `sync` 时，先在测试目录验证

---

## 7. 加密模式建议

### `temp` 模式
优点：
- 已做真实上传验证
- 更稳
- 失败更容易排查

缺点：
- 占额外临时磁盘空间

### `stream` 模式
优点：
- 理论上更省本地空间
- 更接近长期目标

缺点：
- 当前更依赖 token / 频控情况
- 更适合实验与调试

### 实际建议
- 正式使用优先 `temp`
- `stream` 先小文件测试，再逐步扩大

---

## 8. 常见问题

### Q1. `refresh frequently`
含义：
- 115 Open 限频 / 风控

处理：
- 降低验证频率
- 不要连续大量 `ls` / `test`
- 必要时重新拿 token

### Q2. `refresh token error`
含义：
- 当前 token 很可能已失效或异常

处理：
- 重新获取 `access_token / refresh_token`
- 用全新实例先测 `open115/test`

### Q3. Git push 后本地报 `update_ref failed`
处理原则：
- 如果输出里已经有：
  - `master -> master`
- 则远端 push 已成功
- 本地 `.git/logs/...` 写失败不影响 GitHub 已更新

---

## 9. 发布前检查清单

- [ ] `go test ./...` 通过
- [ ] `open115/test` 通过
- [ ] 根目录 `open115/ls` 通过
- [ ] 小目录明文上传通过
- [ ] 小目录 `temp` 加密上传通过
- [ ] 第二次 copy 跳过未变化文件通过
- [ ] 修改 1 个文件只上传 1 个通过
- [ ] 如启用 `sync`，确认 `allow_remote_delete` 开关逻辑符合预期
- [ ] 如启用 `stream`，至少做过一次 debug 闭环验证

---

## 10. 当前推荐的发布策略

### 稳定版推荐
- Open115
- token 模式
- `copy`
- 不加密 或 `temp` 加密

### 实验版
- Open115
- token 模式
- `stream` 加密
- 只在小目录或测试环境使用
