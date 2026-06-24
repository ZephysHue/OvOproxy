# OvOproxy

Windows 下的多 Profile Hosts 切换 + 本地代理轻量工具。

**当前版本：v1.0.1** | 支持在线更新

## 核心功能

- 多 Profile 并行管理，支持单独启用/禁用
- 系统 Hosts 自动写入（标记块管理，支持一键禁用全部）
- 本地 HTTP 代理（每个 Profile 独立端口）
- 远程规则订阅（HTTPS 定时自动拉取）
- 备份与恢复 + 导入/导出
- 系统托盘常驻 + 在线更新

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

编译产物：`release\OvOproxy.exe`

## 分发

打包 `release\` 目录即可。首次运行自动从 `configs.example/` 模板初始化配置。

## 更新日志

详见 [CHANGELOG.md](./CHANGELOG.md)
