# Zephy UI 优化、Hosts 写入修复与全局 Skill/Rule 安装

## 原始需求

1. 删除订阅远程 host 模块
2. 删除详情中的"导入"按钮，将"导出"移至左侧配置列表右键菜单（默认导出桌面，txt 格式）
3. 将重命名从详情移至左侧配置列表右键菜单
4. 删除操作审计模块
5. 删除格式语法预览模块
6. 保存按钮移到顶部删除位置，删除按钮移至右键菜单
7. 修复 Zephy 写入系统 hosts 与 SwitchHosts 行为不一致的问题（重复域名处理策略）
8. 删除去重功能（与 SwitchHosts 保持一致）
9. 安装全局 session-continuation rule、interactive skill、ask-next skill

## 当前阶段

编码 — 100% 完成，已编译产出 release 版本

## 已完成

- 删除 SubscriptionPanel.vue、DiagnosticsPanel.vue、AuditPanel.vue 组件 | `frontend/src/components/`
- ProfileEditor.vue 大幅简化：移除导入/导出/重命名/删除按钮、订阅面板、诊断面板、审计面板、语法预览面板、去重警告和按钮 | `frontend/src/components/ProfileEditor.vue`
- 保存按钮从底部 footer 移至顶部 header（原删除按钮位置），仅在有未保存修改时显示 | `frontend/src/components/ProfileEditor.vue`
- App.vue 新增右键上下文菜单（导出/重命名/删除），移除 handleImportHosts/handleDedup 及相关导入 | `frontend/src/App.vue`
- Go 后端 ExportHostsToDialog 改为默认桌面目录 + .txt 格式 | `app.go`
- 修复 hosts 写入问题：EntriesToMap 改为首次出现优先 | `internal/hosts/hosts.go`
- 新增 DedupEntriesKeepFirst 函数 | `internal/hosts/hosts.go`
- StartProfile 不再去重，所有条目原样写入系统 hosts（与 SwitchHosts 行为一致） | `app.go`
- 编译路径规范：wails build → build/bin/ → 复制到 release/ZephyHosts.exe | `wails.json`
- 安装 session-continuation rule | `~/.cursor/rules/session-continuation.mdc`
- 安装 interactive skill | `~/.cursor/skills/interactive/SKILL.md`
- 安装 ask-next skill | `~/.cursor/skills/ask-next/SKILL.md`

## 关键决策

| 决策 | 原因 | 排除方案 |
|---|---|---|
| 去重功能完全删除 | 与 SwitchHosts 保持一致，用户认为去重不重要 | 保留去重但改为 KeepFirst |
| EntriesToMap 改为首次出现优先 | Windows hosts 解析规则是"第一个匹配生效"，原来 KeepLast 导致代理规则与系统行为不一致 | 不改 EntriesToMap（会导致代理和系统 hosts 行为不同） |
| StartProfile 不去重直接写入 | 完全匹配 SwitchHosts 行为，保留用户配置原意 | DedupKeepFirst（会丢失用户有意的重复条目） |
| 导出默认桌面 + txt | 用户要求 | 保持原来的 .hosts 格式和文件选择器默认路径 |
| 编译产物路径 release/ | 用户要求，build/bin/ 是 Wails 默认不可配置的输出目录 | 修改 wails.json outputdir（实测不生效） |

## 发现与风险

- Wails v2 的 `outputdir` 配置项不生效，编译产物始终在 `build/bin/`，需手动复制到 `release/`
- Go 后端的订阅相关代码（auto-refresh scheduler 等）未删除，仅前端入口移除；如有已配置 auto_enabled=true 的 profile，后台仍会执行刷新
- 审计日志 Go 代码（addAudit/GetAuditLogs）仍保留在 app.go 中，各处仍在写入审计记录，只是前端不再展示
- Zephy/ 内部有独立 .git 仓库，工作区根目录 E:\OVOhost 无 Git

## 活跃上下文

- 项目路径：`E:\OVOhost\Zephy`
- 编译产物：`E:\OVOhost\Zephy\release\ZephyHosts.exe`
- 编译流程：`taskkill ZephyHosts → 删除旧 release exe → wails build -clean → 复制到 release/`

## 恢复指令

新会话先读 `E:\OVOhost\.cursor\archives\2026-04-16-zephy-ui-optimization-and-hosts-fix.md` 了解历史。如需清理后端残留代码（订阅模块、审计模块），从 `app.go` 和 `internal/` 入手。
