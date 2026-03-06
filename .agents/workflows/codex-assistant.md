---
description: 调用 OpenAI Codex CLI 执行代码任务（重构、Bug 修复、测试生成、代码解释、迁移、审查、文档生成）
---

# Codex Assistant Workflow

当用户通过 `/codex-assistant` 调用时，按以下步骤执行：

## 前置准备

1. 读取 skill 文件，了解 Codex 的能力和最佳实践
   // turbo
2. 使用 `view_file` 工具读取 `~/.gemini/antigravity/skills/codex-assistant/SKILL.md`

## 执行流程

3. **解析用户需求** - 从用户的自然语言输入中提取：
   - 任务类型（重构 / Bug修复 / 测试生成 / 代码解释 / 跨语言迁移 / 代码审查 / 文档生成 / 样板代码 / 代码清理）
   - 目标文件或代码片段
   - 任何特殊要求或约束

4. **构建 Codex Prompt** - 根据任务类型构建清晰的 prompt，格式为：

   ```
   [任务类型]: [具体描述]
   [上下文信息，如相关代码、文件路径等]
   ```

5. **执行 Codex 命令** - 使用 `run_command` 工具运行以下命令：

   ```powershell
   echo "[构建好的prompt]" | codex exec
   ```

   注意：
   - 如果用户指定了特定文件，先用 `view_file` 读取文件内容，将内容包含在 prompt 中
   - 如果在 Windows 上失败，尝试在 PowerShell 中直接运行
   - 等待命令完成，超时设为 30 秒

6. **返回结果** - 将 Codex 的输出直接展示给用户，包括：
   - 生成的代码（如有）
   - 解释说明（如有）
   - 修复建议（如有）
   - 如果 Codex 执行失败，显示错误信息并建议排查步骤

## 注意事项

- 如果 `codex` 命令不可用，提示用户安装：`npm install -g @openai/codex`
- Codex 生成的代码建议 review 后使用
- 复杂任务可以拆分为多个小任务逐步执行
