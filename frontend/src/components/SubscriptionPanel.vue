<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { t } from '../i18n'
import {
  GetProfileSubscriptions,
  GetSubscriptionRefreshSettings,
  UpdateSubscriptionRefreshSettings,
  GetSubscriptionRefreshHistory,
  IsSubscriptionRefreshRunning,
  AddProfileSubscription,
  RemoveProfileSubscription,
  SetProfileSubscriptionEnabled,
  SetAllProfileSubscriptionsEnabled,
  UpdateProfileSubscription,
  RefreshProfileSubscriptionsWithReport,
  RefreshSingleProfileSubscription,
  RetryFailedProfileSubscriptions,
  PreviewSubscriptionConflicts,
  ResolveSubscriptionConflicts,
} from '../../wailsjs/go/main/App'

interface Subscription {
  id: string
  name: string
  url: string
  enabled: boolean
  last_updated?: string
  last_status?: string
}

interface ConflictPreview {
  sub_id: string
  sub_name: string
  domains: string[]
  items?: Array<{
    domain: string
    local_ip: string
    remote_ip: string
  }>
  total: number
  truncate: boolean
}

interface RefreshFailure {
  sub_id: string
  sub_name: string
  reason: string
}

interface RefreshReport {
  time: string
  source?: string
  success?: boolean
  enabled_total: number
  success_total: number
  failed_total: number
  added_total: number
  conflict_diff: number
  conflict_same: number
  failures: RefreshFailure[]
}

interface RefreshSettings {
  auto_enabled: boolean
  interval_seconds: number
  max_backoff_seconds: number
  history_limit: number
}

const props = defineProps<{ profileName: string }>()
const emit = defineEmits<{ changed: [] }>()

const subs = ref<Subscription[]>([])
const addingName = ref('')
const addingURL = ref('')
const busy = ref(false)
const subBusyId = ref('')
const previews = ref<Record<string, ConflictPreview | undefined>>({})
const refreshReport = ref<RefreshReport | null>(null)
const subFilter = ref('')
const editingId = ref('')
const editName = ref('')
const editURL = ref('')
const refreshSettings = ref<RefreshSettings>({
  auto_enabled: false,
  interval_seconds: 600,
  max_backoff_seconds: 900,
  history_limit: 20,
})
const runningAutoRefresh = ref(false)
const historyExpanded = ref(false)
const refreshHistory = ref<RefreshReport[]>([])

const filteredSubs = computed(() => {
  const q = subFilter.value.trim().toLowerCase()
  if (!q) return subs.value
  return subs.value.filter((s) => {
    const name = (s.name || '').toLowerCase()
    const url = (s.url || '').toLowerCase()
    return name.includes(q) || url.includes(q)
  })
})

async function loadSubs() {
  try {
    subs.value = await GetProfileSubscriptions(props.profileName)
  } catch (e) {
    console.error('load subscriptions failed', e)
  }
}

async function loadRefreshSettings() {
  try {
    const s = await GetSubscriptionRefreshSettings(props.profileName)
    refreshSettings.value = {
      auto_enabled: !!s.auto_enabled,
      interval_seconds: Number(s.interval_seconds || 600),
      max_backoff_seconds: Number(s.max_backoff_seconds || 900),
      history_limit: Number(s.history_limit || 20),
    }
  } catch (e) {
    console.error('load refresh settings failed', e)
  }
}

async function saveRefreshSettings() {
  busy.value = true
  try {
    await UpdateSubscriptionRefreshSettings(
      props.profileName,
      !!refreshSettings.value.auto_enabled,
      Number(refreshSettings.value.interval_seconds || 0),
      Number(refreshSettings.value.max_backoff_seconds || 0),
      Number(refreshSettings.value.history_limit || 0),
    )
    await loadRefreshSettings()
  } catch (e) {
    alert(String(e))
  } finally {
    busy.value = false
  }
}

async function loadRefreshHistory() {
  try {
    refreshHistory.value = await GetSubscriptionRefreshHistory(
      props.profileName,
      Number(refreshSettings.value.history_limit || 20),
    )
    runningAutoRefresh.value = await IsSubscriptionRefreshRunning(props.profileName)
  } catch (e) {
    console.error('load refresh history failed', e)
  }
}

async function addSub() {
  if (!addingURL.value.trim()) return
  busy.value = true
  try {
    await AddProfileSubscription(props.profileName, addingName.value.trim(), addingURL.value.trim())
    addingName.value = ''
    addingURL.value = ''
    await loadSubs()
  } catch (e) {
    alert(String(e))
  } finally {
    busy.value = false
  }
}

async function removeSub(id: string) {
  busy.value = true
  try {
    await RemoveProfileSubscription(props.profileName, id)
    await loadSubs()
  } catch (e) {
    alert(String(e))
  } finally {
    busy.value = false
  }
}

