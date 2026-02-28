---
name: Hosts GUI 可视化工具
overview: 使用 Wails (Go + Vue 3) 构建一个毛玻璃风格的现代化 Hosts 管理桌面应用，支持可视化配置、一键切换、系统托盘等功能。
todos:
  - id: install-wails
    content: 安装 Wails CLI 并初始化 Vue 3 项目
    status: completed
  - id: backend-api
    content: 改造 Go 后端，暴露 API 给前端调用
    status: completed
  - id: frontend-ui
    content: 开发毛玻璃风格的 Vue 3 前端界面
    status: completed
  - id: systray
    content: 添加系统托盘支持
    status: completed
  - id: build-exe
    content: 编译打包为 Windows EXE
    status: completed
isProject: false
---

# 现代化 Hosts 管理 GUI 工具

## 技术选型

**推荐使用 Wails v2 + Vue 3 + TailwindCSS**


| 方案             | 优点                      | 缺点           |
| -------------- | ----------------------- | ------------ |
| **Wails (推荐)** | 单 EXE 约 10MB，原生性能，Go 后端 | 需要安装 Node.js |
| Electron       | 生态丰富                    | 体积大 (100MB+) |
| Fyne           | 纯 Go，体积小                | UI 较丑，难做毛玻璃  |


## 界面设计

毛玻璃 (Glassmorphism) 风格特点：

- 半透明背景 + 模糊效果
- 渐变色边框
- 柔和阴影
- 圆角卡片

```
+--------------------------------------------------+
|  [Logo] Multi-Host Proxy          [_] [□] [×]    |
+--------------------------------------------------+
|                                                   |
|  +--------------------+  +---------------------+  |
|  |  ● dev-a           |  |  Profile 详情        |  |
|  |    127.0.0.1:8081  |  |                      |  |
|  |    [Running]       |  |  Name: dev-a         |  |
|  +--------------------+  |  Port: 8081          |  |
|                          |  Status: ● Running   |  |
|  +--------------------+  |                      |  |
|  |  ○ dev-b           |  |  Hosts 映射:         |  |
|  |    127.0.0.1:8082  |  |  ┌────────────────┐  |  |
|  |    [Stopped]       |  |  │ api.example.com │  |  |
|  +--------------------+  |  │ → 10.0.0.1      │  |  |
|                          |  └────────────────┘  |  |
|  [+ 添加 Profile]        |                      |  |
|                          |  [编辑] [启动/停止]   |  |
+--------------------------------------------------+
|  系统托盘图标：右键菜单快速切换 Profile           |
+--------------------------------------------------+
```

## 核心功能

1. **Profile 卡片列表** - 左侧显示所有配置，状态指示灯
2. **详情编辑面板** - 右侧编辑 hosts 映射
3. **一键启停** - 每个 profile 独立控制
4. **系统托盘** - 最小化到托盘，右键快速切换
5. **开机自启** - 可选
6. **深色/浅色主题** - 自动跟随系统

## 项目结构

```
Zephy/
├── main.go                    # Wails 入口
├── app.go                     # Go 后端逻辑
├── frontend/
│   ├── src/
│   │   ├── App.vue            # 主界面
│   │   ├── components/
│   │   │   ├── ProfileCard.vue    # Profile 卡片
│   │   │   ├── ProfileEditor.vue  # 编辑面板
│   │   │   └── HostsTable.vue     # Hosts 表格
│   │   └── styles/
│   │       └── glassmorphism.css  # 毛玻璃样式
│   └── wailsjs/               # 自动生成的 Go 绑定
├── internal/                  # 复用现有代码
│   ├── config/
│   ├── hosts/
│   └── proxy/
└── build/                     # 编译输出
```

## 实施步骤

1. 安装 Wails CLI
2. 初始化 Wails + Vue 3 项目
3. 迁移现有 Go 后端代码
4. 开发前端 UI 组件
5. 实现系统托盘功能
6. 编译打包为 EXE

---

## 变更日志（按时间顺序）

> 说明：这里记录每次迭代的“做了什么 + 为什么 + 踩过的坑”。EXE 属于构建产物，不作为源码提交，源码变更以仓库为准。

### 2026-02-27：从 CLI 迁移到 Wails GUI

- **新增**：Wails 桌面端框架（Go 后端 + Vue 前端），可视化管理 profiles/hosts。
- **新增**：多 profile 启停、hosts 编辑、导入/导出、重复域名检测/去重。
- **新增**：语言切换（中文/英文）与设置入口。

### 2026-02-27：性能与可用性改造

- **变更**：移除毛玻璃/透明/Backdrop（改为不透明 UI），降低性能开销并减少 WebView2 交互异常。
- **变更**：hosts 编辑从“组件化条目”改为“记事本式文本编辑”，支持直接编辑注释/空行/任意格式。

### 2026-02-27：窗口拖动、托盘与退出行为

