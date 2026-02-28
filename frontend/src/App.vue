<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { GetProfiles, StartProfile, StopProfile, AddProfile, DeleteProfile, ImportHostsFromDialog, ExportHostsToDialog, DedupHosts, GetHostsText, SetHostsText, RenameProfile, IsAdmin } from '../wailsjs/go/main/App'
import { WindowMinimise, WindowToggleMaximise, Quit } from '../wailsjs/runtime/runtime'
import ProfileCard from './components/ProfileCard.vue'
import ProfileEditor from './components/ProfileEditor.vue'
import AddProfileModal from './components/AddProfileModal.vue'
import SettingsModal from './components/SettingsModal.vue'
import RenameProfileModal from './components/RenameProfileModal.vue'
import { t } from './i18n'

interface Profile {
  name: string
  listen_ip: string
  port: number
  hosts_file: string
  running: boolean
  hosts: Record<string, string>
  duplicate_domains?: Array<{ domain: string; count: number }>
  system_hosts_active?: boolean
}

const profiles = ref<Profile[]>([])
const selectedProfile = ref<Profile | null>(null)
const showAddModal = ref(false)
const showSettings = ref(false)
const showRename = ref(false)
const renameFrom = ref('')
const loading = ref(false)
const hostsText = ref('')

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

async function loadHostsText(name: string) {
  try {
    hostsText.value = await GetHostsText(name)
  } catch (e) {
    hostsText.value = ''
  }
}

async function handleStart(name: string) {
  loading.value = true
  try {
    const admin = await IsAdmin()
    if (!admin) {
      alert('需要管理员权限才能修改系统 hosts 文件。请右键以管理员身份运行。')
      loading.value = false
      return
    }
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
    const admin = await IsAdmin()
    if (!admin) {
      alert('需要管理员权限才能修改系统 hosts 文件。请右键以管理员身份运行。')
      loading.value = false
      return
    }
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
      hostsText.value = ''
    }
    await loadProfiles()
  } catch (e) {
    console.error('Failed to delete:', e)
  }
}

async function handleSaveText(name: string, text: string) {
  try {
    await SetHostsText(name, text)
    await loadProfiles()
    await loadHostsText(name)
  } catch (e) {
    console.error('Failed to save hosts text:', e)
  }
}

async function handleImportHosts(name: string) {
  try {
    await ImportHostsFromDialog(name)
    await loadProfiles()
    await loadHostsText(name)
  } catch (e) {
    console.error('Failed to import hosts:', e)
  }
}

async function handleExportHosts(name: string) {
  try {
    await ExportHostsToDialog(name)
  } catch (e) {
    console.error('Failed to export hosts:', e)
  }
}

async function handleDedup(name: string) {
  try {
    await DedupHosts(name)
    await loadProfiles()
    await loadHostsText(name)
  } catch (e) {
    console.error('Failed to dedup hosts:', e)
  }
}

function openRename(name: string) {
  renameFrom.value = name
  showRename.value = true
}

async function handleRename(newName: string) {
  try {
    await RenameProfile(renameFrom.value, newName)
    showRename.value = false
    await loadProfiles()
    const updated = profiles.value.find(p => p.name === newName)
    if (updated) {
      selectedProfile.value = updated
      await loadHostsText(updated.name)
    }
  } catch (e) {
    console.error('Failed to rename profile:', e)
  }
}

function selectProfile(profile: Profile) {
  selectedProfile.value = profile
  loadHostsText(profile.name)
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
        <span class="text-white/90 font-medium text-sm">{{ t('appTitle') }}</span>
      </div>
      <div class="flex items-center">
        <button class="titlebar-button" @click="showSettings = true" :title="t('settings')">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
          </svg>
        </button>
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
          <h2 class="text-lg font-semibold text-white/90">{{ t('profiles') }}</h2>
          <button 
            class="glass-button text-sm text-blue-300 hover:text-blue-200"
            @click="showAddModal = true"
          >
            <span class="flex items-center gap-1">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
              </svg>
              {{ t('add') }}
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
            <p>{{ t('noProfiles') }}</p>
            <p class="text-sm mt-1">{{ t('clickAddToCreate') }}</p>
          </div>
        </div>
      </div>

      <!-- Right Panel: Profile Editor -->
      <div class="flex-1 overflow-hidden">
        <ProfileEditor
          v-if="selectedProfile"
          :profile="selectedProfile"
          :hosts-text="hostsText"
          :duplicates="selectedProfile.duplicate_domains || []"
          @save-text="handleSaveText"
          @delete="handleDelete"
          @start="handleStart"
          @stop="handleStop"
          @import-hosts="handleImportHosts"
          @export-hosts="handleExportHosts"
          @dedup="handleDedup"
          @rename="openRename"
        />

        <div 
          v-else 
          class="h-full glass-card flex items-center justify-center text-white/40"
        >
          <div class="text-center">
            <svg class="w-16 h-16 mx-auto mb-4 opacity-30" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M15 15l-2 5L9 9l11 4-5 2zm0 0l5 5M7.188 2.239l.777 2.897M5.136 7.965l-2.898-.777M13.95 4.05l-2.122 2.122m-5.657 5.656l-2.12 2.122"/>
            </svg>
            <p class="text-lg">{{ t('selectProfile') }}</p>
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

    <SettingsModal :show="showSettings" @close="showSettings = false" />

    <RenameProfileModal
      :show="showRename"
      :current-name="renameFrom"
      @close="showRename = false"
      @rename="handleRename"
    />
  </div>
</template>
