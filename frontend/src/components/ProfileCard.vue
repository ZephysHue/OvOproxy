<script setup lang="ts">
import { t } from '../i18n'

interface Profile {
  name: string
  listen_ip: string
  port: number
  running: boolean
  system_hosts_active?: boolean
  proxy_active?: boolean
  proxy_error?: string
}

const props = defineProps<{
  profile: Profile
  active: boolean
}>()

const emit = defineEmits<{
  click: []
  start: [name: string]
  stop: [name: string]
}>()

function handleToggle(e: Event) {
  e.stopPropagation()
  if (props.profile.running) {
    emit('stop', props.profile.name)
  } else {
    emit('start', props.profile.name)
  }
}
</script>

<template>
  <div 
    class="profile-card"
    :class="{ active }"
    @click="emit('click')"
  >
    <div class="flex items-start justify-between">
      <div class="flex items-center gap-3">
        <div class="flex flex-col gap-1">
          <div 
            class="status-dot"
            :class="profile.proxy_active ? 'status-running' : 'status-stopped'"
            :title="profile.proxy_active ? t('proxyActive') : (profile.proxy_error || t('proxyError'))"
          />
        </div>
        <div>
          <h3 class="font-medium text-white/90 flex items-center gap-2">
            {{ profile.name }}
            <span 
              v-if="profile.system_hosts_active" 
              class="text-xs px-1.5 py-0.5 rounded bg-blue-500/30 text-blue-300"
            >
              Hosts
            </span>
          </h3>
          <p class="text-sm text-white/50 mt-0.5">
            {{ profile.listen_ip }}:{{ profile.port }}
          </p>
          <p v-if="profile.proxy_error" class="text-xs text-red-400 mt-0.5">
            {{ profile.proxy_error }}
          </p>
        </div>
      </div>

      <button
        class="px-3 py-1.5 rounded-lg text-xs font-medium transition-all duration-200"
        :class="profile.system_hosts_active 
          ? 'bg-red-500/20 text-red-300 hover:bg-red-500/30 border border-red-500/30'
          : 'bg-emerald-500/20 text-emerald-300 hover:bg-emerald-500/30 border border-emerald-500/30'"
        :disabled="!profile.proxy_active && !profile.system_hosts_active"
        :title="!profile.proxy_active && !profile.system_hosts_active ? t('proxyNotActive') : ''"
        @click="handleToggle"
      >
        {{ profile.system_hosts_active ? t('disableHosts') : t('enableHosts') }}
      </button>
    </div>
  </div>
</template>
