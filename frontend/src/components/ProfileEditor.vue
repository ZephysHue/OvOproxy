<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { t } from '../i18n'
import { GetProxyAddress, SetSubscription, RemoveSubscription, RefreshSubscription } from '../../wailsjs/go/main/App'
import BackupPanel from './BackupPanel.vue'
import type { Profile, ParsedLine, SubscriptionResult } from '../types'
import { SUBSCRIPTION_INTERVAL_OPTIONS, DEFAULT_SUB_INTERVAL } from '../constants'

const props = defineProps<{
  profile: Profile
  hostsText: string
}>()

const emit = defineEmits<{
  saveText: [name: string, text: string, confirmedRisk?: boolean]
  start: [name: string]
  stop: [name: string]
  reloadHosts: [name: string]
}>()

const editedText = ref('')
const hasChanges = ref(false)
const copiedMsg = ref(false)

watch(() => props.hostsText, (v) => {
  editedText.value = v || ''
  hasChanges.value = false
}, { immediate: true })

function saveChanges() {
  emit('saveText', props.profile.name, editedText.value, true)
  hasChanges.value = false
}

function onEdit() {
  hasChanges.value = true
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

// Subscription
const showSubPanel = ref(props.profile.type === 'remote')
const subUrl = ref('')
const subInterval = ref(DEFAULT_SUB_INTERVAL)
const subRefreshing = ref(false)
const subResult = ref<SubscriptionResult | null>(null)

watch(() => props.profile, (p) => {
  subUrl.value = p.subscription_url || ''
  subInterval.value = p.subscription_interval || DEFAULT_SUB_INTERVAL
  if (p.subscription_last_fetch) {
    subResult.value = { status: 'ok', message: '', last_fetch: p.subscription_last_fetch, entry_count: 0 }
  }
}, { immediate: true })

async function saveSubscription() {
  const url = subUrl.value.trim()
  if (!url) {
    await RemoveSubscription(props.profile.name)
    subResult.value = null
    return
  }
  try {
    await SetSubscription(props.profile.name, url, subInterval.value)
    emit('reloadHosts', props.profile.name)
  } catch (e: any) {
    console.error('SetSubscription:', e)
  }
}

async function refreshSubscription() {
  subRefreshing.value = true
  try {
    const result = await RefreshSubscription(props.profile.name)
    subResult.value = result
    emit('reloadHosts', props.profile.name)
  } catch (e: any) {
    subResult.value = { status: 'error', message: e?.message || String(e), last_fetch: '', entry_count: 0 }
  }
  subRefreshing.value = false
}

function formatLastFetch(ts: string): string {
  if (!ts) return ''
  try {
    const d = new Date(ts)
    return d.toLocaleString()
  } catch { return ts }
}

const showFind = ref(false)
const findQuery = ref('')
const findMatches = ref<number[]>([])
const currentMatchIndex = ref(0)
const textareaRef = ref<HTMLTextAreaElement | null>(null)
const lineNumberRef = ref<HTMLDivElement | null>(null)
const ruleFilter = ref('')

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
  if (ip.includes(':') && /^[0-9a-fA-F:]+$/.test(ip)) return true
  return false
}

