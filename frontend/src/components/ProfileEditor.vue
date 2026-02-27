<script setup lang="ts">
import { ref, watch } from 'vue'

interface Profile {
  name: string
  listen_ip: string
  port: number
  running: boolean
  hosts_file: string
}

interface HostEntry {
  domain: string
  ip: string
}

const props = defineProps<{
  profile: Profile
  hosts: HostEntry[]
}>()

const emit = defineEmits<{
  updateHosts: [entries: HostEntry[]]
  delete: [name: string]
  start: [name: string]
  stop: [name: string]
}>()

const editedHosts = ref<HostEntry[]>([])
const newDomain = ref('')
const newIP = ref('')
const hasChanges = ref(false)

watch(() => props.hosts, (newHosts) => {
  editedHosts.value = [...newHosts]
  hasChanges.value = false
}, { immediate: true, deep: true })

function addHost() {
  if (newDomain.value && newIP.value) {
    editedHosts.value.push({
      domain: newDomain.value.trim(),
      ip: newIP.value.trim()
    })
    newDomain.value = ''
    newIP.value = ''
    hasChanges.value = true
  }
}

function removeHost(index: number) {
  editedHosts.value.splice(index, 1)
  hasChanges.value = true
}

function saveChanges() {
  emit('updateHosts', editedHosts.value)
  hasChanges.value = false
}

function confirmDelete() {
  if (confirm(`Delete profile "${props.profile.name}"?`)) {
    emit('delete', props.profile.name)
  }
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
                {{ profile.running ? 'Running' : 'Stopped' }}
              </span>
            </p>
          </div>
        </div>

        <div class="flex items-center gap-2">
          <button
            class="glass-button"
            :class="profile.running 
              ? 'text-red-300 hover:bg-red-500/20'
              : 'text-emerald-300 hover:bg-emerald-500/20'"
            @click="profile.running ? emit('stop', profile.name) : emit('start', profile.name)"
          >
            {{ profile.running ? 'Stop' : 'Start' }}
          </button>
          <button
            class="glass-button text-red-400 hover:bg-red-500/20"
            :disabled="profile.running"
            :class="{ 'opacity-50 cursor-not-allowed': profile.running }"
            @click="confirmDelete"
          >
            Delete
          </button>
        </div>
      </div>
    </div>

    <!-- Hosts Editor -->
    <div class="flex-1 p-5 overflow-y-auto scrollbar-thin">
      <div class="flex items-center justify-between mb-4">
        <h3 class="text-white/80 font-medium">Host Mappings</h3>
        <span class="text-sm text-white/40">{{ editedHosts.length }} entries</span>
      </div>

      <!-- Add new host -->
      <div class="flex gap-3 mb-4">
        <input
          v-model="newDomain"
          type="text"
          placeholder="Domain (e.g. api.example.com)"
          class="glass-input flex-1"
          @keyup.enter="addHost"
        />
        <input
          v-model="newIP"
          type="text"
          placeholder="IP (e.g. 10.0.0.1)"
          class="glass-input w-40"
          @keyup.enter="addHost"
        />
        <button 
          class="glass-button text-blue-300 hover:bg-blue-500/20"
          @click="addHost"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
          </svg>
        </button>
      </div>

      <!-- Hosts list -->
      <div class="space-y-2">
        <div
          v-for="(entry, index) in editedHosts"
          :key="index"
          class="flex items-center gap-3 p-3 rounded-xl bg-white/5 border border-white/10 group hover:bg-white/10 transition-all"
        >
          <div class="flex-1 flex items-center gap-3">
            <span class="text-white/80 font-mono text-sm">{{ entry.domain }}</span>
            <svg class="w-4 h-4 text-white/30" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3"/>
            </svg>
            <span class="text-blue-300 font-mono text-sm">{{ entry.ip }}</span>
          </div>
          <button
            class="opacity-0 group-hover:opacity-100 p-1.5 rounded-lg text-red-400 hover:bg-red-500/20 transition-all"
            @click="removeHost(index)"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
            </svg>
          </button>
        </div>

        <div 
          v-if="editedHosts.length === 0"
          class="text-center py-8 text-white/40"
        >
          <p>No host mappings</p>
          <p class="text-sm mt-1">Add a domain → IP mapping above</p>
        </div>
      </div>
    </div>

    <!-- Footer: Save -->
    <div v-if="hasChanges" class="p-4 border-t border-white/10 bg-blue-500/10">
      <div class="flex items-center justify-between">
        <span class="text-blue-300 text-sm">You have unsaved changes</span>
        <button
          class="glass-button bg-blue-500/30 text-blue-200 hover:bg-blue-500/40 border-blue-400/30"
          @click="saveChanges"
        >
          Save Changes
        </button>
      </div>
    </div>
  </div>
</template>
