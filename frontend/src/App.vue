<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { GetProfiles, StartProfile, StopProfile, AddProfile, AddRemoteProfile, DeleteProfile, ExportHostsToDialog, GetHostsText, SetHostsText, RenameProfile, IsAdmin, GetProxyAddress, RelaunchAsAdmin, CheckUpdate, DownloadUpdate, ApplyUpdate, GetVersion } from '../wailsjs/go/main/App'
import { WindowMinimise, WindowToggleMaximise, Quit, EventsOn } from '../wailsjs/runtime/runtime'
import ProfileCard from './components/ProfileCard.vue'
import ProfileEditor from './components/ProfileEditor.vue'
import AddProfileModal from './components/AddProfileModal.vue'
import SettingsModal from './components/SettingsModal.vue'
import RenameProfileModal from './components/RenameProfileModal.vue'
import { t } from './i18n'
import type { Profile } from './types'

const profiles = ref<Profile[]>([])
const selectedProfile = ref<Profile | null>(null)
const showAddModal = ref(false)
const showSettings = ref(false)
const showRename = ref(false)
const renameFrom = ref('')
const loading = ref(false)
const hostsText = ref('')
const searchQuery = ref('')
const isAdmin = ref(true)
const showAdminModal = ref(false)
const contextMenu = ref({ show: false, x: 0, y: 0, profileName: '' })

// 更新相关
const updateAvailable = ref(false)
const updateDownloaded = ref(false)
const updateDownloading = ref(false)

const filteredProfiles = computed(() => {
  const q = searchQuery.value.toLowerCase().trim()
  if (!q) return profiles.value
  return profiles.value.filter(p => {
    if (p.name.toLowerCase().includes(q)) return true
    if (String(p.port).includes(q)) return true
    for (const domain of Object.keys(p.hosts || {})) {
      if (domain.toLowerCase().includes(q)) return true
    }
    for (const ip of Object.values(p.hosts || {})) {
      if (ip.includes(q)) return true
    }
    return false
  })
})

const usedPorts = computed(() => profiles.value.map(p => p.port))

