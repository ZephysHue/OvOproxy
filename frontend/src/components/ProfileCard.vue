<script setup lang="ts">
import { t } from '../i18n'

interface Profile {
  name: string
  listen_ip: string
  port: number
  running: boolean
  system_hosts_active?: boolean
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
        <div 
          class="status-dot"
          :class="profile.running ? 'status-running' : 'status-stopped'"
        />
        <div>
          <h3 class="font-medium text-white/90">{{ profile.name }}</h3>
          <p class="text-sm text-white/50 mt-0.5">
            {{ profile.listen_ip }}:{{ profile.port }}
          </p>
        </div>
      </div>

      <button
        class="px-3 py-1.5 rounded-lg text-xs font-medium transition-all duration-200"
        :class="profile.running 
          ? 'bg-red-500/20 text-red-300 hover:bg-red-500/30 border border-red-500/30'
          : 'bg-emerald-500/20 text-emerald-300 hover:bg-emerald-500/30 border border-emerald-500/30'"
        @click="handleToggle"
      >
        {{ profile.running ? t('stop') : t('start') }}
      </button>
    </div>
  </div>
</template>
