# HostOVO (原 Zephy / OvOproxy)

Windows 下的多 Profile Hosts 切换 + 本地代理轻量工具。

## 技术栈

| 层级 | 技术 |
|------|------|
| 桌面框架 | Wails v2 + Go |
| 前端 UI | Vue 3 + TypeScript + Vite + Tailwind CSS |
| 系统集成 | systray (系统托盘)、Windows hosts 编辑 |

## 编译

```batch
.\build.bat
```

编译产物：`release\HostOVO.exe`

---

## 改动记录

### 2026-06-17 — 后端死代码清理 + 品牌更新

**后端死代码清理**（由 CodeBuddy 执行）
- `app.go`：从 ~1926 行精简到 430 行，删除 ~1400 行死代码
  - 删除订阅相关全部代码（25+ 方法/函数/类型）
  - 删除审计日志（addAudit/GetAuditLogs/所有调用）
  - 删除去重功能（DedupHosts/DuplicateDomains）
  - 删除未使用方法（ImportHostsFromDialog/GetProxyLogs/UpdateHosts）
- `internal/config/config.go`：删除 Subscription 相关类型和字段
- `internal/hosts/hosts.go`：删除 DedupTextKeepLast、DedupEntriesKeepLast、DedupEntriesKeepFirst、DuplicateDomains 函数
- Go 交叉编译验证通过

**品牌更新**
- 应用图标替换为 QQ 企鹅 logo
- 编译输出文件名从 `ZephyHosts` 改为 `HostOVO`
- 更新 `build.bat` 一键编译脚本适配新名称

**工程优化**
- 创建 `build.bat` 一键编译脚本（关闭旧进程 → 清理 → 编译 → 复制到 release）
- 添加 `appicon.png` 解决应用图标缺失
- `.cursor/` 目录纳入 Git 管理
- GitHub 推送流程建立（Token 认证）

### 2026-04-16 — UI 大重构 + Hosts 去重修复（Cursor 会话记录）

> 以下为 Cursor IDE 中完成的改动，已归档

**前端 UI 优化**（6 项改动）
1. **删除订阅远程 host 模块** — 移除 SubscriptionPanel.vue 及所有引用
2. **导入/导出调整** — 移除配置详情顶部"导入"按钮；"导出"移至左侧列表右键菜单（默认保存到桌面，txt 格式）
3. **重命名移至右键菜单** — 从配置详情顶部移除，改由右键列表触发
4. **删除操作审计模块** — 移除 AuditPanel.vue
5. **删除格式语法预览模块** — 移除 DiagnosticsPanel.vue
6. **保存/删除按钮位置调整** — "保存修改"移至顶部；"删除"隐藏至右键菜单

**Hosts 去重 Bug 修复**
- **问题**：Zephy 使用 `DedupEntriesKeepLast`（保留最后出现的 IP），与 SwitchHosts 和 Windows 系统 hosts 行为（首次匹配生效）相反
- **修复**：最终完全去掉去重逻辑，原样写入系统 hosts，与 SwitchHosts 行为一致

**编译流程规范化**
- 编译路径固定为 `release\` 目录
- 编译前自动关闭旧进程、清理旧产物
