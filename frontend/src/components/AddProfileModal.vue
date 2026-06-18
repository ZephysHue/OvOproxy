<script setup lang="ts">
import { ref, watch } from 'vue'
import { t } from '../i18n'
import { SUBSCRIPTION_INTERVAL_OPTIONS, DEFAULT_SUB_INTERVAL, DEFAULT_PORT } from '../constants'

const props = defineProps<{
  show: boolean
  usedPorts?: number[]
}>()

const emit = defineEmits<{
  close: []
  add: [name: string, ip: string, port: number]
  addRemote: [name: string, ip: string, port: number, url: string, interval: number]
}>()

const name = ref('')
const ip = ref('127.0.0.1')
const port = ref(DEFAULT_PORT)
const error = ref('')
const profileType = ref<'local' | 'remote'>('local')
const subUrl = ref('')
const subInterval = ref(DEFAULT_SUB_INTERVAL)

function nextAvailablePort(usedPorts: number[]) {
  const used = new Set(usedPorts)
  let candidate = DEFAULT_PORT
  while (used.has(candidate) && candidate <= 65535) candidate++
  return candidate > 65535 ? 8080 : candidate
}

watch(() => props.show, (show) => {
  if (show) {
    name.value = ''
    ip.value = '127.0.0.1'
    port.value = nextAvailablePort(props.usedPorts || [])
    error.value = ''
    profileType.value = 'local'
    subUrl.value = ''
    subInterval.value = DEFAULT_SUB_INTERVAL
  }
})

function handleSubmit() {
  if (!name.value.trim()) { error.value = t('profileNameRequired'); return }
  if (!ip.value.trim()) { error.value = t('listenIPRequired'); return }
  if (port.value < 1 || port.value > 65535) { error.value = t('portRangeError'); return }
  if (profileType.value === 'remote' && !subUrl.value.trim()) { error.value = '订阅 URL 不能为空'; return }

  if (profileType.value === 'remote') {
    emit('addRemote', name.value.trim(), ip.value.trim(), port.value, subUrl.value.trim(), subInterval.value)
  } else {
    emit('add', name.value.trim(), ip.value.trim(), port.value)
  }
}
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="show" class="fixed inset-0 z-50 flex items-center justify-center p-4">
        <div class="absolute inset-0 bg-black/60" @click="emit('close')" />
        <div class="relative glass-card w-full max-w-md p-6 animate-in fade-in zoom-in-95 duration-200">
          <div class="flex items-center justify-between mb-6">
            <h3 class="text-xl font-semibold text-neutral-900">{{ t('newProfile') }}</h3>
            <button class="p-1.5 rounded-lg text-neutral-500 hover:text-white hover:bg-white/10 transition-all" @click="emit('close')">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
              </svg>
            </button>
          </div>

          <form @submit.prevent="handleSubmit" class="space-y-4">
            <!-- Type selector -->
            <div>
              <label class="block text-sm text-neutral-500 mb-2">类型</label>
              <div class="flex gap-2">
                <button type="button" class="flex-1 py-2 rounded-lg text-sm transition-all"
                  :class="profileType === 'local' ? 'bg-blue-500/30 text-blue-200 border border-blue-400/30' : 'bg-white/5 text-neutral-400 border border-neutral-200'"
                  @click="profileType = 'local'">本地 Hosts</button>
                <button type="button" class="flex-1 py-2 rounded-lg text-sm transition-all"
                  :class="profileType === 'remote' ? 'bg-purple-500/30 text-purple-200 border border-purple-400/30' : 'bg-white/5 text-neutral-400 border border-neutral-200'"
                  @click="profileType = 'remote'">订阅远程</button>
              </div>
            </div>

            <div>
              <label class="block text-sm text-neutral-500 mb-2">{{ t('profileName') }}</label>
              <input v-model="name" type="text" placeholder="e.g. dev-server" class="glass-input" autofocus />
            </div>

            <div class="flex gap-4">
              <div class="flex-1">
                <label class="block text-sm text-neutral-500 mb-2">{{ t('listenIP') }}</label>
                <input v-model="ip" type="text" placeholder="127.0.0.1" class="glass-input" />
              </div>
              <div class="w-32">
                <label class="block text-sm text-neutral-500 mb-2">{{ t('port') }}</label>
                <input v-model.number="port" type="number" min="1" max="65535" class="glass-input" />
              </div>
            </div>

            <!-- Remote fields -->
            <template v-if="profileType === 'remote'">
              <div>
                <label class="block text-sm text-neutral-500 mb-2">{{ t('subscriptionUrl') }}</label>
                <input v-model="subUrl" type="url" placeholder="https://example.com/hosts.txt" class="glass-input" />
              </div>
              <div>
                <label class="block text-sm text-neutral-500 mb-2">{{ t('refreshIntervalSeconds') }}</label>
                <select v-model.number="subInterval" class="glass-input py-2">
                  <option v-for="opt in SUBSCRIPTION_INTERVAL_OPTIONS" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
                </select>
              </div>
            </template>

            <div v-if="error" class="text-red-400 text-sm">{{ error }}</div>

            <div class="flex justify-end gap-3 pt-4">
              <button type="button" class="glass-button text-neutral-500" @click="emit('close')">{{ t('cancel') }}</button>
              <button type="submit" class="glass-button bg-blue-500/30 text-blue-200 hover:bg-blue-500/40 border-blue-400/30">
                {{ t('createProfile') }}
              </button>
            </div>
          </form>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.modal-enter-active, .modal-leave-active { transition: all 0.2s ease; }
.modal-enter-from, .modal-leave-to { opacity: 0; }
.modal-enter-from .glass-card, .modal-leave-to .glass-card { transform: scale(0.95); }
</style>
