# Learnings
Corrections, insights, and knowledge gaps captured during development.
**Categories**: correction | insight | knowledge_gap | best_practice

---

## [LRN-20260618-001] correction
**Logged**: 2026-06-18T11:40:00Z
**Priority**: critical
**Status**: promoted
**Area**: backend, frontend

### Summary
改 Go 结构体后必须 grep 所有 struct 初始化位置

### Details
多次出现：改 config.Profile 新增字段后，前端 Profile 接口未同步导致 TS 编译报错
- AddRemoteProfile 中 HostsFile 漏设 → 刷新读目录报错
- ProfileEditor.vue 本地 Profile 类型缺少新字段

### Suggested Action
改 Go 字段 → grep struct 初始化 → 同步前端 Profile 接口 → verify.sh → push

### Metadata
- Source: user_feedback
- Related Files: internal/config/config.go, app.go, frontend/src/App.vue, frontend/src/components/ProfileEditor.vue
- Tags: typescript, golang, data-model, synchronization
- Pattern-Key: struct.field.sync
- Recurrence-Count: 3
- First-Seen: 2026-06-17
- Last-Seen: 2026-06-18

---

## [LRN-20260618-002] correction
**Logged**: 2026-06-18T11:40:00Z
**Priority**: high
**Status**: promoted
**Area**: frontend

### Summary
ICO 文件必须用 32bit 格式，8bit/256色会导致 systray 空白

### Details
ImageMagick 默认 convert 产生的 8bit 256色 ICO 与 getlantern/systray 底层 LoadImage API 不兼容
PowerShell System.Drawing.Icon.Save() 产生的是带 ICONDIR 头的完整 ICO 文件而非 ICONIMAGE 资源数据

### Suggested Action
ICO 生成统一用: convert appicon.png -resize 32x32 -depth 8 assets/tray.ico（32bit）

### Metadata
- Source: error
- Related Files: assets/tray.ico, tray.go
- Tags: ico, systray, windows, imagemagick
- Pattern-Key: ico.format.32bit
- Recurrence-Count: 2
- First-Seen: 2026-06-18
- Last-Seen: 2026-06-18

---

## [LRN-20260618-003] best_practice
**Logged**: 2026-06-18T11:40:00Z
**Priority**: high
**Status**: promoted
**Area**: config, frontend, backend

### Summary
固定定位 UI（右键菜单/tooltip/dropdown）统一做溢出翻转

### Details
右键菜单用 clientX/clientY 定位，窗口缩小时底部菜单被截断
修复：检测 clientY+menuHeight > innerHeight 则翻转到上方

### Suggested Action
所有 fixed/absolute 定位的 UI 组件都加溢出检测：下→翻上，右→翻左

### Metadata
- Source: user_feedback
- Related Files: frontend/src/App.vue
- Tags: css, positioning, overflow, ux
- Pattern-Key: ui.overflow.flip
- Recurrence-Count: 1
- First-Seen: 2026-06-18
- Last-Seen: 2026-06-18

## [LRN-20260618-004] correction
**Logged**: 2026-06-18T12:00:00Z
**Priority**: high
**Status**: promoted
**Area**: frontend, backend

### Summary
Vue 组件间传对象必须深拷贝，不能传引用

### Details
selectProfile 直接传 profiles 数组中的对象引用给 selectedProfile，子组件可绕过 Vue 单向数据流修改父组件状态

### Suggested Action
selectedProfile.value = { ...profile, hosts: { ...profile.hosts } }

### Metadata
- Source: audit
- Related Files: frontend/src/App.vue
- Tags: vue, reactivity, data-flow
- Pattern-Key: vue.ref.deepcopy
- Recurrence-Count: 1
- First-Seen: 2026-06-18
- Last-Seen: 2026-06-18

---

## [LRN-20260618-005] correction
**Logged**: 2026-06-18T12:00:00Z
**Priority**: medium
**Status**: promoted
**Area**: backend

### Summary
锁释放后重新获取时必须验证状态仍有效（TOCTOU）

### Details
StartProfile 在 RUnlock→Lock 窗口期间 profile 可能被删除，ApplyProfileBlock 已写入系统 hosts 留下孤儿块

### Suggested Action
重新获取锁后验证 profile 仍存在，不存在则回滚副作用