- **修复**：frameless 标题栏无法拖动到副屏。
  - 采用 Wails 的 `CSSDragProperty/CSSDragValue` + CSS `--wails-draggable: drag/no-drag`。
- **修复/优化**：关闭窗口不退出（隐藏到托盘），托盘菜单提供 Show/Hide/Quit，Quit 才真正退出并停止代理。

### 2026-02-27：系统代理开关

- **新增**：每个配置支持“设为系统代理/关闭系统代理”。
- **兼容性增强**：ProxyServer 写入改为 `http=...;https=...`，并清理可能覆盖手动代理的 PAC/AutoDetect（如果存在）。

---

## 踩坑总结（重要）

### 1) Wails frameless 拖动不是 `-webkit-app-region`

- **现象**：标题栏看似可拖拽，但无法把窗口拖到副屏/拖拽不生效。
- **原因**：Wails v2 推荐使用自定义 CSS 拖拽属性而不是 Chromium 的 `-webkit-app-region`。
- **修复**：
  - `main.go` 设置：
    - `CSSDragProperty: "--wails-draggable"`
    - `CSSDragValue: "drag"`
  - CSS 设置：
    - 标题栏：`--wails-draggable: drag`
    - 按钮：`--wails-draggable: no-drag`

### 2) 透明/毛玻璃会引发交互异常与性能问题

- **现象**：拖拽区域、点击命中、性能出现异常。
- **修复**：禁用 WebView/Window 透明与 Backdrop，改为不透明 UI。

### 3) Wails v2 托盘能力有限

- **现象**：仅靠 `HideWindowOnClose` 无法满足“托盘图标 + 菜单 + 退出语义”。
- **方案**：改用 `systray` 实现托盘菜单逻辑（Show/Hide/Quit）。

### 4) 托盘图标透明（Windows）

- **现象**：右下角托盘显示成透明/不可见。
- **原因**：Windows 托盘对 PNG/透明通道兼容性差（尤其是小图标缩放与 alpha）。
- **修复**：使用 `.ico` 作为托盘图标（已引入 `assets/tray.ico` 并 embed）。

### 5) 系统代理“看起来不生效”

- **常见原因**：
  - PAC/AutoDetect 覆盖手动代理；
  - ProxyServer 格式不兼容某些场景（仅 `ip:port` vs `http=...;https=...`）；
  - 应用使用 WinHTTP（与 WinINET 分离），仅改注册表不会影响所有程序。
- **修复/增强**：
  - Enable 写入 `http=...;https=...`；
  - 设置 `ProxyOverride=<local>`；
  - 尝试清理 `AutoConfigURL` 并设置 `AutoDetect=0`；
  - 调用 `InternetSetOption` 通知系统刷新代理设置。

---

## Hosts 修改模块 —— 强约束规格（已实现）

**总原则**：必须最小化修改范围，仅操作受控标记块，绝不影响用户原有系统配置。

### 核心目标

- **必须真实修改 Windows 系统 hosts 文件**实现 profile 启用/关闭时的域名映射切换。
- 禁止用代理/DNS/本地 DNS 服务/内存映射替代。

### 系统路径（不可更改）

- 仅允许修改：`C:\Windows\System32\drivers\etc\hosts`
- 不得修改其他系统文件（除备份文件外）。

### 写入策略（标记块）

写入内容必须使用独立标记块包裹：

```txt
# >>> Zephy Managed Start
120.92.124.158 365.kdocs.cn
# <<< Zephy Managed End
```

规则：

- 不允许删除或修改标记块外内容
- 不允许重写整个文件结构
- **仅替换本工具管理的标记区块**

### 启用 Profile 行为

- 读取系统 hosts
- 查找并删除旧 Zephy 标记块（如存在）
- 保留其他所有内容
- 写入新的 Zephy 标记块（以 Profile 配置为准覆盖）
- 可选：`ipconfig /flushdns`

### 关闭 Profile 行为

- 删除 Zephy 标记块
- 保留其他所有原始内容

### 备份机制（强制）

- 每次写入前生成备份：`hosts.bak_yyyyMMddHHmmss`（同目录）
- 写入失败自动恢复备份

### 权限控制

- 必须检测管理员权限
- 无管理员权限：拒绝执行并提示用户（不绕过、不静默）

### 冲突处理

- 用户手动改了标记块内部：下次启用以 profile 覆盖
- 标记块外永远不改

### 严格禁止行为（已遵守）

- 禁止修改注册表（含系统代理/WinHTTP/DNS 等）
- 禁止影响除 hosts 文件外的系统配置

---

## 变更日志（新增）

### 2026-02-27：切换为“真实修改系统 hosts”模式

- **变更**：`Start/Stop` 不再启动代理服务器，而是启用/关闭系统 hosts 的 Zephy 标记块。
- **移除**：系统代理/注册表相关能力与 UI（为满足“严格禁止行为”）。
- **新增**：管理员权限校验 + 写入前备份 + 写入失败恢复 + 可选 flushdns。

