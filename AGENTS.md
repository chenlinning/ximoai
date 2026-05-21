# XimoAi 项目规范

## 本次会话项目环境约束

- `D:\sub2api中转站` 是本次会话的项目目录；所有项目相关的读取、修改、构建和测试默认都应在该目录内进行。
- 本项目如果需要新增安装依赖，依赖文件统一安装或放置到 `D:\Program Files` 下；不同依赖应分别创建独立文件夹，避免混放。
- 本项目产生的构建产物、临时文件和工具缓存应保存到 `D:\sub2api中转站` 下的项目专用缓存目录中，不应写入用户主目录或系统默认缓存目录，除非用户明确要求。

## Coding Principles

Behavioral guidelines to reduce common LLM coding mistakes. Merge with project-specific instructions as needed.

**Tradeoff:** These guidelines bias toward caution over speed. For trivial tasks, use judgment.

### 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:
- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them - don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

### 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

### 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:
- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it - don't delete it.

When your changes create orphans:
- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

### 4. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:
- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:
```
1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
```

Strong success criteria let you loop independently. Weak criteria ("make it work") require constant clarification.

**These guidelines are working if:** fewer unnecessary changes in diffs, fewer rewrites due to overcomplication, and clarifying questions come before implementation rather than after mistakes.

## XimoAi Customization / Upstream Merge Policy

后续所有针对 sub2api 上游源码的定制，默认采用“最小冲突方案”：

- 优先新增 `ximoai`、`platformPatch`、`composables`、独立 handler/test/helper 文件承载定制逻辑，避免直接把定制大段写进上游高频文件。
- 必须改上游文件时，只保留必要挂接点，例如一行路由注册、一行函数调用、一个字段声明；复杂逻辑放到独立文件。
- 不为了风格、注释、格式统一而改上游文件；不得引入与功能无关的注释重写、乱码修复或批量格式化。
- 前端 i18n 定制优先通过 patch 文件运行时合并，不直接改 `frontend/src/i18n/locales/en.ts` 和 `frontend/src/i18n/locales/zh.ts`。
- Wire 生成文件冲突不手工长期维护；合并上游后重新生成并验证。
- 每次合并 sub2api 上游前，先检查本地定制与上游改动的文件交集，再决定是否拆分定制，避免把可隔离改动留在上游热区。