### Metadata
- Source: audit
- Related Files: app.go
- Tags: concurrency, toctou, race-condition
- Pattern-Key: lock.toctou.verify
- Recurrence-Count: 1
- First-Seen: 2026-06-18
- Last-Seen: 2026-06-18

---

## [LRN-20260618-006] correction
**Logged**: 2026-06-18T13:30:00Z
**Priority**: critical
**Area**: workflow

### Summary
诊断项目状态前必须先 git fetch 对比 origin/main，沙箱本地副本可能严重落后

### Details
本次沙箱本地 HEAD 落后 origin/main 整整 23 个提交，导致误判"Aqua/订阅/HostOVO 改动全丢失"，
差点基于过时的旧版深色代码乱改。实际所有工作都在远程，只是本地没 fetch。

### Suggested Action
任何"代码状态/历史"判断前，先执行：git fetch origin && git log origin/main --oneline -10
&& git rev-list --left-right --count HEAD...origin/main，确认本地与远程同步后再动手。

### Metadata
- Source: error
- Related Files: (git workflow)
- Tags: git, sync, diagnosis, sandbox
- Pattern-Key: git.fetch.before.diagnose
- Recurrence-Count: 1
- First-Seen: 2026-06-18
- Last-Seen: 2026-06-18

---

## [LRN-20260618-007] best_practice
**Logged**: 2026-06-18T13:30:00Z
**Priority**: high
**Area**: frontend

### Summary
暗色→浅色(Aqua)重构后，必须全局 grep 清理残留的 text-*-200/300 暗色淡彩类

### Details
Aqua 浅色重构只改了 CSS 变量和部分模板，大量按钮/徽章/状态块仍挂着旧暗色体系的
text-blue-200/text-emerald-300/text-red-300 等淡彩工具类。这些为深色背景设计的浅色文字
放到奶白浅底上对比度极低，表现为"文字太淡"。用户点名的左右侧"启用host"按钮即此问题。
另：半透明底 bg-*-500/30 在浅底偏重，应降到 /12~/15。紫色应按"奶白+蓝/橙"偏好收敛为橙。

### Suggested Action
浅色重构收尾时 grep: text-(blue|amber|red|cyan|emerald|green|purple)-(100|200|300)
逐个评估背景，文字统一加深到 -600/-700，半透明底 /30→/15、/20→/12，紫色→橙色。

### Metadata
- Source: user_feedback
- Related Files: ProfileCard.vue, ProfileEditor.vue, BackupPanel.vue, App.vue, AddProfileModal.vue, RenameProfileModal.vue
- Tags: css, tailwind, theme, contrast, aqua, dark-residue
- Pattern-Key: ui.dark.residue.cleanup
- Recurrence-Count: 1
- First-Seen: 2026-06-18
- Last-Seen: 2026-06-18

---

## [LRN-20260618-008] correction
**Logged**: 2026-06-18T14:00:00Z
**Priority**: critical
**Area**: backend, workflow

### Summary
go build 因 go:embed 资源缺失会提前中止,给出"编译通过"假象;必须用 verify.sh 验证

### Details
本次 `GOOS=windows go build ./...` 返回 exit 0,但其实在 main.go 的
`//go:embed all:frontend/dist` 缺失处就中止了,根本没编译到 internal/winhosts。
导致漏掉一个真 bug: helper 函数 flushDNS() 与函数参数 flushDNS bool 同名被遮蔽
(cannot call non-function flushDNS)。verify.sh 先建 dist 占位再编译才暴露出来。

### Suggested Action
1. 验证 Go 编译前先 `mkdir -p frontend/dist && echo '<html></html>' > frontend/dist/index.html`
   或直接跑 verify.sh,不能只看 `go build ./...` 退出码。
2. Go helper 命名避开同作用域的函数参数名(flushDNS bool → helper 用 flushDNSCache)。

### Metadata
- Source: error
- Related Files: internal/winhosts/winhosts_windows.go, verify.sh, main.go
- Tags: golang, go-embed, build-false-positive, naming-shadow, verify
- Pattern-Key: gobuild.embed.false-positive
- Recurrence-Count: 1
- First-Seen: 2026-06-18
- Last-Seen: 2026-06-18
