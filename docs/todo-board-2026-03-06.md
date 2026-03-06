# ImmichTo115 修复看板（2026-03-06）

> 基于 `docs/code-review-2026-03-06.md` 与 `docs/fix-plan-2026-03-06.md` 整理。
> 目标：把问题清单压缩成可直接执行的 Todo 看板。
> 当前仍是**计划文档**，不代表这些项已开始修复。

## 使用方式

- 先从 `P0` 开始，按顺序推进。
- 每完成一个任务，补上验证结果。
- 不建议跨批次并行大改，避免认证、备份、部署三条主链同时漂移。

---

## P0：必须优先处理

### AUTH
- [ ] `AUTH-01` 收紧 setup 前后的认证边界
  - 范围：`internal/api/router.go`、`internal/config/config.go`
  - 完成标准：未完成初始化时也不会把敏感接口整体裸露出去
  - 验证：未配置状态下访问 `/`、`/api/v1/config`、`/api/v1/backup/start`

- [ ] `AUTH-02` 修正前端状态未知时的路由放行逻辑
  - 范围：`web/src/router.ts`、`web/src/api.ts`
  - 完成标准：后端异常、401、未初始化三种状态能清晰区分
  - 验证：模拟 API 不可达 / 401 / setup 未完成

### BACKUP
- [ ] `BACKUP-01` 让 rclone 失败可结构化上报到备份主流程
  - 范围：`internal/rclone/rclone.go`、`internal/api/router.go`
  - 完成标准：失败不会再被视作成功阶段结束
  - 验证：错误路径、错误远端、手动停止三种场景

- [ ] `BACKUP-02` 让通知结果与真实备份结论对齐
  - 范围：`internal/api/router.go`、`internal/notify/bark.go`
  - 完成标准：只有真实成功才发送成功通知；失败/取消有单独结果
  - 验证：Bark 标题正文包含中文、空格、emoji；成功/失败/取消三种场景

---

## P1：稳定运行与交付

### OPS
- [ ] `OPS-01` 修复 Docker 健康检查与镜像依赖不一致
  - 范围：`deploy/docker-compose.yml`、`deploy/Dockerfile`
  - 完成标准：容器健康检查可实际执行并正确反映状态
  - 验证：冷启动、重启、异常退出后健康状态是否正确

- [ ] `OPS-02` 收口 root 运行与配置目录权限
  - 范围：`deploy/Dockerfile`、`deploy/install.sh`、`internal/config/config.go`
  - 完成标准：服务默认权限更收敛，配置目录权限与文档一致
  - 验证：Docker 与 systemd 两种模式均可正常运行、读写配置

### RELEASE
- [ ] `RELEASE-01` 统一版本能力、安装脚本与发布流程
  - 范围：`cmd/server/main.go`、`deploy/install.sh`、`.github/workflows/release.yml`、`README.md`
  - 完成标准：程序版本、安装脚本展示、GitHub Release 行为一致
  - 验证：打 tag 后完整跑一次 release；安装脚本能正确显示版本

---

## P2：能力边界与用户认知

### RESTORE
- [ ] `RESTORE-01` 明确 Restore Explorer 当前能力边界
  - 范围：`web/src/views/RestoreExplorer.vue`、`README.md`
  - 完成标准：用户能一眼区分“可浏览”与“可恢复”
  - 验证：空目录/有文件/有文件夹三种场景文案与按钮行为一致

- [ ] `RESTORE-02` 修正 Restore Explorer 的选择语义
  - 范围：`web/src/views/RestoreExplorer.vue`
  - 完成标准：文件夹与文件的选择逻辑不再误导用户
  - 验证：全选、单选、文件夹行点击、批量操作预期一致

### STATUS
- [ ] `STATUS-01` 修正 Dashboard 的连接降级提示
  - 范围：`web/src/views/Dashboard.vue`、`web/src/components/Layout.vue`
  - 完成标准：轮询失败、WebSocket 断线、状态过期都有可见提示
  - 验证：断网、401、后端停止、WebSocket 中断场景

---

## P3：维护性与体验收尾

### DOC
- [ ] `DOC-01` 修正文档/API 漂移
  - 范围：`README.md`
  - 完成标准：方法、路径、能力描述与代码实现一致
  - 验证：逐条对照 API 表、功能说明、恢复说明、安装说明

### UI
- [ ] `UI-01` 抽离 Settings / Wizard 公共配置逻辑
  - 范围：`web/src/views/Settings.vue`、`web/src/views/Wizard.vue`、`web/src/components/CronScheduler.vue`
  - 完成标准：重复的默认配置、路径选择、校验逻辑尽可能复用
  - 验证：设置页与引导页行为保持一致，不引入回归

### A11Y
- [ ] `A11Y-01` 补齐 toast 与关键交互控件语义
  - 范围：`web/src/components/GlobalToast.vue`、`web/src/views/Settings.vue`、`web/src/views/Wizard.vue`、`web/src/views/RestoreExplorer.vue`
  - 完成标准：toast、开关、面包屑、可点击行具备更合理的语义与键盘可达性
  - 验证：键盘导航、屏幕阅读器基础行为、自定义控件 focus 流程

---

## 推荐执行顺序

1. `AUTH-01`
2. `AUTH-02`
3. `BACKUP-01`
4. `BACKUP-02`
5. `OPS-01`
6. `OPS-02`
7. `RELEASE-01`
8. `RESTORE-01`
9. `RESTORE-02`
10. `STATUS-01`
11. `DOC-01`
12. `UI-01`
13. `A11Y-01`

---

## 完成定义

每个任务关闭前，至少补齐：

- 修改文件列表
- 风险说明
- 验证命令 / 验证步骤
- 回归结果
- 是否影响 Docker / systemd / release / 文档

---

## 当前建议

如果现在开始真正修，建议第一轮只做这 4 个：

- `AUTH-01`
- `AUTH-02`
- `BACKUP-01`
- `BACKUP-02`

原因：这四项最直接影响**安全性**和**备份可信度**。