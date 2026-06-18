# 更新日志

## 2026-06-17

### 删除冲突提示和 hosts 警告
- **App.vue**：删除 ActiveConflict 类型、冲突检测 computed、冲突面板模板、jumpToProfile
- **ProfileEditor.vue**：删除 validateHosts 警告栏、getRiskLines 风险检测、风险确认弹窗
- 启用多个配置时按列表顺序写入 hosts，不再提示冲突

### 修复图标编译机制
- **build.bat**：编译前强制删除 `build\windows\icon.ico`，确保 Wails 从 `build/appicon.png` 重新生成
- 根因：Wails v2 在 `build/windows/icon.ico` 已存在时跳过重新生成

### 修复冲突提示显示异常
- 冲突提示从 absolute 定位改为正常文档流
- 区分真冲突（不同 IP，红色）和同 IP 信息提示（琥珀色）
- 修复"收起历史/展开历史"文案错用问题
- 移除 slice(0,20) 硬截断

### 品牌更新
- 应用图标替换为 QQ 企鹅 logo
- 编译输出文件名 `ZephyHosts` → `HostOVO`
- 更新 `build.bat` 适配新名称

### 后端死代码清理
- **app.go**：从 ~1926 行精简到 ~430 行
  - 删除订阅相关全部代码（25+ 方法/函数/类型）
  - 删除审计日志（addAudit/GetAuditLogs）
  - 删除去重功能（DedupHosts/DuplicateDomains）
  - 删除未使用方法（ImportHostsFromDialog/GetProxyLogs/UpdateHosts）
- **internal/config/config.go**：删除 Subscription 相关类型和字段
- **internal/hosts/hosts.go**：删除去重函数

### 工程优化
- 创建 `build.bat` 一键编译脚本
- 添加 `appicon.png` 解决应用图标缺失
- `.cursor/` 目录纳入 Git 管理
- 更新 `README.md` 含完整改动时间线

---

## 2026-04-16（Cursor IDE 中完成）

### 前端 UI 大重构
- 删除订阅远程 host 模块（SubscriptionPanel.vue）
- 导入/导出调整：导出移至左侧列表右键菜单，默认桌面 .txt
- 重命名移至右键菜单
- 删除操作审计模块（AuditPanel.vue）
- 删除格式语法预览模块（DiagnosticsPanel.vue）
- 保存/删除按钮位置调整

### Hosts 去重 Bug 修复
- 修复 `DedupEntriesKeepLast` 与 Windows 系统 hosts 行为（首次匹配生效）相反的问题
- 最终完全去掉去重逻辑，原样写入系统 hosts
