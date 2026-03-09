# Open115 模式加密备份设计方案

> 目标：在保留当前 `provider=open115` 的 token 接入、目录浏览、增量 copy/sync 能力基础上，为 **Open115 模式**补充“本地加密后再上传”的能力。

---

## 1. 背景

当前项目已经支持两条主线：

1. **WebDAV 模式**
   - 通过 `rclone + WebDAV` 上传到 115
   - 已支持 `Rclone Crypt` 加密

2. **Open115 模式**
   - 通过 `access_token / refresh_token` 调 115 Open API
   - 已支持目录浏览、单文件上传、multipart、大文件路径、manifest 增量 copy/sync
   - **目前不支持加密上传**

Open115 之所以暂未支持加密，是因为它绕过了 `rclone crypt`，直接上传本地原始文件字节和原始文件名。

---

## 2. 设计目标

Open115 加密备份希望达到：

- [ ] 上传前在本地完成文件内容加密
- [ ] 可选对文件名 / 目录名进行混淆
- [ ] 保持现有 `manifest.db` 增量逻辑可用
- [ ] 保留 `copy` / `sync` 模式
- [ ] 尽量不影响现有 WebDAV 模式
- [ ] 尽量不引入必须依赖 CGO 的库

---

## 3. 非目标（第一阶段不做）

- [ ] 不做在线预览
- [ ] 不做服务端解密下载
- [ ] 不做跨端实时恢复浏览器
- [ ] 不做和 `rclone crypt` 完全字节级兼容
- [ ] 不做多算法切换 UI

第一阶段只做：

> **本地加密后上传到 115 Open，支持增量备份和后续恢复所需的元数据记录。**

---

## 4. 总体方案

核心思路：

> 在 Open115 上传前增加一层“本地加密管道”，把原始文件转换为加密文件，再由现有 Open115 上传器上传。

也就是：

```text
本地原始文件
  -> 计算变化
  -> 本地加密（内容 / 可选文件名）
  -> 上传加密后的文件
  -> manifest 记录原始路径、远端路径、摘要、加密参数版本
```

---

## 5. 推荐实现路线

### 方案 A：项目内自定义加密层（推荐）

即在项目里自己实现一套稳定、可维护的加密格式。

#### 内容加密
推荐：
- `AES-256-GCM`
- 每个文件独立随机 `nonce`
- 主密钥由用户密码派生
- 派生方案：`scrypt` 或 `argon2id`

#### 文件名混淆
推荐：
- 第一阶段不做完全可逆目录树加密
- 先做：
  - 文件内容加密必做
  - 文件名可选“哈希化”或“映射化”

例如：
- 原文件：`library/2026/03/hello.jpg`
- 远端：`library/2026/03/3f4a8c9e.enc`

再通过 manifest 保存：
- 原路径
- 远端加密名

#### 优点
- 不依赖 rclone
- 完全适配 Open115 现有上传器
- 后续可持续演进

#### 缺点
- 需要自己维护加密格式
- 恢复逻辑也要自己做

---

### 方案 B：复用 rclone crypt 做“本地中转加密文件”

思路是：
- 先把文件通过 rclone crypt 或等价逻辑生成加密文件
- 再让 Open115 上传器上传这个加密文件

#### 优点
- 可以复用已有 crypt 语义

#### 缺点
- 架构会变得很拧巴
- Open115 不再是纯 API 上传器
- 仍然会引入大量 rclone 兼容细节

**不推荐。**

---

## 6. 建议的数据结构

### 6.1 新增配置项

建议在 `AppConfig` 中增加 `open115_encrypt` 或扩展现有 `encrypt`，但为了避免混淆，推荐单独分支：

```yaml
open115_encrypt:
  enabled: false
  password: ""
  salt: ""
  filename_mode: "mapped"   # mapped | plain
  algorithm: "aes256gcm-v1"
```

#### 字段解释
- `enabled`
  - 是否启用 Open115 模式加密
- `password`
  - 主密码
- `salt`
  - KDF salt