async function toggleSub(id: string, enabled: boolean) {
  busy.value = true
  try {
    await SetProfileSubscriptionEnabled(props.profileName, id, enabled)
    await loadSubs()
  } catch (e) {
    alert(String(e))
  } finally {
    busy.value = false
  }
}

async function setAllSubsEnabled(enabled: boolean) {
  busy.value = true
  try {
    await SetAllProfileSubscriptionsEnabled(props.profileName, enabled)
    await loadSubs()
  } catch (e) {
    alert(String(e))
  } finally {
    busy.value = false
  }
}

async function manualRefresh() {
  busy.value = true
  try {
    refreshReport.value = await RefreshProfileSubscriptionsWithReport(props.profileName)
    await loadRefreshHistory()
    await loadSubs()
    emit('changed')
  } catch (e) {
    alert(String(e))
  } finally {
    busy.value = false
  }
}

async function retryFailedSubs() {
  busy.value = true
  try {
    refreshReport.value = await RetryFailedProfileSubscriptions(props.profileName)
    await loadRefreshHistory()
    await loadSubs()
    emit('changed')
  } catch (e) {
    alert(String(e))
  } finally {
    busy.value = false
  }
}

async function manualRefreshOne(subId: string) {
  subBusyId.value = subId
  try {
    await RefreshSingleProfileSubscription(props.profileName, subId)
    await loadRefreshHistory()
    await loadSubs()
    emit('changed')
  } catch (e) {
    alert(String(e))
  } finally {
    subBusyId.value = ''
  }
}

async function previewConflicts(subId: string) {
  subBusyId.value = subId
  try {
    const preview = await PreviewSubscriptionConflicts(props.profileName, subId)
    previews.value[subId] = preview
  } catch (e) {
    alert(String(e))
  } finally {
    subBusyId.value = ''
  }
}

async function resolveConflicts(subId: string, strategy: 'use_remote' | 'keep_local') {
  subBusyId.value = subId
  try {
    await ResolveSubscriptionConflicts(props.profileName, subId, strategy)
    if (strategy === 'use_remote') {
      await loadSubs()
      emit('changed')
    }
    const preview = await PreviewSubscriptionConflicts(props.profileName, subId)
    previews.value[subId] = preview
  } catch (e) {
    alert(String(e))
  } finally {
    subBusyId.value = ''
  }
}

function exportConflictDetails(sub: Subscription) {
  const preview = previews.value[sub.id]
  if (!preview || !preview.items || preview.items.length === 0) {
    return
  }
  const lines = [
    `Profile: ${props.profileName}`,
    `Subscription: ${sub.name || sub.url}`,
    `URL: ${sub.url}`,
    `Generated: ${new Date().toISOString()}`,
    '',
    'Domain,LocalIP,RemoteIP',
  ]
  for (const item of preview.items) {
    lines.push(`${item.domain},${item.local_ip},${item.remote_ip}`)
  }
  const blob = new Blob([lines.join('\n')], { type: 'text/csv;charset=utf-8' })
  const a = document.createElement('a')
  a.href = URL.createObjectURL(blob)
  const safeName = (sub.name || 'subscription').replace(/[^\w.-]+/g, '_')
  a.download = `conflicts_${safeName}.csv`
  a.click()
  URL.revokeObjectURL(a.href)
}

function beginEdit(sub: Subscription) {
  editingId.value = sub.id
  editName.value = sub.name || ''
  editURL.value = sub.url || ''
}

function cancelEdit() {
  editingId.value = ''
  editName.value = ''
  editURL.value = ''
}

async function saveEdit(subId: string) {
  subBusyId.value = subId
  try {
    await UpdateProfileSubscription(props.profileName, subId, editName.value.trim(), editURL.value.trim())
    await loadSubs()
    cancelEdit()
  } catch (e) {
    alert(String(e))
  } finally {
    subBusyId.value = ''
  }
}

watch(() => props.profileName, loadSubs, { immediate: true })
watch(
  () => props.profileName,
  async () => {
    await loadRefreshSettings()
    await loadRefreshHistory()
  },
  { immediate: true },
)
</script>

