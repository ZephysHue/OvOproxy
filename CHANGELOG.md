# 更新日志

## v1.0.1 (2026-06-24)

### 在线更新
- GitHub Releases API 检测新版本，12 小时定时检查
- 标题栏红点气泡提示"有新版本啦，点击即可更新"
- 后台下载 → 下载完成 → 点击重启 → 自动替换 exe 生效
- 设置面板新增版本号显示 + 手动检查更新按钮

### 托盘菜单修复
- 每个 Profile 合并为一个切换项（点击即切换启用/禁用），不再显示两份

### 配置模板化
- `configs/` 从仓库移除，改为 `configs.example/` 示例模板
- 首次运行自动从模板复制默认配置，用户配置永不提交到仓库

### 构建流程优化
- 编译前强制清理前端 dist/.vite 缓存
- ldflags 注入版本号到 exe
- 首次构建自动初始化 release/configs/

## 2026-06-18

### 订阅功能优化（参考 SwitchHosts 设计）
- 刷新时比对条目内容，无变化跳过写盘 + 系统 hosts 重写
- 标记块加只读提醒 `# >>> subscription (auto-managed, do not edit)`

### 新增远程 Hosts 订阅功能
- **config.go**：Profile 新增 `subscription_url`/`subscription_interval`/`subscription_enabled`/`subscription_last_fetch` 字段
- **app.go**：
  - `SetSubscription`：设置订阅 URL 和刷新间隔，立即触发刷新
  - `RemoveSubscription`：清除订阅配置
  - `RefreshSubscription`：HTTP GET 拉取远程 hosts → 解析 → 写入文件底部 `# >>> subscription` 标记块 → 若已启用则自动更新系统 hosts
  - `startSubscriptionAutoRefresh`：30s 轮询定时器，按各 profile 间隔自动刷新
  - 订阅结果返回 `{status, message, last_fetch, entry_count}`
- **ProfileEditor.vue**：新增订阅配置面板（URL 输入、刷新按钮、间隔设置、状态显示）
- 设计原则：每个 profile 一个 URL，远程条目以标记块管理，刷新时替换旧块

### 建立推送前自测流程
- **verify.sh**：14 项自动化检查
- **main.go**：互斥锁名 → `HostOVOManager_Mutex`
- **tray.go**：托盘标题 `ZephyHosts` → `HostOVO`，tooltip 同步更新
- **assets/tray.ico**：QQ 企鹅托盘图标（ImageMagick 生成，可靠格式）
- **build.bat**：移除不可靠的 PowerShell ICO 转换，托盘图标作为静态资源提交

**图标同步机制**（换图标时三步走）：
```
1. 替换 appicon.png（唯一源图）
2. convert appicon.png -resize 32x32 -colors 256 assets/tray.ico
3. cp appicon.png frontend/src/assets/images/logo-universal.png
```
`appicon.png` → exe 图标（build.bat 自动） | `tray.ico` → 托盘（静态提交） | `logo-universal.png` → 界面内（静态提交）

### 新增 CHANGELOG.md 更新日志
- README.md 精简为项目简介，改动记录统一在 CHANGELOG.md

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
