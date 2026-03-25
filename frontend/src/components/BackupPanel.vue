<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { CreateHostsBackup, ListHostsBackups, RestoreHostsBackup, ClearHostsEntries, ResetHostsTemplate } from '../../wailsjs/go/main/App'

interface BackupInfo {
  file_name: string
  path: string
  size: number
  modified: string
}

const props = defineProps<{
  profileName: string
}>()
const emit = defineEmits<{
  changed: []
}>()

const backups = ref<BackupInfo[]>([])
const loading = ref(false)

async function refreshBackups() {
  loading.value = true
  try {
    backups.value = await ListHostsBackups(props.profileName)
  } catch (e) {
    console.error('load backups failed', e)
  } finally {
    loading.value = false
  }
}

async function createBackup() {
  try {
    await CreateHostsBackup(props.profileName)
    await refreshBackups()
  } catch (e) {
    console.error('create backup failed', e)
    alert(String(e))
  }
}

async function restoreBackup(fileName: string) {
  if (!confirm(`确认恢复备份：${fileName} ?`)) return
  try {
    await RestoreHostsBackup(props.profileName, fileName)
    await refreshBackups()
    emit('changed')
  } catch (e) {
    console.error('restore backup failed', e)
    alert(String(e))
  }
}

async function clearHosts() {
  if (!confirm('确认清空当前 hosts 内容?')) return
  try {
    await ClearHostsEntries(props.profileName)
    emit('changed')
  } catch (e) {
    console.error('clear hosts failed', e)
    alert(String(e))
  }
}

async function resetTemplate() {
  if (!confirm('确认恢复默认模板?')) return
  try {
    await ResetHostsTemplate(props.profileName)
    emit('changed')
  } catch (e) {
    console.error('reset template failed', e)
    alert(String(e))
  }
}

watch(() => props.profileName, refreshBackups, { immediate: true })
onMounted(refreshBackups)
</script>

<template>
  <div class="rounded-xl border border-slate-700/60 bg-slate-900 p-3 mb-4">
    <div class="flex items-center justify-between mb-3">
      <div class="text-sm text-white/80">备份与还原</div>
      <button class="glass-button text-xs" @click="refreshBackups" :disabled="loading">刷新</button>
    </div>
    <div class="flex gap-2 mb-3">
      <button class="glass-button text-xs text-blue-200" @click="createBackup">创建快照</button>
      <button class="glass-button text-xs text-amber-200" @click="resetTemplate">恢复模板</button>
      <button class="glass-button text-xs text-red-200" @click="clearHosts">清空</button>
    </div>
    <div class="max-h-28 overflow-y-auto text-xs text-white/70 space-y-1">
      <div v-if="backups.length === 0" class="text-white/40">暂无备份</div>
      <div v-for="b in backups" :key="b.file_name" class="flex items-center justify-between rounded px-2 py-1 bg-slate-800/70">
        <div class="truncate mr-2">
          <div class="font-mono">{{ b.file_name }}</div>
          <div class="text-white/40">{{ b.modified }} · {{ b.size }}B</div>
        </div>
        <button class="glass-button text-[11px]" @click="restoreBackup(b.file_name)">还原</button>
      </div>
    </div>
  </div>
</template>
