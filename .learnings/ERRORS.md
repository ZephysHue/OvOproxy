# Errors
Command failures and integration errors.

---

## [ERR-20260618-001] wails_build_tsc
**Logged**: 2026-06-18T11:40:00Z
**Priority**: high
**Status**: resolved
**Area**: frontend

### Summary
TS 编译报错：hosts_file 类型不匹配

### Error
Type 'ProfileState' is not assignable to type 'Profile'. Types of property 'hosts_file' are incompatible. Type 'string | undefined' is not assignable to type 'string'.

### Context
- Go 端将 HostsFile json tag 改为 omitempty（可选）
- 前端 Profile 接口仍是必填 string
- 修复：hosts_file?: string

### Resolution
- **Resolved**: 2026-06-18
- **Commit**: 176ded1
- **Notes**: 前端两处 Profile 接口 hosts_file 改为可选

### Metadata
- Reproducible: yes
- Related Files: frontend/src/App.vue, frontend/src/components/ProfileEditor.vue, internal/config/config.go
- See Also: LRN-20260618-001

---

## [ERR-20260618-002] refresh_read_directory
**Logged**: 2026-06-18T11:40:00Z
**Priority**: critical
**Status**: resolved
**Area**: backend

### Summary
订阅导入失败：RefreshSubscription 尝试以文件方式读取 exeDir

### Error
read E:\OVOhost\Zephy\release: Incorrect function

### Context
- AddRemoteProfile 未设置 HostsFile → 字段为空
- RefreshSubscription 中 Join(exeDir, "") = exeDir
- os.ReadFile(exeDir) → Windows 报 "Incorrect function"（目录不可作文件读）

### Resolution
- **Resolved**: 2026-06-18
- **Commit**: 637b49e
- **Notes**: AddRemoteProfile 中补上 HostsFile 路径

### Metadata
- Reproducible: yes
- Related Files: app.go
- See Also: LRN-20260618-001
