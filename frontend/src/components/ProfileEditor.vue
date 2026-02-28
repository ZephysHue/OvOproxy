<script setup lang="ts">
import { ref, watch } from 'vue'
import { t } from '../i18n'

interface Profile {
  name: string
  listen_ip: string
  port: number
  running: boolean
  hosts_file: string
  system_hosts_active?: boolean
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
}>()

const editedText = ref('')
const hasChanges = ref(false)

watch(() => props.hostsText, (v) => {
  editedText.value = v || ''
  hasChanges.value = false
}, { immediate: true })

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
            :class="profile.running 
              ? 'bg-emerald-500/20 border border-emerald-500/30' 
              : 'bg-white/10 border border-white/20'"
          >
            <svg class="w-6 h-6" :class="profile.running ? 'text-emerald-400' : 'text-white/60'" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2"/>
            </svg>
          </div>
          <div>
            <h2 class="text-xl font-semibold text-white/90">{{ profile.name }}</h2>
            <p class="text-sm text-white/50 mt-0.5">
              {{ profile.listen_ip }}:{{ profile.port }}
              <span 
                class="ml-2 px-2 py-0.5 rounded-full text-xs"
                :class="profile.running 
                  ? 'bg-emerald-500/20 text-emerald-300' 
                  : 'bg-white/10 text-white/50'"
              >
                {{ profile.running ? t('running') : t('stopped') }}
              </span>
            </p>
          </div>
        </div>

        <div class="flex items-center gap-2">
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
            :disabled="profile.running"
            :class="{ 'opacity-50 cursor-not-allowed': profile.running }"
          >
            {{ t('rename') }}
          </button>
          <button
            class="glass-button"
            :class="profile.running 
              ? 'text-red-300 hover:bg-red-500/20'
              : 'text-emerald-300 hover:bg-emerald-500/20'"
            @click="profile.running ? emit('stop', profile.name) : emit('start', profile.name)"
          >
            {{ profile.running ? t('stop') : t('start') }}
          </button>
          <button
            class="glass-button text-red-400 hover:bg-red-500/20"
            :disabled="profile.running"
            :class="{ 'opacity-50 cursor-not-allowed': profile.running }"
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

      <div class="flex items-center justify-between mb-4">
        <h3 class="text-white/80 font-medium">{{ t('hostMappings') }}</h3>
        <span class="text-sm text-white/40">{{ profile.hosts_file }}</span>
      </div>

      <textarea
        v-model="editedText"
        class="flex-1 w-full rounded-xl bg-slate-950 border border-slate-700/70 text-white/90 p-4 font-mono text-sm leading-6 outline-none focus:border-blue-400/60 resize-none scrollbar-thin"
        spellcheck="false"
        @input="onEdit"
      />
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
