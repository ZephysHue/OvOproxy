<script setup lang="ts">
import { ref, watch } from 'vue'
import { t } from '../i18n'
import {
  GetProfileSubscriptions,
  AddProfileSubscription,
  RemoveProfileSubscription,
  SetProfileSubscriptionEnabled,
  RefreshProfileSubscriptions,
  RefreshSingleProfileSubscription,
  PreviewSubscriptionConflicts,
} from '../../wailsjs/go/main/App'

interface Subscription {
  id: string
  name: string
  url: string
  enabled: boolean
  last_updated?: string
  last_status?: string
}

interface ConflictPreview {
  sub_id: string
  sub_name: string
  domains: string[]
  total: number
  truncate: boolean
}

const props = defineProps<{ profileName: string }>()
const emit = defineEmits<{ changed: [] }>()

const subs = ref<Subscription[]>([])
const addingName = ref('')
const addingURL = ref('')
const busy = ref(false)
const subBusyId = ref('')
const previews = ref<Record<string, ConflictPreview | undefined>>({})

async function loadSubs() {
  try {
    subs.value = await GetProfileSubscriptions(props.profileName)
  } catch (e) {
    console.error('load subscriptions failed', e)
  }
}

async function addSub() {
  if (!addingURL.value.trim()) return
  busy.value = true
  try {
    await AddProfileSubscription(props.profileName, addingName.value.trim(), addingURL.value.trim())
    addingName.value = ''
    addingURL.value = ''
    await loadSubs()
  } catch (e) {
    alert(String(e))
  } finally {
    busy.value = false
  }
}

async function removeSub(id: string) {
  busy.value = true
  try {
    await RemoveProfileSubscription(props.profileName, id)
    await loadSubs()
  } catch (e) {
    alert(String(e))
  } finally {
    busy.value = false
  }
}

async function toggleSub(id: string, enabled: boolean) {
  busy.value = true
  try {
    await SetProfileSubscriptionEnabled(props.profileName, id, enabled)
    await loadSubs()
  } catch (e) {
    alert(String(e))
  } finally {
    busy.value = false
  }
}

async function manualRefresh() {
  busy.value = true
  try {
    await RefreshProfileSubscriptions(props.profileName)
    await loadSubs()
    emit('changed')
  } catch (e) {
    alert(String(e))
  } finally {
    busy.value = false
  }
}

async function manualRefreshOne(subId: string) {
  subBusyId.value = subId
  try {
    await RefreshSingleProfileSubscription(props.profileName, subId)
    await loadSubs()
    emit('changed')
  } catch (e) {
    alert(String(e))
  } finally {
    subBusyId.value = ''
  }
}

async function previewConflicts(subId: string) {
  subBusyId.value = subId
  try {
    const preview = await PreviewSubscriptionConflicts(props.profileName, subId)
    previews.value[subId] = preview
  } catch (e) {
    alert(String(e))
  } finally {
    subBusyId.value = ''
  }
}

watch(() => props.profileName, loadSubs, { immediate: true })
</script>

<template>
  <div class="rounded-xl border border-slate-700/60 bg-slate-900 p-3 mb-4">
    <div class="flex items-center justify-between mb-3">
      <div class="text-sm text-white/80">{{ t('subscriptions') }}</div>
      <button class="glass-button text-xs text-cyan-200" :disabled="busy" @click="manualRefresh">
        {{ t('manualRefresh') }}
      </button>
    </div>
    <div class="grid grid-cols-12 gap-2 mb-3">
      <input
        v-model="addingName"
        class="glass-input text-xs col-span-3 py-2"
        :placeholder="t('subscriptionName')"
      />
      <input
        v-model="addingURL"
        class="glass-input text-xs col-span-7 py-2"
        :placeholder="t('subscriptionUrl')"
      />
      <button class="glass-button text-xs col-span-2" :disabled="busy" @click="addSub">
        {{ t('addSubscription') }}
      </button>
    </div>
    <div class="max-h-36 overflow-y-auto scrollbar-thin space-y-2">
      <div v-if="subs.length === 0" class="text-xs text-white/40">{{ t('noProfiles') }}</div>
      <div
        v-for="sub in subs"
        :key="sub.id"
        class="rounded-lg border border-slate-700/50 bg-slate-800/60 px-2 py-2 text-xs"
      >
        <div class="flex items-center justify-between gap-2">
          <div class="min-w-0">
            <div class="text-white/90 truncate">{{ sub.name || sub.url }}</div>
            <div class="text-white/50 truncate font-mono">{{ sub.url }}</div>
          </div>
          <div class="flex items-center gap-2">
            <label class="flex items-center gap-1 text-white/70">
              <input
                type="checkbox"
                :checked="sub.enabled"
                @change="toggleSub(sub.id, ($event.target as HTMLInputElement).checked)"
              />
              {{ sub.enabled ? t('enabled') : t('disabled') }}
            </label>
            <button
              class="glass-button text-[11px] text-cyan-200 px-2 py-1"
              :disabled="busy || !!subBusyId"
              @click="manualRefreshOne(sub.id)"
            >
              {{ t('refreshThisSubscription') }}
            </button>
            <button
              class="glass-button text-[11px] text-amber-200 px-2 py-1"
              :disabled="busy || !!subBusyId"
              @click="previewConflicts(sub.id)"
            >
              {{ t('conflictPreview') }}
            </button>
            <button class="glass-button text-[11px] text-red-200 px-2 py-1" @click="removeSub(sub.id)">
              {{ t('remove') }}
            </button>
          </div>
        </div>
        <div class="mt-1 text-white/40">
          {{ t('lastStatus') }}: {{ sub.last_status || '-' }} · {{ t('lastUpdated') }}: {{ sub.last_updated || '-' }}
        </div>
        <div v-if="previews[sub.id]" class="mt-1 text-[11px] text-amber-200/90">
          {{ t('conflictCount', { count: previews[sub.id]!.total }) }}
          <span v-if="previews[sub.id]!.domains.length > 0">
            · {{ previews[sub.id]!.domains.join(', ') }}
          </span>
          <span v-if="previews[sub.id]!.truncate"> ...</span>
        </div>
      </div>
    </div>
  </div>
</template>
