# OvOproxy 开发标准

面向单人/小团队的精简规范，不搞大公司那一套，但该有的底线都要有。

---

## 1. Git 规范

### 分支策略

只用 `main` 分支，不打 PR，不走 review 流程（单人项目不需要）。但必须遵守：

- **推送前本地通过 `bash verify.sh`**
- 不推送编译不过的代码
- 不推送包含个人配置的文件（configs/ 已 gitignore）

### Commit 格式

```
<type>: <简短描述>

<详细说明（可选）>
```

| type | 用途 |
|------|------|
| `feat` | 新功能 |
| `fix` | 修 bug |
| `refactor` | 重构（不改变功能） |
| `docs` | 文档 |
| `chore` | 杂项（构建、gitignore 等） |

**规则**：一个 commit 只做一件事。不要混搭（比如 feat 里夹带 fix）。

### Tag 与 Release

- tag 格式：`v<major>.<minor>.<patch>`，如 `v1.0.1`
- 每次 tag 必须对应 GitHub Release，上传 exe 作为资产
- 不做 pre-release，直接 stable

---

## 2. 代码规范

### Go

- 遵循 `gofmt` 默认风格（不调参数）
- 导出方法 → PascalCase，内部方法 → camelCase
- 错误处理：不吞错误，至少 `runtime.LogError`
- 并发：`mu.Lock()` 后必须有对应的 `defer mu.Unlock()`，禁止持锁做 IO
- 配置文件结构体统一放 `internal/config/`
- 新增包放在 `internal/<包名>/`

### TypeScript / Vue

- 组件命名：PascalCase 文件名（`ProfileCard.vue`）
- Props 和 emits 必须显式声明类型
- 组件间传对象深拷贝，不传引用
- import 顺序：Vue 核心 → 第三方 → wailsjs → 项目内
- `any` 尽量不用，非用不可时加注释说明原因
- 新增 i18n key 必须同时补中英文
- **状态管理**：Vue 3.2 + TypeScript 4.6 下 `ref` 泛型推断偶现 `never`，复杂状态对象优先用 `reactive` 替代多个独立 `ref`

### CSS

- 全局样式放 `style.css` 的 `@layer components` 或 `@layer utilities` 中
- 组件样式用 `<style scoped>`
- 颜色、阴影、圆角等视觉参数统一用 CSS 变量（`style.css` 的 `:root`）
- 不做暗色模式（已删除），只维护一套浅色变量

---

## 3. 构建与发布

### 版本号

- 唯一来源：`build.bat` 中的 `APP_VERSION` 变量
- 编译时通过 ldflags 注入：`-X main.Version=v1.0.1`
- 版本号递增规则：小修 `patch++`，新功能 `minor++`，大改 `major++`

### 构建命令

```batch
.\build.bat
```

**不直接用 `wails build`**。build.bat 封装了：
1. 清理旧进程 + 旧 build
2. 清理前端缓存（dist/.vite）
3. 重新 `npm run build`
4. 带 ldflags 版本号编译
5. 复制 exe → release/
6. 首次自动初始化 configs + 复制模板

### 发布 Checklist

每次发布前逐项确认：

- [ ] `bash verify.sh` 全部通过
- [ ] 本地完整 `.\build.bat` 编译通过
- [ ] 本地启动 exe 功能正常（手动冒烟）
- [ ] `CHANGELOG.md` 已更新
- [ ] `frontend/src/changelog.ts` 已同步
- [ ] `README.md` 版本号已更新
- [ ] `build.bat` 的 `APP_VERSION` 已更新
- [ ] git commit + push
- [ ] GitHub 创建 tag + Release + 上传 `release\OvOproxy.exe`

---

## 4. 前后端对接

### Wails 绑定规则

Go struct 导出到前端时，Wails 生成的 TypeScript 类型**保留 Go 字段原名**：

```go
type Result struct {
    HasUpdate   bool   // 前端: result.HasUpdate
    DownloadURL string // 前端: result.DownloadURL
}
```

**例外**：Go struct 有 `json:"snake_case"` tag 时，前端对应 snake_case。

**三大绑定陷阱**（高频踩坑）：

| 陷阱 | 现象 | 规则 |
|------|------|------|
| 返回类型在 `internal/` 包 | 方法从 App.d.ts 消失，前端报 `never` | **绑定返回类型必须在 main 包** |
| struct 上挂方法 | TS 类型推断崩溃，`ref.value` 报 `never` | **绑定 struct 禁止挂方法** |
| 字段名用 snake_case 但不加 json tag | 前后端字段名不一致，数据丢失 | **加 json tag 或用 PascalCase** |

### 新增 API

1. Go 方法掛在 `*App` 上，PascalCase 命名
2. 返回类型必须可导出且**在 main 包中**
3. `wails build` 自动重新生成 `frontend/wailsjs/`
4. 前端 import 路径：`../../wailsjs/go/main/App`
5. **沙箱验证**：推代码前在沙箱重建 wailsjs 桩文件跑 `vue-tsc --noEmit`

---

## 5. 质量保证

### 编译检查

**Go**：
```bash
mkdir -p frontend/dist && echo '<html></html>' > frontend/dist/index.html
GOOS=windows go build ./...
```
注意：`go build ./...` exit 0 不代表编译通过！go:embed 资源缺失会提前中止。

**前端**：
```bash
cd frontend && npx vue-tsc --noEmit
```

### 必须测试的场景

- [ ] 首次启动（无 configs 目录）
- [ ] 管理员权限 + 非管理员权限
- [ ] 启用/禁用 Profile
- [ ] 窗口隐藏 → 托盘菜单恢复
- [ ] 在线更新检测流程

### 在线更新调试

- GitHub API **必须带 `User-Agent` 头**，否则返回 403
- 未认证请求限 60 次/小时，设置 `GITHUB_TOKEN` 环境变量提升到 5000
- 仓库无 Release 时 API 返回 404，代码已做容错处理
- 测试更新流程：临时改 `build.bat` 的 `APP_VERSION` 为更早版本号编译

---

## 6. 文档维护

| 文档 | 内容 | 更新时机 |
|------|------|---------|
| `README.md` | 项目简介、功能列表、编译方式、版本号 | 每次发版 |
| `CHANGELOG.md` | 版本更新明细 | 每次发版 |
| `CONTRIBUTING.md` | 本文档（开发标准） | 标准变更时 |
| `frontend/src/changelog.ts` | 更新日志结构化数据（设置面板展示用） | 每次发版 |

---

## 记录

本标准的制定参考了项目实际开发中遇到的问题和业界通用实践。
首次版本：v1.0.1，基于 2026-06-24 会话总结。
