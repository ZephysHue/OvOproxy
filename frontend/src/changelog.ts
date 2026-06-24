// 更新日志结构化数据
// 与 CHANGELOG.md 保持同步，此文件供设置弹窗展示用

export interface ChangelogEntry {
  title: string
  desc: string
}

export interface ChangelogVersion {
  date: string
  entries: ChangelogEntry[]
}

export const changelog: ChangelogVersion[] = [
  {
    date: 'v1.0.1 (2026-06-24)',
    entries: [
      { title: '在线更新', desc: '支持 GitHub Releases 自动检测更新，12h 定时检查 + 标题栏红点提示 + 下载后重启替换' },
      { title: '设置面板增强', desc: '新增当前版本号显示 + 手动检查更新按钮 + 重启应用按钮' },
      { title: '托盘菜单修复', desc: '每个 Profile 合并为一个切换项（不再显示启用/禁用两份）' },
      { title: '配置模板化', desc: 'configs 从仓库移除改为 configs.example 模板，clone 不泄露用户配置' },
      { title: '构建流程优化', desc: '编译前清理前端缓存，ldflags 注入版本号，首次构建自动创建 configs' },
    ],
  },
  {
    date: '2026-06-24',
    entries: [
      { title: '统一应用命名', desc: '全位置统一为 OvOproxy（窗口标题、托盘、exe、顶栏等）' },
      { title: '删除主题切换', desc: '移除无效的暗黑模式切换（原 CSS 无 dark 样式，死代码清理）' },
      { title: '修复托盘点击', desc: '托盘菜单"显示主窗口"置顶 + AlwaysOnTop 翻转强制前台' },
      { title: '设置面板增强', desc: '新增项目仓库地址与反馈联系方式' },
      { title: '设置新增更新日志', desc: '设置弹窗内可查看版本更新记录' },
    ],
  },
  {
    date: '2026-06-18',
    entries: [
      { title: '订阅功能优化', desc: '刷新时比对条目内容，无变化跳过写盘；标记块加只读提醒' },
      { title: '远程 Hosts 订阅', desc: 'Profile 新增订阅 URL/间隔/自动刷新；30s 轮询定时器' },
      { title: '推送前自测流程', desc: 'verify.sh 14 项自动化检查；托盘图标格式修复' },
    ],
  },
  {
    date: '2026-06-17',
    entries: [
      { title: '删除冲突提示', desc: '启用多配置按列表顺序写入 hosts，不再提示冲突' },
      { title: '图标编译修复', desc: '编译前强制删除旧 icon.ico，确保 Wails 重新生成' },
      { title: '品牌更新', desc: '应用图标替换；编译输出文件名更新' },
      { title: '后端死代码清理', desc: 'app.go 从 ~1926 行精简到 ~430 行' },
    ],
  },
  {
    date: '2026-04-16',
    entries: [
      { title: '前端 UI 大重构', desc: '导入/导出移至右键菜单；删除审计/诊断模块' },
      { title: 'Hosts 去重修复', desc: '修复去重逻辑与 Windows 系统 hosts 行为相反的问题' },
    ],
  },
]
