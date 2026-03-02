<script setup lang="ts">
import { ref, watch } from 'vue'
import { t } from '../i18n'
import { GetProxyLogs } from '../../wailsjs/go/main/App'

interface LogEntry {
  time: string
  method: string
  host: string
  resolved_ip: string
  success: boolean
  error?: string
}

const props = defineProps<{
  profileName: string
  proxyAddress: string
}>()

const logs = ref<LogEntry[]>([])
const loading = ref(false)

async function loadLogs() {
  loading.value = true
  try {
    logs.value = await GetProxyLogs(props.profileName, 100)
  } catch (e) {
    console.error('Failed to load logs:', e)
  }
  loading.value = false
}

watch(() => props.profileName, () => {
  loadLogs()
}, { immediate: true })
</script>

<template>
  <div class="diagnostics-panel">
    <div class="flex items-center justify-between mb-4">
      <h3 class="text-white/80 font-medium flex items-center gap-2">
        {{ t('diagnostics') }}
        <span class="text-sm text-white/50 font-normal">{{ proxyAddress }}</span>
      </h3>
      <button
        class="glass-button text-sm text-white/70 hover:bg-slate-700"
        @click="loadLogs"
        :disabled="loading"
      >
        {{ t('refreshLogs') }}
      </button>
    </div>

    <div class="logs-container">
      <div v-if="logs.length === 0" class="text-center text-white/40 py-8">
        {{ t('noLogs') }}
      </div>
      <table v-else class="w-full text-sm">
        <thead>
          <tr class="text-left text-white/50 border-b border-white/10">
            <th class="py-2 px-2">{{ t('time') }}</th>
            <th class="py-2 px-2">{{ t('method') }}</th>
            <th class="py-2 px-2">{{ t('host') }}</th>
            <th class="py-2 px-2">{{ t('resolvedIP') }}</th>
            <th class="py-2 px-2">{{ t('status') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr 
            v-for="(log, idx) in logs" 
            :key="idx"
            class="border-b border-white/5 hover:bg-white/5"
          >
            <td class="py-1.5 px-2 text-white/60 font-mono text-xs">{{ log.time }}</td>
            <td class="py-1.5 px-2">
              <span 
                class="px-1.5 py-0.5 rounded text-xs"
                :class="log.method === 'CONNECT' ? 'bg-purple-500/20 text-purple-300' : 'bg-blue-500/20 text-blue-300'"
              >
                {{ log.method }}
              </span>
            </td>
            <td class="py-1.5 px-2 text-white/80 font-mono text-xs">{{ log.host }}</td>
            <td class="py-1.5 px-2 font-mono text-xs">
              <span 
                :class="log.resolved_ip === '(direct)' ? 'text-white/40' : 'text-cyan-400'"
              >
                {{ log.resolved_ip }}
              </span>
            </td>
            <td class="py-1.5 px-2">
              <span 
                v-if="log.success"
                class="px-1.5 py-0.5 rounded text-xs bg-emerald-500/20 text-emerald-300"
              >
                {{ t('success') }}
              </span>
              <span 
                v-else
                class="px-1.5 py-0.5 rounded text-xs bg-red-500/20 text-red-300"
                :title="log.error"
              >
                {{ t('failed') }}
              </span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.diagnostics-panel {
  @apply bg-slate-950/50 rounded-xl p-4 border border-slate-700/50;
}

.logs-container {
  @apply max-h-48 overflow-y-auto scrollbar-thin;
}
</style>
