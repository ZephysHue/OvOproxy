<script setup lang="ts">
import { computed } from 'vue'
import { lang, setLang, t, type Lang } from '../i18n'
import { themeMode, setThemeMode, type ThemeMode } from '../theme'

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

const currentTheme = computed({
  get: () => themeMode.value,
  set: (v: ThemeMode) => setThemeMode(v),
})
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="show" class="fixed inset-0 z-50 flex items-center justify-center p-4">
        <div class="absolute inset-0 bg-black/60" @click="emit('close')" />

        <div class="relative glass-card w-full max-w-md p-6">
          <div class="flex items-center justify-between mb-6">
            <h3 class="text-xl font-semibold text-white/90">{{ t('settings') }}</h3>
            <button
              class="p-1.5 rounded-lg text-white/60 hover:text-white hover:bg-white/10 transition-all"
              @click="emit('close')"
            >
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          <div class="space-y-4">
            <div>
              <label class="block text-sm text-white/70 mb-2">{{ t('language') }}</label>
              <select v-model="current" class="glass-input">
                <option value="zh">{{ t('chinese') }}</option>
                <option value="en">{{ t('english') }}</option>
              </select>
            </div>
            <div>
              <label class="block text-sm text-white/70 mb-2">{{ t('theme') }}</label>
              <select v-model="currentTheme" class="glass-input">
                <option value="system">{{ t('themeSystem') }}</option>
                <option value="dark">{{ t('themeDark') }}</option>
                <option value="light">{{ t('themeLight') }}</option>
              </select>
            </div>
          </div>

          <div class="flex justify-end gap-3 pt-6">
            <button class="glass-button text-white/80" @click="emit('close')">
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