- `filename_mode`
  - `plain`：保留文件名，仅内容加密
  - `mapped`：文件名映射到 manifest，不在远端暴露原始文件名
- `algorithm`
  - 预留版本升级用

---

### 6.2 manifest 扩展字段

当前 `files` 表建议新增：

```sql
ALTER TABLE files ADD COLUMN encrypted INTEGER DEFAULT 0;
ALTER TABLE files ADD COLUMN encrypted_size INTEGER;
ALTER TABLE files ADD COLUMN remote_path TEXT;
ALTER TABLE files ADD COLUMN encryption_version TEXT;
ALTER TABLE files ADD COLUMN content_sha256 TEXT;
```

#### 用途
- `encrypted`
  - 是否走过 Open115 加密上传
- `encrypted_size`
  - 加密后文件大小
- `remote_path`
  - 远端真实存储路径（可能与原路径不同）
- `encryption_version`
  - 加密格式版本
- `content_sha256`
  - 原始内容摘要，用于更稳的变化判断

---

## 7. 加密文件格式建议

第一阶段推荐一个简单可维护的自定义格式：

```text
Magic Header: IM115ENC
Version: 1 byte
Salt length
Nonce length
Metadata length
Salt bytes
Nonce bytes
Metadata JSON
Ciphertext bytes
```

### metadata json 示例

```json
{
  "alg": "aes-256-gcm",
  "filename": "hello.jpg",
  "original_size": 123456,
  "mtime": 1741411414
}
```

#### 这样做的好处
- 后续恢复时容易识别
- 便于版本升级
- 便于调试
- 不把实现完全写死在代码里

---

## 8. 备份流程设计

### 8.1 copy 模式

当前流程：

```text
扫描本地 -> manifest 对比 -> 直接上传原文件 -> 更新 manifest
```

改造后：

```text
扫描本地
 -> manifest 对比
 -> 如果变化：
      生成加密临时文件 / 流
      上传加密文件
      更新 manifest（记录原路径与远端路径映射）
```

---

### 8.2 sync 模式

如果启用 `allow_remote_delete`：
- 仍按现有逻辑找本地缺失项
- 删除的是 `manifest.remote_path`
- 而不是推算原始路径

这样更安全。

---

## 9. 文件名策略建议

### 方案 1：内容加密，文件名保留

远端：
- `library/album1/hello.jpg.enc`

#### 优点
- 简单
- 目录结构还在
- 便于排查

#### 缺点
- 泄露原始文件名与目录结构

---

### 方案 2：文件名映射（推荐）

远端：
- `library/album1/3f4a8c9e1a.enc`

manifest：
- 存原路径与远端路径映射

#### 优点
- 更安全
- 不暴露原始文件名

#### 缺点
- 没有 manifest 就难恢复

---

### 方案 3：目录名也完全加密

#### 不建议第一阶段就做
因为会显著增加：
- 路径浏览复杂度
- sync 删除复杂度
- 恢复复杂度

---

## 10. 临时文件 vs 流式加密

### 方案 A：先生成临时加密文件（第一阶段推荐）

```text
原文件 -> 加密临时文件 -> 上传 -> 删除临时文件
```

#### 优点
- 实现简单
- 易调试
- 与当前 uploader 最兼容

#### 缺点
- 需要额外磁盘空间

---

### 方案 B：边加密边上传（后续优化）

```text
原文件流 -> 加密流 -> multipart 上传
```

#### 优点
- 节省磁盘

#### 缺点
- 实现复杂
- multipart 分块与加密边界更难处理

---

## 11. 推荐实施顺序

### Phase 1：最小可用版
- [ ] 新增 `open115_encrypt` 配置结构
- [ ] 新增本地加密器（AES-GCM）
- [ ] 先做“内容加密 + 文件名保留”
- [ ] 上传前生成临时 `.enc` 文件
- [ ] manifest 记录 `encrypted / remote_path / encryption_version`
- [ ] copy 模式跑通

