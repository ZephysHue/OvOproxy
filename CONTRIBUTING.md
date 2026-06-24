# OvOproxy 开发标准

## 构建与发布

### 发布前必须通过 verify.sh

```bash
bash verify.sh
```

必须在 push 前通过所有检查。当前 5 大类：Go 交叉编译、前端 TS、embed 资源、关键文件完整性、旧名称残留。

### build.bat 修改规则

- 所有 `copy`/`xcopy` 目标目录必须先 `if not exist "..." mkdir ...`
- 版本号在 `build.bat` 顶部的 `APP_VERSION` 变量修改
- 每次发版同步三个地方：`build.bat` 版本号 → git tag → changelog

### 发版流程

```
1. 改 build.bat 的 APP_VERSION
2. 更新 frontend/src/changelog.ts + CHANGELOG.md
3. 更新 README.md 版本号
4. bash verify.sh → 通过
5. git commit → git push
6. GitHub Releases 新建 tag（与 APP_VERSION 一致），上传 release\OvOproxy.exe
```

## 前后端对接

### Wails 绑定的类型规则

Wails 自动生成的 TypeScript 类型保留 Go 字段的 **PascalCase**：

| Go 结构体字段 | 前端 TS 字段 |
|---------------|-------------|
| `HasUpdate bool` | `result.HasUpdate` |
| `DownloadURL string` | `result.DownloadURL` |
| `Latest string` | `result.Latest` |

**不是 snake_case！** 只有 Go struct 显式打了 `json:"snake_case"` tag 的才用小写。

### 新增 Go 导出方法

1. 方法名 PascalCase，接收器 `*App`
2. 返回类型必须是可导出的 Go 类型
3. 新增方法后，前端 `npm run build` 或 `wails build` 会自动重新生成 `frontend/wailsjs/` 绑定
4. `frontend/wailsjs/` 在 `.gitignore` 中，不提交

## 配置管理

- `configs.example/` 是仓库模板，只放 dev-a/dev-b 示例
- `configs/` 是本地用户配置，永远不提交
- 首次运行 `app.go initConfigDir()` 从模板自动复制
- build.bat 首次构建时从 `configs.example/` 初始化 `release/configs/`

## Go 编码

### 编译验证

`go build ./...` 的 exit 0 不可信！因为 `go:embed` 缺失时会提前中止：
```
mkdir -p frontend/dist && echo '<html></html>' > frontend/dist/index.html
GOOS=windows go build ./...
```
验证完务必清理 dist 占位文件。

### 锁操作

- 重新获取锁后验证状态仍有效（TOCTOU）
- mutex.Unlock → mutex.Lock 窗口期间目标可能被删除

## 前端编码

### 构建前清缓存

```
if exist "frontend\dist" rmdir /s /q "frontend\dist"
if exist "frontend\.vite" rmdir /s /q "frontend\.vite"
```

Vite 缓存残留会导致前端改动不生效，build.bat 已内置此步骤。

### Vue 组件间传对象

必须深拷贝，不能传引用：
```ts
selectedProfile.value = { ...profile, hosts: { ...profile.hosts } }
```

## Windows API

- ICO 固定 32bit：`convert appicon.png -resize 32x32 -depth 8 assets/tray.ico`
- 溢出翻转：所有 fixed/absolute 定位 UI 统一做溢出检测
