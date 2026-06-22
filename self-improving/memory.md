# HOT Memory

## Communication
- 回复前必须等待用户确认方案，不要自作主张动手（2026-06-18, user correction ×2）
- 回复必须简洁明了，不写长篇大论（2026-06-18, user correction）
- 每次改动后必须输出总结：改了什么文件、做了什么、用户下一步做什么（2026-06-18, user rule）
- 如问题可能复现，总结中加"后续改进措施"字段（2026-06-18, user rule）
- 图表、详细模块设计等内容对用户是废话，非必要不输出（2026-06-18, user correction）

## Code
- 改 Go 结构体字段后，必须 grep 所有 struct 初始化位置，确保新字段已覆盖（2026-06-18）
- 改 Go export 字段时，同步检查 App.vue/ProfileEditor.vue 的本地 Profile 接口（2026-06-18）
- ICO 文件固化 32bit 格式，勿用 8bit/256色（2026-06-18, 托盘空白）
- 固定定位 UI（右键菜单、tooltip）统一做溢出翻转（2026-06-18）
- 推送前跑 verify.sh 自测（2026-06-18, user rule）

## Habits (永久铁律)
- 每轮回复前被纠正过→先查 self-improving/memory.md 再回复（2026-06-18, permanent）
- 每轮回复末输出总结(改什么+做什么+下一步)+必要时加后续改进措施
- 改代码前先确认方案，不直接动手
- 新 Go 导出方法→同步检查前端类型定义+所有 struct 初始化点