async function loadProfiles() {
  try {
    const data = await GetProfiles()
    profiles.value = data || []
    if (selectedProfile.value) {
      const updated = profiles.value.find(p => p.name === selectedProfile.value?.name)
      if (updated) {
        selectedProfile.value = updated
      } else {
        selectedProfile.value = null
        hostsText.value = ''
      }
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
    if (!isAdmin.value) {
      showAdminModal.value = true
      loading.value = false
      return
    }
    await StartProfile(name)
    await loadProfiles()
  } catch (e: any) {
    console.error('Failed to start:', e)
    alert(e?.message || String(e))
  }
  loading.value = false
}

async function handleStop(name: string) {
  loading.value = true
  try {
    if (!isAdmin.value) {
      showAdminModal.value = true
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

async function handleAddRemote(name: string, ip: string, port: number, url: string, interval: number) {
  try {
    await AddRemoteProfile(name, ip, port, url, interval)
    await loadProfiles()
    showAddModal.value = false
  } catch (e) {
    console.error('Failed to add remote:', e)
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

async function handleSaveText(name: string, text: string, _confirmedRisk?: boolean) {
  try {
    await SetHostsText(name, text)
    await loadProfiles()
    await loadHostsText(name)
  } catch (e) {
    console.error('Failed to save hosts text:', e)
  }
}

async function handleRelaunchAsAdmin() {
  try {
    await RelaunchAsAdmin()
  } catch (e: any) {
    alert(e?.message || String(e))
  }
}

async function handleExportHosts(name: string) {
  try {
    await ExportHostsToDialog(name)
  } catch (e) {
    console.error('Failed to export hosts:', e)
  }
}

async function handleReloadHosts(name: string) {
  await loadProfiles()
  await loadHostsText(name)
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
  selectedProfile.value = { ...profile, hosts: { ...profile.hosts } }
  loadHostsText(profile.name)
}

function onProfileContextMenu(event: MouseEvent, profile: Profile) {
  event.preventDefault()
  const menuW = 170; const menuH = 160
  const x = event.clientX + menuW > window.innerWidth ? event.clientX - menuW : event.clientX
  const y = event.clientY + menuH > window.innerHeight ? event.clientY - menuH : event.clientY
  contextMenu.value = {
    show: true,
    x, y,
    profileName: profile.name,
  }
}

function closeContextMenu() {
  contextMenu.value.show = false
}

function handleExportFromMenu() {
  const name = contextMenu.value.profileName
  closeContextMenu()
  handleExportHosts(name)
}

function handleRenameFromMenu() {
  const name = contextMenu.value.profileName
  closeContextMenu()
  openRename(name)
}

function handleDeleteFromMenu() {
  const name = contextMenu.value.profileName
  const profile = profiles.value.find(p => p.name === name)
  closeContextMenu()
  if (profile?.system_hosts_active) return
  if (confirm(`${t('delete')} "${name}"?`)) {
    handleDelete(name)
  }
}

function onDocumentClick() {
  closeContextMenu()
}

async function handleUpdateClick() {
  if (updateDownloaded) {
    await ApplyUpdate()
    return
  }
  if (updateDownloading) return
  updateDownloading.value = true
  try {
    const result = await CheckUpdate()
    if (result.HasUpdate && result.DownloadURL) {
      await DownloadUpdate(result.DownloadURL)
      updateDownloaded.value = true
      updateAvailable.value = true
    } else {
      alert(t('alreadyLatest'))
    }
  } catch (e: any) {
    console.error('Update failed:', e)
    alert(e?.message || String(e))
  }
  updateDownloading.value = false
}

onMounted(() => {
  loadProfiles()
  IsAdmin().then(v => { isAdmin.value = !!v }).catch(() => { isAdmin.value = false })
  EventsOn('profiles:changed', () => {
    loadProfiles()
    if (selectedProfile.value) {
      loadHostsText(selectedProfile.value.name)
    }
  })
  EventsOn('update:available', () => { updateAvailable.value = true })
  EventsOn('update:downloaded', () => { updateDownloaded.value = true; updateAvailable.value = true })
  document.addEventListener('click', onDocumentClick)
})

onUnmounted(() => {
  document.removeEventListener('click', onDocumentClick)
})
</script>

<template>
  <div class="app-shell h-full w-full flex flex-col">
    <!-- Titlebar -->
    <div class="titlebar">
      <div class="flex items-center gap-3">
        <div class="w-6 h-6 rounded-lg bg-gradient-to-br from-blue-400 to-purple-500 flex items-center justify-center">
          <svg class="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01"/>
          </svg>
        </div>
        <span class="text-neutral-900 font-medium text-sm">{{ t('appTitle') }}</span>
        <!-- 更新提示气泡 -->
        <span
          v-if="updateAvailable"
          class="update-badge"
          @click="handleUpdateClick"
          :title="updateDownloaded ? t('downloadDone') : t('updateDot')"
        >
          <span class="update-dot"></span>
          <span class="update-text">{{ updateDownloaded ? t('restartNow') : t('updateDot') }}</span>
        </span>
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
        <!-- Admin Warning -->
        <div
          v-if="!isAdmin"
          class="rounded-xl border border-orange-500/40 bg-orange-500/12 px-3 py-2 text-orange-700 text-xs"
        >
          {{ t('adminRequiredBanner') }} · {{ t('adminRequiredAction') }}
        </div>
        <div class="flex items-center justify-between">
          <h2 class="text-lg font-semibold text-neutral-900">{{ t('profiles') }}</h2>
          <button 
            class="glass-button text-sm text-blue-600 hover:text-blue-700"
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

        <!-- Search Box -->
        <div class="relative">
          <input
            v-model="searchQuery"
            type="text"
            :placeholder="t('searchPlaceholder')"
            class="w-full glass-input text-sm pl-9"
          />
          <svg class="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
          </svg>
        </div>

        <div class="flex-1 overflow-y-auto scrollbar-thin space-y-3 pr-2">
          <ProfileCard
            v-for="profile in filteredProfiles"
            :key="profile.name"
            :profile="profile"
            :active="selectedProfile?.name === profile.name"
            @click="selectProfile(profile)"
            @start="handleStart"
            @stop="handleStop"
            @contextmenu="onProfileContextMenu($event, profile)"
          />

          <div 
            v-if="filteredProfiles.length === 0 && profiles.length > 0"
            class="glass-card p-8 text-center text-neutral-500"
          >
            <p>{{ t('noMatches') }}</p>
          </div>

          <div 
            v-if="profiles.length === 0"
            class="glass-card p-8 text-center text-neutral-500"
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
      <div class="flex-1 overflow-y-auto scrollbar-thin">
        <ProfileEditor
          v-if="selectedProfile"
          :profile="selectedProfile"
          :hosts-text="hostsText"
          @save-text="handleSaveText"
          @start="handleStart"
          @stop="handleStop"
          @reload-hosts="handleReloadHosts"
        />

        <div 
          v-else 
          class="h-full glass-card flex items-center justify-center text-neutral-400"
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
      :used-ports="usedPorts"
      @close="showAddModal = false"
      @add="handleAdd"
      @add-remote="handleAddRemote"
    />

    <SettingsModal :show="showSettings" @close="showSettings = false" />

    <RenameProfileModal
      :show="showRename"
      :current-name="renameFrom"
      @close="showRename = false"
      @rename="handleRename"
    />

    <!-- Admin Modal -->
    <Teleport to="body">
      <div
        v-if="showAdminModal"
        class="fixed inset-0 z-[60] flex items-center justify-center p-4"
      >
        <div class="absolute inset-0 bg-black/25" @click="showAdminModal = false" />
        <div class="relative glass-card w-full max-w-md p-6">
          <h3 class="text-lg font-semibold text-neutral-900 mb-3">{{ t('adminRequiredTitle') }}</h3>
          <p class="text-sm text-orange-600 mb-2">{{ t('adminRequiredBanner') }}</p>
          <p class="text-sm text-neutral-700 mb-5">{{ t('adminRequiredAction') }}</p>
          <div class="flex justify-end gap-2">
            <button class="glass-button text-orange-600 border-orange-400/40" @click="handleRelaunchAsAdmin">
              {{ t('relaunchAsAdmin') }}
            </button>
            <button class="glass-button text-neutral-800" @click="showAdminModal = false">
              {{ t('gotIt') }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Context Menu -->
    <Teleport to="body">
      <template v-if="contextMenu.show">
        <div class="fixed inset-0 z-[50]" @click="closeContextMenu" @contextmenu.prevent="closeContextMenu" />
        <div
          class="fixed z-[51] rounded-xl border border-neutral-200 bg-white/80 backdrop-blur-md shadow-2xl py-1.5 min-w-[160px]"
          :style="{ left: contextMenu.x + 'px', top: contextMenu.y + 'px' }"
        >
          <button
            class="w-full text-left px-4 py-2 text-sm text-neutral-800 hover:bg-white/10 transition-colors flex items-center gap-2.5"
            @click="handleExportFromMenu"
          >
            <svg class="w-4 h-4 text-neutral-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
            </svg>
            {{ t('export') }}
          </button>
          <button
            class="w-full text-left px-4 py-2 text-sm text-neutral-800 hover:bg-white/10 transition-colors flex items-center gap-2.5"
            @click="handleRenameFromMenu"
          >
            <svg class="w-4 h-4 text-neutral-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/>
            </svg>
            {{ t('rename') }}
          </button>
          <div class="border-t border-neutral-200 my-1" />
          <button
            class="w-full text-left px-4 py-2 text-sm transition-colors flex items-center gap-2.5"
            :class="profiles.find(p => p.name === contextMenu.profileName)?.system_hosts_active
              ? 'text-neutral-400 cursor-not-allowed'
              : 'text-red-400 hover:bg-red-500/10'"
            :disabled="!!profiles.find(p => p.name === contextMenu.profileName)?.system_hosts_active"
            @click="handleDeleteFromMenu"
          >
            <svg class="w-4 h-4" :class="profiles.find(p => p.name === contextMenu.profileName)?.system_hosts_active ? 'text-neutral-300' : 'text-red-400/50'" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
            </svg>
            {{ t('delete') }}
          </button>
        </div>
      </template>
    </Teleport>
  </div>
</template>
