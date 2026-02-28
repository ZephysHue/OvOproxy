import { ref } from 'vue'

export type Lang = 'zh' | 'en'

const stored = (localStorage.getItem('lang') || '') as Lang
export const lang = ref<Lang>(stored === 'en' || stored === 'zh' ? stored : 'zh')

const dict = {
  zh: {
    appTitle: '多 Hosts 代理工具',
    profiles: '配置列表',
    add: '新增',
    settings: '设置',
    noProfiles: '还没有配置',
    clickAddToCreate: '点击“新增”创建一个配置',
    selectProfile: '选择一个配置进行编辑',

    start: '启动',
    stop: '停止',
    running: '运行中',
    stopped: '已停止',
    delete: '删除',
    rename: '重命名',

    hostMappings: 'Hosts 映射',
    entriesCount: '{count} 条',
    domainPlaceholder: '域名（例如 api.example.com）',
    ipPlaceholder: 'IP（例如 10.0.0.1）',
    noHostMappings: '暂无映射',
    addMappingHint: '在上方添加“域名 → IP”映射',
    unsavedChanges: '有未保存的修改',
    saveChanges: '保存修改',

    import: '导入',
    export: '导出',
    dedupe: '去重',
    duplicatesFound: '发现重复域名：{count} 个',

    systemProxy: '系统代理',
    setSystemProxy: '设为系统代理',
    unsetSystemProxy: '关闭系统代理',
    renameProfile: '重命名配置',
    newProfileName: '新配置名',
    save: '保存',

    newProfile: '新建配置',
    profileName: '配置名',
    listenIP: '监听 IP',
    port: '端口',
    cancel: '取消',
    createProfile: '创建配置',
    profileNameRequired: '配置名不能为空',
    listenIPRequired: '监听 IP 不能为空',
    portRangeError: '端口必须在 1-65535 之间',

    language: '语言',
    chinese: '中文',
    english: 'English',
  },
  en: {
    appTitle: 'Multi-Host Proxy',
    profiles: 'Profiles',
    add: 'Add',
    settings: 'Settings',
    noProfiles: 'No profiles yet',
    clickAddToCreate: 'Click \"Add\" to create one',
    selectProfile: 'Select a profile to edit',

    start: 'Start',
    stop: 'Stop',
    running: 'Running',
    stopped: 'Stopped',
    delete: 'Delete',
    rename: 'Rename',

    hostMappings: 'Host Mappings',
    entriesCount: '{count} entries',
    domainPlaceholder: 'Domain (e.g. api.example.com)',
    ipPlaceholder: 'IP (e.g. 10.0.0.1)',
    noHostMappings: 'No host mappings',
    addMappingHint: 'Add a domain → IP mapping above',
    unsavedChanges: 'You have unsaved changes',
    saveChanges: 'Save Changes',

    import: 'Import',
    export: 'Export',
    dedupe: 'Dedupe',
    duplicatesFound: 'Duplicate domains: {count}',

    systemProxy: 'System Proxy',
    setSystemProxy: 'Set as System Proxy',
    unsetSystemProxy: 'Disable System Proxy',
    renameProfile: 'Rename Profile',
    newProfileName: 'New name',
    save: 'Save',

    newProfile: 'New Profile',
    profileName: 'Profile Name',
    listenIP: 'Listen IP',
    port: 'Port',
    cancel: 'Cancel',
    createProfile: 'Create Profile',
    profileNameRequired: 'Profile name is required',
    listenIPRequired: 'Listen IP is required',
    portRangeError: 'Port must be between 1 and 65535',

    language: 'Language',
    chinese: '中文',
    english: 'English',
  },
} satisfies Record<Lang, Record<string, string>>

type DictKey = keyof typeof dict.en

export function setLang(next: Lang) {
  lang.value = next
  localStorage.setItem('lang', next)
}

export function t(key: DictKey, vars?: Record<string, string | number>) {
  const table = dict[lang.value] as Record<DictKey, string>
  let s: string = table[key] ?? String(key)
  if (vars) {
    for (const [k, v] of Object.entries(vars)) {
      s = s.replaceAll(`{${k}}`, String(v))
    }
  }
  return s
}

