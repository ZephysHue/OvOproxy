<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { GetProfiles, StartProfile, StopProfile, AddProfile, DeleteProfile, UpdateHosts } from '../wailsjs/go/main/App'
import { WindowMinimise, WindowToggleMaximise, Quit } from '../wailsjs/runtime/runtime'
import ProfileCard from './components/ProfileCard.vue'
import ProfileEditor from './components/ProfileEditor.vue'
import AddProfileModal from './components/AddProfileModal.vue'

interface HostEntry {
  domain: string
  ip: string
}

interface Profile {
  name: string
  listen_ip: string
  port: number
  hosts_file: string
  running: boolean
  hosts: Record<string, string>
}

const profiles = ref<Profile[]>([])
const selectedProfile = ref<Profile | null>(null)
const showAddModal = ref(false)
const loading = ref(false)

const selectedHosts = computed<HostEntry[]>(() => {
  if (!selectedProfile.value?.hosts) return []
  return Object.entries(selectedProfile.value.hosts).map(([domain, ip]) => ({
    domain,
    ip
  }))
})

async function loadProfiles() {
  try {
    const data = await GetProfiles()
    profiles.value = data || []
    if (selectedProfile.value) {
      const updated = profiles.value.find(p => p.name === selectedProfile.value?.name)
      if (updated) selectedProfile.value = updated
    }
  } catch (e) {
    console.error('Failed to load profiles:', e)
  }
}

async function handleStart(name: string) {
  loading.value = true
  try {
    await StartProfile(name)
    await loadProfiles()
  } catch (e) {
    console.error('Failed to start:', e)
  }
  loading.value = false
}

async function handleStop(name: string) {
  loading.value = true
  try {
    await StopProfile(name)
    await loadProfiles()
  } catch (e) {
    console.error('Failed to stop:', e)
  }
  loading.value = false
}

async function handleAdd(name: string, ip: string, port: number) {
  try {
    await AddProfile(name, ip, port)
    await loadProfiles()
    showAddModal.value = false
  } catch (e) {
    console.error('Failed to add:', e)
  }
}

async function handleDelete(name: string) {
  try {
    await DeleteProfile(name)
    if (selectedProfile.value?.name === name) {
      selectedProfile.value = null
    }
    await loadProfiles()
  } catch (e) {
    console.error('Failed to delete:', e)
  }
}

async function handleUpdateHosts(entries: HostEntry[]) {
  if (!selectedProfile.value) return
  try {
    await UpdateHosts(selectedProfile.value.name, entries)
    await loadProfiles()
  } catch (e) {
    console.error('Failed to update hosts:', e)
  }
}

function selectProfile(profile: Profile) {
  selectedProfile.value = profile
}

onMounted(() => {
  loadProfiles()
})
</script>

<template>
  <div class="h-full w-full flex flex-col bg-gradient-to-br from-slate-900 via-purple-900/30 to-slate-900">
    <!-- Titlebar -->
    <div class="titlebar">
      <div class="flex items-center gap-3">
        <div class="w-6 h-6 rounded-lg bg-gradient-to-br from-blue-400 to-purple-500 flex items-center justify-center">
          <svg class="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01"/>
          </svg>
        </div>
        <span class="text-white/90 font-medium text-sm">Multi-Host Proxy</span>
      </div>
      <div class="flex items-center">
        <button class="titlebar-button" @click="WindowMinimise">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 12H4"/>
          </svg>
        </button>
        <button class="titlebar-button" @click="WindowToggleMaximise">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5v-4m0 4h-4m4 0l-5-5"/>
          </svg>
        </button>
        <button class="titlebar-button titlebar-close" @click="Quit">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
          </svg>
        </button>
      </div>
    </div>

    <!-- Main Content -->
    <div class="flex-1 flex gap-6 p-6 overflow-hidden">
      <!-- Left Panel: Profile List -->
      <div class="w-80 flex flex-col gap-4">
        <div class="flex items-center justify-between">
          <h2 class="text-lg font-semibold text-white/90">Profiles</h2>
          <button 
            class="glass-button text-sm text-blue-300 hover:text-blue-200"
            @click="showAddModal = true"
          >
            <span class="flex items-center gap-1">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
              </svg>
              Add
            </span>
          </button>
        </div>

        <div class="flex-1 overflow-y-auto scrollbar-thin space-y-3 pr-2">
          <ProfileCard
            v-for="profile in profiles"
            :key="profile.name"
            :profile="profile"
            :active="selectedProfile?.name === profile.name"
            @click="selectProfile(profile)"
            @start="handleStart"
            @stop="handleStop"
          />

          <div 
            v-if="profiles.length === 0"
            class="glass-card p-8 text-center text-white/50"
          >
            <svg class="w-12 h-12 mx-auto mb-3 opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"/>
            </svg>
            <p>No profiles yet</p>
            <p class="text-sm mt-1">Click "Add" to create one</p>
          </div>
        </div>
      </div>

      <!-- Right Panel: Profile Editor -->
      <div class="flex-1 overflow-hidden">
        <ProfileEditor
          v-if="selectedProfile"
          :profile="selectedProfile"
          :hosts="selectedHosts"
          @update-hosts="handleUpdateHosts"
          @delete="handleDelete"
          @start="handleStart"
          @stop="handleStop"
        />

        <div 
          v-else 
          class="h-full glass-card flex items-center justify-center text-white/40"
        >
          <div class="text-center">
            <svg class="w-16 h-16 mx-auto mb-4 opacity-30" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M15 15l-2 5L9 9l11 4-5 2zm0 0l5 5M7.188 2.239l.777 2.897M5.136 7.965l-2.898-.777M13.95 4.05l-2.122 2.122m-5.657 5.656l-2.12 2.122"/>
            </svg>
            <p class="text-lg">Select a profile to edit</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Add Profile Modal -->
    <AddProfileModal
      :show="showAddModal"
      @close="showAddModal = false"
      @add="handleAdd"
    />
  </div>
</template>
