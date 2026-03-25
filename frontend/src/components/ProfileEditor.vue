<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { t } from '../i18n'
import { GetProxyAddress } from '../../wailsjs/go/main/App'
import DiagnosticsPanel from './DiagnosticsPanel.vue'
import BackupPanel from './BackupPanel.vue'
import SubscriptionPanel from './SubscriptionPanel.vue'

interface Profile {
  name: string
  listen_ip: string
  port: number
  running: boolean
  hosts_file: string
  system_hosts_active?: boolean
  proxy_active?: boolean
  proxy_error?: string
}

const props = defineProps<{
  profile: Profile
  hostsText: string
  duplicates: Array<{ domain: string; count: number }>
}>()

const emit = defineEmits<{
  saveText: [name: string, text: string]
  delete: [name: string]
  start: [name: string]
  stop: [name: string]
  importHosts: [name: string]
  exportHosts: [name: string]
  dedup: [name: string]
  rename: [name: string]
  reloadHosts: [name: string]
}>()

const editedText = ref('')
const hasChanges = ref(false)
const copiedMsg = ref(false)
const showDiagnostics = ref(false)
const hostsWarnings = ref<string[]>([])

const proxyAddress = computed(() => `${props.profile.listen_ip}:${props.profile.port}`)

watch(() => props.hostsText, (v) => {
  editedText.value = v || ''
  hasChanges.value = false
  validateHosts(v || '')
}, { immediate: true })

function validateHosts(text: string) {
  const warnings: string[] = []
  const lines = text.split('\n')
  let invalidIP = false
  let invalidDomain = false
  for (const line of lines) {
    const trimmed = line.trim()
    if (!trimmed || trimmed.startsWith('#')) continue
    const parts = trimmed.split(/\s+/)
    if (parts.length >= 2) {
      const domain = parts[1]
      if (domain.includes(':')) {
        warnings.push(t('hostsWarningPort'))
        break
      }
    }
    if (parts.length >= 1) {
      const ip = parts[0]
      if (ip === '127.0.0.1' || ip === '0.0.0.0') {
        warnings.push(t('hostsWarningLoopback'))
        break
      }
      if (!isValidIP(ip)) {
        invalidIP = true
      }
    }
    if (parts.length >= 2) {
      const domain = parts[1]
      if (!isValidDomain(domain)) {
        invalidDomain = true
      }
    }
  }
  if (invalidIP) warnings.push(t('hostsWarningInvalidIP'))
  if (invalidDomain) warnings.push(t('hostsWarningInvalidDomain'))
  hostsWarnings.value = warnings
}

function confirmDelete() {
  if (confirm(`${t('delete')} "${props.profile.name}"?`)) {
    emit('delete', props.profile.name)
  }
}

function saveChanges() {
  emit('saveText', props.profile.name, editedText.value)
  hasChanges.value = false
}

function onEdit() {
  hasChanges.value = true
  validateHosts(editedText.value)
}

async function copyProxyAddr() {
  try {
    const addr = await GetProxyAddress(props.profile.name)
    await navigator.clipboard.writeText(addr)
    copiedMsg.value = true
    setTimeout(() => { copiedMsg.value = false }, 1500)
  } catch (e) {
    console.error('Copy failed:', e)
  }
}

const showFind = ref(false)
const findQuery = ref('')
const findMatches = ref<number[]>([])
const currentMatchIndex = ref(0)
const textareaRef = ref<HTMLTextAreaElement | null>(null)
const lineNumberRef = ref<HTMLDivElement | null>(null)
const ruleFilter = ref('')

type ParsedLine = {
  lineNo: number
  raw: string
  type: 'mapping' | 'comment' | 'blank' | 'invalid'
  enabled?: boolean
  ip?: string
  domain?: string
}

