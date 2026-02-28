<script setup lang="ts">
import { ref, watch } from 'vue'
import { t } from '../i18n'

const props = defineProps<{
  show: boolean
  currentName: string
}>()

const emit = defineEmits<{
  close: []
  rename: [newName: string]
}>()

const newName = ref('')
const error = ref('')

watch(() => props.show, (show) => {
  if (show) {
    newName.value = props.currentName
    error.value = ''
  }
})

function submit() {
  const v = newName.value.trim()
  if (!v) {
    error.value = t('profileNameRequired')
    return
  }
  emit('rename', v)
}
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="show" class="fixed inset-0 z-50 flex items-center justify-center p-4">
        <div class="absolute inset-0 bg-black/60" @click="emit('close')" />

        <div class="relative glass-card w-full max-w-md p-6">
          <div class="flex items-center justify-between mb-6">
            <h3 class="text-xl font-semibold text-white/90">{{ t('renameProfile') }}</h3>
            <button
              class="p-1.5 rounded-lg text-white/60 hover:text-white hover:bg-white/10 transition-all"
              @click="emit('close')"
            >
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          <form @submit.prevent="submit" class="space-y-4">
            <div>
              <label class="block text-sm text-white/70 mb-2">{{ t('newProfileName') }}</label>
              <input v-model="newName" class="glass-input" :placeholder="props.currentName" autofocus />
            </div>
            <div v-if="error" class="text-red-400 text-sm">{{ error }}</div>

            <div class="flex justify-end gap-3 pt-4">
              <button type="button" class="glass-button text-white/80" @click="emit('close')">
                {{ t('cancel') }}
              </button>
              <button type="submit" class="glass-button bg-blue-500/30 text-blue-200 hover:bg-blue-500/40 border-blue-400/30">
                {{ t('save') }}
              </button>
            </div>
          </form>
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