<template>
  <div class="rounded-xl border border-slate-700/60 bg-slate-900 p-3 mb-4">
    <div class="flex items-center justify-between mb-3">
      <div class="text-sm text-white/80">{{ t('subscriptions') }}</div>
      <div class="flex items-center gap-2">
        <button class="glass-button text-xs text-slate-200" :disabled="busy" @click="setAllSubsEnabled(true)">
          {{ t('enableAll') }}
        </button>
        <button class="glass-button text-xs text-slate-200" :disabled="busy" @click="setAllSubsEnabled(false)">
          {{ t('disableAll') }}
        </button>
        <button
          class="glass-button text-xs text-amber-200"
          :disabled="busy || !refreshReport || refreshReport.failed_total === 0"
          @click="retryFailedSubs"
        >
          {{ t('retryFailedSubscriptions') }}
        </button>
        <button class="glass-button text-xs text-cyan-200" :disabled="busy" @click="manualRefresh">
          {{ t('manualRefresh') }}
        </button>
      </div>
    </div>
    <div class="rounded-lg border border-slate-700/40 bg-slate-800/50 px-2 py-2 text-[11px] text-white/70 mb-3">
      <div class="flex items-center justify-between gap-2">
        <div class="text-white/80">{{ t('autoRefreshSettings') }}</div>
        <div class="text-[11px] text-cyan-200">
          {{ runningAutoRefresh ? t('autoRefreshRunning') : t('autoRefreshIdle') }}
        </div>
      </div>
      <div class="grid grid-cols-12 gap-2 mt-2">
        <label class="col-span-2 flex items-center gap-1">
          <input v-model="refreshSettings.auto_enabled" type="checkbox" />
          {{ t('enabled') }}
        </label>
        <input
          v-model.number="refreshSettings.interval_seconds"
          class="glass-input text-xs col-span-3 py-1"
          type="number"
          min="30"
          :placeholder="t('refreshIntervalSeconds')"
        />
        <input
          v-model.number="refreshSettings.max_backoff_seconds"
          class="glass-input text-xs col-span-3 py-1"
          type="number"
          min="30"
          :placeholder="t('maxBackoffSeconds')"
        />
        <input
          v-model.number="refreshSettings.history_limit"
          class="glass-input text-xs col-span-2 py-1"
          type="number"
          min="5"
          :placeholder="t('historyLimit')"
        />
        <button class="glass-button text-xs col-span-2" :disabled="busy" @click="saveRefreshSettings">
          {{ t('save') }}
        </button>
      </div>
    </div>
    <div v-if="refreshReport" class="rounded-lg border border-slate-700/40 bg-slate-800/50 px-2 py-2 text-[11px] text-white/70 mb-3">
      <div>
        {{ t('refreshSummary') }}: {{ t('enabled') }} {{ refreshReport.enabled_total }} ·
        {{ t('success') }} {{ refreshReport.success_total }} ·
        {{ t('failed') }} {{ refreshReport.failed_total }} ·
        {{ t('addedCount', { count: refreshReport.added_total }) }} ·
        {{ t('conflictDiffCount', { count: refreshReport.conflict_diff }) }} ·
        {{ t('conflictSameCount', { count: refreshReport.conflict_same }) }}
      </div>
      <div class="mt-1 text-white/50">{{ t('time') }}: {{ refreshReport.time }}</div>
      <div v-if="refreshReport.failures.length > 0" class="mt-1 space-y-1">
        <div v-for="f in refreshReport.failures" :key="f.sub_id" class="text-red-300/90">
          {{ f.sub_name || f.sub_id }}: {{ f.reason }}
        </div>
      </div>
    </div>
    <div class="rounded-lg border border-slate-700/40 bg-slate-800/50 px-2 py-2 text-[11px] text-white/70 mb-3">
      <div class="flex items-center justify-between">
        <button class="glass-button text-[11px] text-slate-200 px-2 py-1" @click="historyExpanded = !historyExpanded">
          {{ historyExpanded ? t('collapseHistory') : t('expandHistory') }}
        </button>
        <button class="glass-button text-[11px] text-cyan-200 px-2 py-1" :disabled="busy" @click="loadRefreshHistory">
          {{ t('refreshHistory') }}
        </button>
      </div>
      <div v-if="historyExpanded" class="mt-2 space-y-1">
        <div v-if="refreshHistory.length === 0" class="text-white/50">{{ t('noRefreshHistory') }}</div>
        <div
          v-for="h in refreshHistory"
          :key="`${h.time}-${h.source}`"
          class="rounded border border-slate-700/40 px-2 py-1"
        >
          <div class="text-white/80">
            {{ h.time }} · {{ h.source || '-' }} · {{ h.success ? t('success') : t('failed') }}
          </div>
          <div class="text-white/60">
            {{ t('enabled') }} {{ h.enabled_total }} · {{ t('success') }} {{ h.success_total }} ·
            {{ t('failed') }} {{ h.failed_total }} · {{ t('addedCount', { count: h.added_total }) }}
          </div>
        </div>
      </div>
    </div>
    <div class="grid grid-cols-12 gap-2 mb-3">
      <input
        v-model="addingName"
        class="glass-input text-xs col-span-3 py-2"
        :placeholder="t('subscriptionName')"
      />
      <input
        v-model="addingURL"
        class="glass-input text-xs col-span-7 py-2"
        :placeholder="t('subscriptionUrl')"
      />
      <button class="glass-button text-xs col-span-2" :disabled="busy" @click="addSub">
        {{ t('addSubscription') }}
      </button>
    </div>
    <div class="mb-3">
      <input
        v-model="subFilter"
        class="glass-input text-xs py-2 w-full"
        :placeholder="t('subscriptionFilterPlaceholder')"
      />
    </div>
    <div class="max-h-36 overflow-y-auto scrollbar-thin space-y-2">
      <div v-if="subs.length === 0" class="text-xs text-white/40">{{ t('noProfiles') }}</div>
      <div
        v-for="sub in filteredSubs"
        :key="sub.id"
        class="rounded-lg border border-slate-700/50 bg-slate-800/60 px-2 py-2 text-xs"
      >
        <div class="flex items-center justify-between gap-2">
          <div class="min-w-0">
            <template v-if="editingId === sub.id">
              <input
                v-model="editName"
                class="glass-input text-xs py-1 w-full mb-1"
                :placeholder="t('subscriptionName')"
              />
              <input
                v-model="editURL"
                class="glass-input text-xs py-1 w-full font-mono"
                :placeholder="t('subscriptionUrl')"
              />
            </template>
            <template v-else>
              <div class="text-white/90 truncate">{{ sub.name || sub.url }}</div>
              <div class="text-white/50 truncate font-mono">{{ sub.url }}</div>
            </template>
          </div>
          <div class="flex items-center gap-2">
            <label class="flex items-center gap-1 text-white/70">
              <input
                type="checkbox"
                :checked="sub.enabled"
                @change="toggleSub(sub.id, ($event.target as HTMLInputElement).checked)"
              />
              {{ sub.enabled ? t('enabled') : t('disabled') }}
            </label>
            <button
              v-if="editingId !== sub.id"
              class="glass-button text-[11px] text-slate-200 px-2 py-1"
              :disabled="busy || !!subBusyId"
              @click="beginEdit(sub)"
            >
              {{ t('editSubscription') }}
            </button>
            <button
              v-else
              class="glass-button text-[11px] text-emerald-200 px-2 py-1"
              :disabled="busy || !!subBusyId"
              @click="saveEdit(sub.id)"
            >
              {{ t('save') }}
            </button>
            <button
              v-if="editingId === sub.id"
              class="glass-button text-[11px] text-slate-200 px-2 py-1"
              :disabled="busy || !!subBusyId"
              @click="cancelEdit"
            >
              {{ t('cancel') }}
            </button>
            <button
              class="glass-button text-[11px] text-cyan-200 px-2 py-1"
              :disabled="busy || !!subBusyId"
              @click="manualRefreshOne(sub.id)"
            >
              {{ t('refreshThisSubscription') }}
            </button>
            <button
              class="glass-button text-[11px] text-amber-200 px-2 py-1"
              :disabled="busy || !!subBusyId"
              @click="previewConflicts(sub.id)"
            >
              {{ t('conflictPreview') }}
            </button>
            <button class="glass-button text-[11px] text-red-200 px-2 py-1" @click="removeSub(sub.id)">
              {{ t('remove') }}
            </button>
          </div>
        </div>
        <div class="mt-1 text-white/40">
          {{ t('lastStatus') }}: {{ sub.last_status || '-' }} · {{ t('lastUpdated') }}: {{ sub.last_updated || '-' }}
        </div>
        <div v-if="previews[sub.id]" class="mt-1 text-[11px] text-amber-200/90">
          {{ t('conflictCount', { count: previews[sub.id]!.total }) }}
          <span v-if="previews[sub.id]!.truncate"> · {{ t('previewTruncated') }}</span>
          <div
            v-for="item in previews[sub.id]!.items || []"
            :key="item.domain"
            class="mt-1 font-mono text-[11px] text-white/70"
          >
            {{ item.domain }}: {{ item.local_ip }} -> {{ item.remote_ip }}
          </div>
          <div class="mt-2 flex items-center gap-2">
            <button
              class="glass-button text-[11px] text-emerald-200 px-2 py-1"
              :disabled="busy || !!subBusyId || previews[sub.id]!.total === 0"
              @click="resolveConflicts(sub.id, 'use_remote')"
            >
              {{ t('useSubscriptionValue') }}
            </button>
            <button
              class="glass-button text-[11px] text-slate-200 px-2 py-1"
              :disabled="busy || !!subBusyId || previews[sub.id]!.total === 0"
              @click="resolveConflicts(sub.id, 'keep_local')"
            >
              {{ t('keepLocalValue') }}
            </button>
            <button
              class="glass-button text-[11px] text-cyan-200 px-2 py-1"
              :disabled="busy || !!subBusyId || previews[sub.id]!.total === 0"
              @click="exportConflictDetails(sub)"
            >
              {{ t('exportConflictDetails') }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
