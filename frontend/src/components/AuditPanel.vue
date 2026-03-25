<script setup lang="ts">
import { ref, watch } from 'vue'
import { t } from '../i18n'
import { GetAuditLogs } from '../../wailsjs/go/main/App'

interface AuditLogEntry {
  time: string
  action: string
  profile: string
  detail: string
  success: boolean
}

const props = defineProps<{ profileName: string }>()
const logs = ref<AuditLogEntry[]>([])
const busy = ref(false)

async function loadLogs() {
  busy.value = true
  try {
    const all = await GetAuditLogs(120)
    logs.value = (all || []).filter((l) => !props.profileName || l.profile === '' || l.profile === props.profileName)
  } catch (e) {
    console.error('load audit logs failed', e)
  } finally {
    busy.value = false
  }
}

watch(() => props.profileName, loadLogs, { immediate: true })
</script>

<template>
  <div class="rounded-xl border border-slate-700/60 bg-slate-900 p-3 mb-4">
    <div class="flex items-center justify-between mb-2">
      <div class="text-sm text-white/80">{{ t('auditLogs') }}</div>
      <button class="glass-button text-xs text-cyan-200" :disabled="busy" @click="loadLogs">
        {{ t('refreshLogs') }}
      </button>
    </div>
    <div class="max-h-44 overflow-y-auto scrollbar-thin space-y-1">
      <div v-if="logs.length === 0" class="text-xs text-white/40">{{ t('noLogs') }}</div>
      <div
        v-for="log in logs"
        :key="`${log.time}-${log.action}-${log.detail}`"
        class="rounded border border-slate-700/40 bg-slate-800/50 p-2 text-xs"
      >
        <div class="text-white/70">
          {{ log.time }} · {{ log.action }} ·
          <span :class="log.success ? 'text-emerald-300' : 'text-red-300'">
            {{ log.success ? t('success') : t('failed') }}
          </span>
        </div>
        <div class="text-white/50">{{ log.profile || '-' }}</div>
        <div class="text-white/60 break-all">{{ log.detail || '-' }}</div>
      </div>
    </div>
  </div>
</template>