### Phase 2：增强版
- [ ] 文件名映射（`filename_mode = mapped`）
- [ ] sync 模式删除走 `remote_path`
- [ ] 恢复逻辑需要的元数据补齐

### Phase 3：优化版
- [ ] 流式加密上传
- [ ] 大文件性能优化
- [ ] 更好的目录展示与恢复工具

---

## 12. 安全注意事项

- [ ] 默认不要开启 Open115 加密删除联动，除非 manifest 已验证稳定
- [ ] 如果使用文件名映射，manifest 是恢复关键，必须备份
- [ ] 建议给 manifest/db 单独做本地备份
- [ ] 密码丢失后无法恢复，应在 UI 中明确提示
- [ ] 同一个项目不要在中途随意切换加密算法版本

---

## 13. 我建议的最终取舍

如果要尽快把 Open115 加密做出来，我建议：

## 第一版只做：
- **内容加密**
- **文件名保留或仅简单 `.enc` 后缀**
- **manifest 记录必要元数据**
- **临时文件上传**

不要第一版就做：
- 完全目录树加密
- 流式加密
- 复杂恢复页面

因为第一版的目标应该是：

> **让 Open115 模式具备“可用的加密备份能力”，而不是一次性把恢复生态全做完。**

---

## 14. 代码结构草图（第一阶段）

如果正式开工，建议新增/修改如下：

```text
internal/
  open115crypt/
    config.go
    kdf.go
    encrypt.go
    header.go
    tempfile.go

  manifest/
    models.go        # 扩展 encrypted / remote_path / content_sha256 等字段
    sqlite.go        # 增加迁移逻辑

  backup/
    open115_copy.go  # 在上传前插入加密步骤
```

### 关键调用链建议

```text
Open115CopyRunner.Run()
  -> 比较原始文件变化
  -> 若未变化：skip
  -> 若变化：
       open115crypt.EncryptFileToTemp(...)
       backend.UploadFile(tempEncPath, remoteEncPath)
       manifest.Put(...)
       CleanupTempFile(...)
```

### manifest 扩展后的核心职责

- 记录原始路径
- 记录远端实际路径
- 记录是否加密
- 记录加密版本
- 记录原始内容摘要

这样后续做 `sync` 删除和恢复时，不需要“猜”远端文件名。

---

## 15. 临时文件空间占用与增量策略

### 临时文件方案的现实问题

如果采用“先加密成临时文件再上传”，会带来额外磁盘占用：

```text
原始文件 10GB
-> 临时加密文件 ~10GB+
-> 峰值磁盘占用接近 20GB+
```

所以第一阶段必须配套以下约束：

- **只对变化文件生成临时文件**
- **一次只处理一个大文件**（串行，避免并发爆盘）
- 上传成功后立即删除临时文件
- 对 `temp_dir` 做可用空间检查

### 为什么增量仍然成立

因为增量判断应始终基于：

- 原始路径
- 原始文件大小
- 原始 mtime
- （后续可升级为原始内容 hash）

也就是说：

```text
先判断原始文件有没有变化
-> 没变化：直接跳过，不加密，不上传
-> 有变化：才生成临时加密文件并上传
```

因此，临时文件只会出现在“有变化的文件”上，而不是每次全量都生成。

### 第一阶段推荐配置

```yaml
open115_encrypt:
  enabled: false
  password: ""
  salt: ""
  filename_mode: plain
  algorithm: aes256gcm-v1
  temp_dir: /tmp/immichto115-encrypt
  min_free_space_mb: 1024
```

### 第二阶段优化方向

如果第一阶段稳定，再升级为：

- 流式加密上传
- 边加密边 multipart 上传
- 尽量消除完整临时文件副本

---

## 16. 一句话结论

Open115 模式要支持加密，**最稳的做法不是复用 rclone crypt**，而是：

> **在项目内新增一层本地加密管道，把原始文件加密成临时文件后再走现有 Open115 上传器。**

第一阶段可接受“临时文件 + 串行 + 空间检查”，而增量判断必须始终基于**原始文件**而不是加密产物。
