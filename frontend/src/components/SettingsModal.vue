<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { lang, setLang, t, type Lang } from '../i18n'
import { changelog } from '../changelog'
import { GetVersion, CheckUpdate, DownloadUpdate, ApplyUpdate } from '../../wailsjs/go/main/App'

const props = defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  close: []
}>()

const current = computed({
  get: () => lang.value,
  set: (v: Lang) => setLang(v),
})

const repoUrl = 'https://github.com/ZephysHue/OvOproxy'

const appVersion = ref('')
const updateStatus = ref('') // '' | 'checking' | 'latest' | 'found' | 'downloading' | 'downloaded'
const updateMsg = ref('')

async function loadVersion() {
  try { appVersion.value = await GetVersion() } catch { appVersion.value = 'unknown' }
}

async function handleCheckUpdate() {
  updateStatus.value = 'checking'
  updateMsg.value = ''
  try {
    const result = await CheckUpdate()
    if (result.has_update && result.download_url) {
      updateStatus.value = 'found'
      updateMsg.value = t('newVersionFound') + ': ' + result.latest
      // 自动下载
      updateStatus.value = 'downloading'
      await DownloadUpdate(result.download_url)
      updateStatus.value = 'downloaded'
      updateMsg.value = t('downloadDone')
    } else {
      updateStatus.value = 'latest'
      updateMsg.value = t('alreadyLatest')
    }
  } catch (e: any) {
    updateStatus.value = ''
    updateMsg.value = e?.message || String(e)
  }
}

async function handleRestart() {
  await ApplyUpdate()
}

onMounted(() => { loadVersion() })
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="show" class="fixed inset-0 z-50 flex items-center justify-center p-4">
        <div class="absolute inset-0 bg-black/60" @click="emit('close')" />

        <div class="relative glass-card w-full max-w-md p-6">
          <div class="flex items-center justify-between mb-6">
            <h3 class="text-xl font-semibold text-neutral-900">{{ t('settings') }}</h3>
            <button
              class="p-1.5 rounded-lg text-neutral-500 hover:text-white hover:bg-white/10 transition-all"
              @click="emit('close')"
            >
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          <div class="space-y-4">
            <div>
              <label class="block text-sm text-neutral-700 mb-2">{{ t('language') }}</label>
              <select v-model="current" class="glass-input">
                <option value="zh">{{ t('chinese') }}</option>
                <option value="en">{{ t('english') }}</option>
              </select>
            </div>

            <!-- 分隔线 -->
            <div class="border-t border-neutral-200/60 my-2"></div>

            <!-- 版本与更新 -->
            <div>
              <div class="flex items-center justify-between mb-2">
                <label class="text-sm text-neutral-700">{{ t('versionLabel') }}</label>
                <span class="text-xs text-neutral-500">{{ appVersion }}</span>
              </div>
              <div class="flex items-center gap-2">
                <button
                  class="glass-button text-sm text-blue-600"
                  :disabled="updateStatus === 'checking' || updateStatus === 'downloading'"
                  @click="handleCheckUpdate"
                >
                  {{ updateStatus === 'checking' || updateStatus === 'downloading' ? t('checkingUpdate') : t('checkUpdate') }}
                </button>
                <button
                  v-if="updateStatus === 'downloaded'"
                  class="glass-button text-sm text-red-500 border-red-300/40 hover:text-red-600"
                  @click="handleRestart"
                >
                  {{ t('restartNow') }}
                </button>
              </div>
              <p v-if="updateMsg" class="text-xs mt-1.5" :class="updateStatus === 'downloaded' ? 'text-green-600' : updateStatus === 'latest' ? 'text-neutral-500' : 'text-blue-600'">
                {{ updateMsg }}
              </p>
            </div>

            <!-- 分隔线 -->
            <div class="border-t border-neutral-200/60 my-2"></div>

            <!-- 项目仓库 -->
            <div>
              <label class="block text-sm text-neutral-700 mb-1.5">{{ t('repoUrl') }}</label>
              <a
                :href="repoUrl"
                target="_blank"
                rel="noopener"
                class="text-sm text-blue-600 hover:text-blue-700 transition-colors break-all"
              >
                {{ repoUrl }}
              </a>
            </div>

            <!-- 反馈联系方式 -->
            <div>
              <label class="block text-sm text-neutral-700 mb-1.5">{{ t('feedbackTitle') }}</label>
              <p class="text-xs text-neutral-500 mb-1.5">{{ t('feedbackDesc') }}</p>
              <div class="text-sm text-neutral-800 space-y-0.5">
                <p>{{ t('contactWeCom') }}</p>
                <p>{{ t('contactEmail') }}</p>
              </div>
            </div>

            <!-- 分隔线 -->
            <div class="border-t border-neutral-200/60 my-2"></div>

            <!-- 更新日志 -->
            <div>
              <label class="block text-sm text-neutral-700 mb-2">{{ t('changelogTitle') }}</label>
              <div class="changelog-scroll max-h-48 overflow-y-auto scrollbar-thin space-y-3 pr-1">
                <div v-for="version in changelog" :key="version.date">
                  <div class="text-xs font-semibold text-neutral-900 mb-1">{{ version.date }}</div>
                  <div class="space-y-1">
                    <div v-for="(entry, idx) in version.entries" :key="idx" class="text-xs text-neutral-600 leading-relaxed">
                      <span class="font-medium text-neutral-800">{{ entry.title }}</span>
                      <span class="text-neutral-400"> — </span>
                      <span>{{ entry.desc }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div class="flex justify-end gap-3 pt-6">
            <button class="glass-button text-neutral-800" @click="emit('close')">
              {{ t('cancel') }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.modal-enter-active,
.modal-leave-active {
  transition: all 0.2s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}
</style>