function isValidDomain(domain: string): boolean {
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
  <div class="h-full flex flex-col glass-card overflow-y-auto scrollbar-thin">
    <!-- Header -->
    <div class="p-5 border-b border-neutral-200 sticky top-0 z-20 bg-white/70 backdrop-blur-xl backdrop-blur-sm">
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
            <h2 class="text-xl font-semibold text-neutral-900 flex items-center gap-2">
              {{ profile.name }}
              <span 
                v-if="profile.system_hosts_active" 
                class="text-xs px-2 py-0.5 rounded bg-blue-500/30 text-blue-300"
              >
                {{ t('hostsEnabled') }}
              </span>
              <span 
                v-if="profile.type === 'remote'" 
                class="text-xs px-2 py-0.5 rounded bg-purple-500/30 text-purple-300"
              >
                订阅
              </span>
            </h2>
            <p class="text-sm text-neutral-500 mt-0.5 flex items-center gap-2">
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
            v-if="hasChanges && profile.type !== 'remote'"
            class="glass-button bg-blue-500/30 text-blue-200 hover:bg-blue-500/40 border-blue-400/30"
            @click="saveChanges"
          >
            {{ t('saveChanges') }}
          </button>
        </div>
      </div>
    </div>

    <!-- Hosts Editor -->
    <div class="flex-1 p-5 flex flex-col">

      <div class="flex items-center justify-between mb-4">
        <h3 class="text-neutral-800 font-medium">{{ t('hostMappings') }}</h3>
        <span class="text-sm text-neutral-400">{{ profile.hosts_file || '(订阅远程)' }}</span>
      </div>

      <BackupPanel v-if="profile.type !== 'remote'" :profile-name="profile.name" @changed="emit('reloadHosts', profile.name)" />

      <!-- Subscription Panel -->
      <div class="rounded-xl border border-neutral-200 bg-white/75 p-3 mb-3" :class="profile.type === 'remote' ? 'border-purple-500/30' : ''">
        <div class="flex items-center justify-between cursor-pointer" @click="showSubPanel = !showSubPanel">
          <div class="text-xs text-neutral-700 flex items-center gap-2">
            <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
            </svg>
            {{ t('subscriptions') }}
            <span v-if="subUrl" class="text-emerald-400/70">●</span>
          </div>
          <svg class="w-3.5 h-3.5 text-neutral-400 transition-transform" :class="showSubPanel ? 'rotate-180' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"/>
          </svg>
        </div>
        <div v-if="showSubPanel" class="mt-2 space-y-2">
          <input
            v-model="subUrl"
            type="url"
            :placeholder="t('subscriptionUrl')"
            class="glass-input text-xs w-full"
            @blur="saveSubscription"
          />
          <div class="flex items-center gap-2">
            <span class="text-xs text-neutral-500">{{ t('refreshIntervalSeconds') }}</span>
            <select
              v-model.number="subInterval"
              class="glass-input text-xs py-1.5"
              @change="saveSubscription"
            >
              <option v-for="opt in SUBSCRIPTION_INTERVAL_OPTIONS" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
            </select>
            <button
              class="glass-button text-[11px] px-2 py-1 text-cyan-300"
              :disabled="!subUrl || subRefreshing"
              @click="refreshSubscription"
            >
              {{ subRefreshing ? '...' : t('manualRefresh') || '刷新' }}
            </button>
          </div>
          <div v-if="subResult" class="text-xs" :class="subResult.status === 'ok' ? 'text-emerald-400' : 'text-red-400'">
            <template v-if="subResult.status === 'ok'">
              {{ subResult.last_fetch ? formatLastFetch(subResult.last_fetch) : '' }}
              新增 {{ subResult.entry_count }} 条
            </template>
            <template v-else>
              {{ subResult.message }}
            </template>
          </div>
        </div>
      </div>

      <div v-if="profile.type !== 'remote'" class="rounded-xl border border-neutral-200 bg-white/75 p-3 max-h-32 overflow-y-auto scrollbar-thin mb-3">
        <div class="flex items-center justify-between mb-2 gap-2">
          <div class="text-xs text-neutral-700">{{ t('rulesPanel') }}</div>
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
        <div v-if="toggleableRules.length === 0" class="text-xs text-neutral-400">{{ t('noHostMappings') }}</div>
        <label
          v-for="rule in filteredToggleableRules"
          :key="`${rule.lineNo}-${rule.domain}`"
          class="flex items-center gap-2 text-xs text-neutral-800 py-1"
        >
          <input
            type="checkbox"
            :checked="rule.enabled"
            @change="toggleRule(rule.lineNo, ($event.target as HTMLInputElement).checked)"
          />
          <span class="text-neutral-400">{{ t('line') }} {{ rule.lineNo }}</span>
          <span class="truncate">{{ rule.domain }}</span>
        </label>
      </div>

      <!-- Find Bar (Ctrl+F) -->
      <div v-if="showFind" class="mb-3 p-3 rounded-xl bg-white/70 border border-neutral-200 flex items-center gap-3">
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
        <span v-if="findMatches.length > 0" class="text-sm text-neutral-500">
          {{ t('matchCount', { current: currentMatchIndex + 1, total: findMatches.length }) }}
        </span>
        <span v-else-if="findQuery" class="text-sm text-neutral-400">
          {{ t('noMatches') }}
        </span>
        <button class="glass-button text-sm text-neutral-700" @click="findPrev" :disabled="findMatches.length === 0">
          {{ t('findPrev') }}
        </button>
        <button class="glass-button text-sm text-neutral-700" @click="findNext" :disabled="findMatches.length === 0">
          {{ t('findNext') }}
        </button>
        <button class="glass-button text-sm text-neutral-500" @click="showFind = false">
          {{ t('close') }}
        </button>
      </div>

      <div class="relative flex-1 flex min-h-[320px]">
        <div
          ref="lineNumberRef"
          class="w-12 rounded-l-xl border border-r-0 border-neutral-200 bg-white/75 text-right pr-2 pt-4 text-xs font-mono text-neutral-400 overflow-hidden"
        >
          <div v-for="ln in lineNumbers" :key="`ln-${ln}`" class="leading-6">{{ ln }}</div>
        </div>
        <textarea
          ref="textareaRef"
          v-model="editedText"
          class="w-full h-full rounded-r-xl rounded-l-none bg-white/90 border border-neutral-200 text-neutral-900 p-4 font-mono text-sm leading-6 outline-none focus:border-blue-400/60 resize-none scrollbar-thin"
          :class="profile.type === 'remote' ? 'text-neutral-500 cursor-default' : ''"
          spellcheck="false"
          :readonly="profile.type === 'remote'"
          @input="onEdit"
          @keydown="handleKeydown"
          @scroll="onEditorScroll"
        />
      </div>
    </div>
  </div>
</template>