const parsedLines = computed<ParsedLine[]>(() => {
  const lines = editedText.value.split('\n')
  return lines.map((raw, idx) => {
    const lineNo = idx + 1
    const trimmed = raw.trim()
    if (!trimmed) {
      return { lineNo, raw, type: 'blank' }
    }
    const commentedMapping = raw.match(/^\s*#\s*([0-9a-fA-F:.]+)\s+([^\s#]+)\s*$/)
    if (commentedMapping) {
      if (!isValidIP(commentedMapping[1]) || !isValidDomain(commentedMapping[2])) {
        return { lineNo, raw, type: 'invalid' }
      }
      return {
        lineNo,
        raw,
        type: 'mapping',
        enabled: false,
        ip: commentedMapping[1],
        domain: commentedMapping[2],
      }
    }
    const mapping = raw.match(/^\s*([0-9a-fA-F:.]+)\s+([^\s#]+)\s*$/)
    if (mapping) {
      if (!isValidIP(mapping[1]) || !isValidDomain(mapping[2])) {
        return { lineNo, raw, type: 'invalid' }
      }
      return {
        lineNo,
        raw,
        type: 'mapping',
        enabled: true,
        ip: mapping[1],
        domain: mapping[2],
      }
    }
    if (trimmed.startsWith('#')) {
      return { lineNo, raw, type: 'comment' }
    }
    return { lineNo, raw, type: 'invalid' }
  })
})

const toggleableRules = computed(() => parsedLines.value.filter(l => l.type === 'mapping'))
const filteredToggleableRules = computed(() => {
  const q = ruleFilter.value.trim().toLowerCase()
  if (!q) return toggleableRules.value
  return toggleableRules.value.filter(r => (r.domain || '').toLowerCase().includes(q))
})
const lineNumbers = computed(() => parsedLines.value.map(l => l.lineNo))

function toggleRule(lineNo: number, enabled: boolean) {
  const lines = editedText.value.split('\n')
  const idx = lineNo - 1
  if (idx < 0 || idx >= lines.length) return
  const line = lines[idx]
  if (enabled) {
    lines[idx] = line.replace(/^(\s*)#\s*/, '$1')
  } else {
    if (!/^\s*#/.test(line)) {
      lines[idx] = line.replace(/^(\s*)/, '$1# ')
    }
  }
  editedText.value = lines.join('\n')
  onEdit()
}

function toggleAllRules(enabled: boolean) {
  const lines = editedText.value.split('\n')
  for (const rule of toggleableRules.value) {
    const idx = rule.lineNo - 1
    if (idx < 0 || idx >= lines.length) continue
    const line = lines[idx]
    if (enabled) {
      lines[idx] = line.replace(/^(\s*)#\s*/, '$1')
    } else if (!/^\s*#/.test(line)) {
      lines[idx] = line.replace(/^(\s*)/, '$1# ')
    }
  }
  editedText.value = lines.join('\n')
  onEdit()
}

function isValidIP(ip: string): boolean {
  const ipv4 =
    /^(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)$/
  if (ipv4.test(ip)) return true
  // Lightweight IPv6 acceptance
  if (ip.includes(':') && /^[0-9a-fA-F:]+$/.test(ip)) return true
  return false
}

function isValidDomain(domain: string): boolean {
  // Allow localhost and standard hostnames/domains
  if (domain === 'localhost') return true
  if (domain.length > 253) return false
  const labels = domain.split('.')
  if (labels.some(l => !l || l.length > 63)) return false
  return labels.every(l => /^[a-zA-Z0-9-]+$/.test(l) && !l.startsWith('-') && !l.endsWith('-'))
}

function handleKeydown(e: KeyboardEvent) {
  if ((e.ctrlKey || e.metaKey) && e.key === 'f') {
    e.preventDefault()
    showFind.value = true
  }
  if (e.key === 'Escape' && showFind.value) {
    showFind.value = false
  }
}

function updateFindMatches() {
  const q = findQuery.value.toLowerCase()
  if (!q) {
    findMatches.value = []
    currentMatchIndex.value = 0
    return
  }
  const text = editedText.value.toLowerCase()
  const indices: number[] = []
  let idx = text.indexOf(q)
  while (idx !== -1) {
    indices.push(idx)
    idx = text.indexOf(q, idx + 1)
  }
  findMatches.value = indices
  currentMatchIndex.value = indices.length > 0 ? 0 : -1
  scrollToMatch()
}

function findNext() {
  if (findMatches.value.length === 0) return
  currentMatchIndex.value = (currentMatchIndex.value + 1) % findMatches.value.length
  scrollToMatch()
}

function findPrev() {
  if (findMatches.value.length === 0) return
  currentMatchIndex.value = (currentMatchIndex.value - 1 + findMatches.value.length) % findMatches.value.length
  scrollToMatch()
}

function scrollToMatch() {
  if (findMatches.value.length === 0 || currentMatchIndex.value < 0) return
  const pos = findMatches.value[currentMatchIndex.value]
  if (textareaRef.value) {
    textareaRef.value.focus()
    textareaRef.value.setSelectionRange(pos, pos + findQuery.value.length)
    const linesBefore = editedText.value.substring(0, pos).split('\n').length
    const lineHeight = 24
    textareaRef.value.scrollTop = Math.max(0, (linesBefore - 3) * lineHeight)
  }
}

function onEditorScroll() {
  if (!textareaRef.value || !lineNumberRef.value) return
  lineNumberRef.value.scrollTop = textareaRef.value.scrollTop
}
</script>

<template>
  <div class="h-full flex flex-col glass-card overflow-hidden">
    <!-- Header -->
    <div class="p-5 border-b border-white/10">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-4">
          <div 
            class="w-12 h-12 rounded-xl flex items-center justify-center"
            :class="profile.proxy_active 
              ? 'bg-emerald-500/20 border border-emerald-500/30' 
              : 'bg-red-500/20 border border-red-500/30'"
          >
            <svg class="w-6 h-6" :class="profile.proxy_active ? 'text-emerald-400' : 'text-red-400'" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2"/>
            </svg>
          </div>
          <div>
            <h2 class="text-xl font-semibold text-white/90 flex items-center gap-2">
              {{ profile.name }}
              <span 
                v-if="profile.system_hosts_active" 
                class="text-xs px-2 py-0.5 rounded bg-blue-500/30 text-blue-300"
              >
                {{ t('hostsEnabled') }}
              </span>
            </h2>
            <p class="text-sm text-white/50 mt-0.5 flex items-center gap-2">
              {{ profile.listen_ip }}:{{ profile.port }}
              <span 
                class="px-2 py-0.5 rounded-full text-xs"
                :class="profile.proxy_active 
                  ? 'bg-emerald-500/20 text-emerald-300' 
                  : 'bg-red-500/20 text-red-300'"
              >
                {{ profile.proxy_active ? t('proxyActive') : t('proxyError') }}
              </span>
            </p>
            <p v-if="profile.proxy_error" class="text-xs text-red-400 mt-1">
              {{ profile.proxy_error }}
            </p>
          </div>
        </div>

        <div class="flex items-center gap-2">
          <button
            class="glass-button text-cyan-300 hover:bg-cyan-500/20 border-cyan-500/30"
            @click="copyProxyAddr"
            :title="t('copyProxyAddr')"
          >
            {{ copiedMsg ? t('copied') : t('copyProxyAddr') }}
          </button>
          <button
            class="glass-button text-white/80 hover:bg-slate-700"
            @click="emit('importHosts', profile.name)"
          >
            {{ t('import') }}
          </button>
          <button
            class="glass-button text-white/80 hover:bg-slate-700"
            @click="emit('exportHosts', profile.name)"
          >
            {{ t('export') }}
          </button>
          <button
            class="glass-button text-white/80 hover:bg-slate-700"
            @click="emit('rename', profile.name)"
            :disabled="profile.system_hosts_active"
            :class="{ 'opacity-50 cursor-not-allowed': profile.system_hosts_active }"
          >
            {{ t('rename') }}
          </button>
          <button
            class="glass-button"
            :class="profile.system_hosts_active 
              ? 'text-red-300 hover:bg-red-500/20'
              : 'text-emerald-300 hover:bg-emerald-500/20'"
            :disabled="!profile.proxy_active && !profile.system_hosts_active"
            :title="!profile.proxy_active && !profile.system_hosts_active ? t('proxyNotActive') : ''"
            @click="profile.system_hosts_active ? emit('stop', profile.name) : emit('start', profile.name)"
          >
            {{ profile.system_hosts_active ? t('disableHosts') : t('enableHosts') }}
          </button>
          <button
            class="glass-button text-red-400 hover:bg-red-500/20"
            :disabled="profile.system_hosts_active"
            :class="{ 'opacity-50 cursor-not-allowed': profile.system_hosts_active }"
            @click="confirmDelete"
          >
            {{ t('delete') }}
          </button>
        </div>
      </div>
    </div>

    <!-- Hosts Editor (Notepad-like) -->
    <div class="flex-1 p-5 overflow-hidden flex flex-col">
      <div v-if="duplicates.length > 0" class="mb-4 p-3 rounded-xl bg-amber-500/15 border border-amber-500/30 flex items-center justify-between">
        <div class="text-amber-200 text-sm">
          {{ t('duplicatesFound', { count: duplicates.length }) }}
        </div>
        <button class="glass-button text-amber-200 hover:bg-amber-500/20 border-amber-500/30" @click="emit('dedup', profile.name)">
          {{ t('dedupe') }}
        </button>
      </div>

      <div v-if="hostsWarnings.length > 0" class="mb-4 p-3 rounded-xl bg-orange-500/15 border border-orange-500/30">
        <div v-for="warn in hostsWarnings" :key="warn" class="text-orange-200 text-sm">
          {{ warn }}
        </div>
      </div>

      <div class="flex items-center justify-between mb-4">
        <h3 class="text-white/80 font-medium">{{ t('hostMappings') }}</h3>
        <div class="flex items-center gap-3">
          <button
            class="text-sm text-white/50 hover:text-white/80"
            @click="showDiagnostics = !showDiagnostics"
          >
            {{ t('diagnostics') }} {{ showDiagnostics ? '▼' : '▶' }}
          </button>
          <span class="text-sm text-white/40">{{ profile.hosts_file }}</span>
        </div>
      </div>

      <DiagnosticsPanel
        v-if="showDiagnostics"
        :profile-name="profile.name"
        :proxy-address="proxyAddress"
        class="mb-4"
      />

      <SubscriptionPanel :profile-name="profile.name" @changed="emit('reloadHosts', profile.name)" />
      <BackupPanel :profile-name="profile.name" @changed="emit('reloadHosts', profile.name)" />

      <div class="grid grid-cols-2 gap-3 mb-3">
        <div class="rounded-xl border border-slate-700/60 bg-slate-900 p-3 max-h-32 overflow-y-auto scrollbar-thin">
          <div class="flex items-center justify-between mb-2 gap-2">
            <div class="text-xs text-white/70">{{ t('rulesPanel') }}</div>
            <div class="flex items-center gap-1">
              <button class="glass-button text-[11px] px-2 py-1" @click="toggleAllRules(true)">{{ t('enableAll') }}</button>
              <button class="glass-button text-[11px] px-2 py-1" @click="toggleAllRules(false)">{{ t('disableAll') }}</button>
            </div>
          </div>
          <input
            v-model="ruleFilter"
            type="text"
            :placeholder="t('ruleFilter')"
            class="glass-input text-xs mb-2 py-2"
          />
          <div v-if="toggleableRules.length === 0" class="text-xs text-white/40">{{ t('noHostMappings') }}</div>
          <label
            v-for="rule in filteredToggleableRules"
            :key="`${rule.lineNo}-${rule.domain}`"
            class="flex items-center gap-2 text-xs text-white/80 py-1"
          >
            <input
              type="checkbox"
              :checked="rule.enabled"
              @change="toggleRule(rule.lineNo, ($event.target as HTMLInputElement).checked)"
            />
            <span class="text-white/40">{{ t('line') }} {{ rule.lineNo }}</span>
            <span class="truncate">{{ rule.domain }}</span>
          </label>
        </div>
        <div class="rounded-xl border border-slate-700/60 bg-slate-900 p-3 max-h-32 overflow-y-auto scrollbar-thin">
          <div class="text-xs text-white/70 mb-2">{{ t('syntaxPreview') }}</div>
          <div
            v-for="lineItem in parsedLines"
            :key="`preview-${lineItem.lineNo}`"
            class="text-xs py-0.5 font-mono"
            :class="lineItem.type === 'mapping' ? (lineItem.enabled ? 'text-emerald-300' : 'text-amber-300') : (lineItem.type === 'invalid' ? 'text-red-300' : 'text-white/40')"
          >
            <span class="text-white/40 mr-2">{{ lineItem.lineNo }}</span>
            <span v-if="lineItem.type === 'invalid'">{{ t('invalidRule') }}: {{ lineItem.raw }}</span>
            <span v-else>{{ lineItem.raw || ' ' }}</span>
          </div>
        </div>
      </div>

      <!-- Find Bar (Ctrl+F) -->
      <div v-if="showFind" class="mb-3 p-3 rounded-xl bg-slate-800/80 border border-slate-600/50 flex items-center gap-3">
        <input
          v-model="findQuery"
          type="text"
          :placeholder="t('find')"
          class="flex-1 glass-input text-sm"
          @input="updateFindMatches"
          @keydown.enter="findNext"
          @keydown.shift.enter="findPrev"
          autofocus
        />
        <span v-if="findMatches.length > 0" class="text-sm text-white/50">
          {{ t('matchCount', { current: currentMatchIndex + 1, total: findMatches.length }) }}
        </span>
        <span v-else-if="findQuery" class="text-sm text-white/40">
          {{ t('noMatches') }}
        </span>
        <button class="glass-button text-sm text-white/70" @click="findPrev" :disabled="findMatches.length === 0">
          {{ t('findPrev') }}
        </button>
        <button class="glass-button text-sm text-white/70" @click="findNext" :disabled="findMatches.length === 0">
          {{ t('findNext') }}
        </button>
        <button class="glass-button text-sm text-white/50" @click="showFind = false">
          {{ t('close') }}
        </button>
      </div>

      <div class="relative flex-1 flex">
        <div
          ref="lineNumberRef"
          class="w-12 rounded-l-xl border border-r-0 border-slate-700/70 bg-slate-900 text-right pr-2 pt-4 text-xs font-mono text-white/40 overflow-hidden"
        >
          <div v-for="ln in lineNumbers" :key="`ln-${ln}`" class="leading-6">{{ ln }}</div>
        </div>
        <textarea
          ref="textareaRef"
          v-model="editedText"
          class="w-full h-full rounded-r-xl rounded-l-none bg-slate-950 border border-slate-700/70 text-white/90 p-4 font-mono text-sm leading-6 outline-none focus:border-blue-400/60 resize-none scrollbar-thin"
          spellcheck="false"
          @input="onEdit"
          @keydown="handleKeydown"
          @scroll="onEditorScroll"
        />
      </div>
    </div>

    <!-- Footer: Save -->
    <div v-if="hasChanges" class="p-4 border-t border-white/10 bg-blue-500/10">
      <div class="flex items-center justify-between">
        <span class="text-blue-300 text-sm">{{ t('unsavedChanges') }}</span>
        <button
          class="glass-button bg-blue-500/30 text-blue-200 hover:bg-blue-500/40 border-blue-400/30"
          @click="saveChanges"
        >
          {{ t('saveChanges') }}
        </button>
      </div>
    </div>
  </div>
</template>
