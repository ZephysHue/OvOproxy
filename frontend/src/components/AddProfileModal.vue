<script setup lang="ts">
import { ref, watch } from 'vue'

const props = defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  close: []
  add: [name: string, ip: string, port: number]
}>()

const name = ref('')
const ip = ref('127.0.0.1')
const port = ref(8080)
const error = ref('')

watch(() => props.show, (show) => {
  if (show) {
    name.value = ''
    ip.value = '127.0.0.1'
    port.value = 8080
    error.value = ''
  }
})

function handleSubmit() {
  if (!name.value.trim()) {
    error.value = 'Profile name is required'
    return
  }
  if (!ip.value.trim()) {
    error.value = 'Listen IP is required'
    return
  }
  if (port.value < 1 || port.value > 65535) {
    error.value = 'Port must be between 1 and 65535'
    return
  }
  
  emit('add', name.value.trim(), ip.value.trim(), port.value)
}
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div 
        v-if="show" 
        class="fixed inset-0 z-50 flex items-center justify-center p-4"
      >
        <!-- Backdrop -->
        <div 
          class="absolute inset-0 bg-black/50 backdrop-blur-sm"
          @click="emit('close')"
        />

        <!-- Modal -->
        <div class="relative glass-card w-full max-w-md p-6 animate-in fade-in zoom-in-95 duration-200">
          <div class="flex items-center justify-between mb-6">
            <h3 class="text-xl font-semibold text-white/90">New Profile</h3>
            <button
              class="p-1.5 rounded-lg text-white/60 hover:text-white hover:bg-white/10 transition-all"
              @click="emit('close')"
            >
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
              </svg>
            </button>
          </div>

          <form @submit.prevent="handleSubmit" class="space-y-4">
            <div>
              <label class="block text-sm text-white/60 mb-2">Profile Name</label>
              <input
                v-model="name"
                type="text"
                placeholder="e.g. dev-server"
                class="glass-input"
                autofocus
              />
            </div>

            <div class="flex gap-4">
              <div class="flex-1">
                <label class="block text-sm text-white/60 mb-2">Listen IP</label>
                <input
                  v-model="ip"
                  type="text"
                  placeholder="127.0.0.1"
                  class="glass-input"
                />
              </div>
              <div class="w-32">
                <label class="block text-sm text-white/60 mb-2">Port</label>
                <input
                  v-model.number="port"
                  type="number"
                  min="1"
                  max="65535"
                  class="glass-input"
                />
              </div>
            </div>

            <div v-if="error" class="text-red-400 text-sm">
              {{ error }}
            </div>

            <div class="flex justify-end gap-3 pt-4">
              <button
                type="button"
                class="glass-button text-white/60"
                @click="emit('close')"
              >
                Cancel
              </button>
              <button
                type="submit"
                class="glass-button bg-blue-500/30 text-blue-200 hover:bg-blue-500/40 border-blue-400/30"
              >
                Create Profile
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

.modal-enter-from .glass-card,
.modal-leave-to .glass-card {
  transform: scale(0.95);
}
</style>
