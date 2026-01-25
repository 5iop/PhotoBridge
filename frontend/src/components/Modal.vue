<script setup>
defineProps({
  show: {
    type: Boolean,
    required: true
  },
  title: {
    type: String,
    default: ''
  },
  maxWidth: {
    type: String,
    default: 'max-w-md'
  }
})

const emit = defineEmits(['close'])

function handleBackdropClick() {
  emit('close')
}
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div
        v-if="show"
        class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50"
        @click="handleBackdropClick"
      >
        <div
          class="card p-6 w-full"
          :class="maxWidth"
          @click.stop
        >
          <div v-if="title" class="flex items-center justify-between mb-4">
            <h3 class="text-lg font-semibold text-white">{{ title }}</h3>
            <button
              @click="emit('close')"
              class="text-gray-400 hover:text-white transition-colors"
            >
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
          <slot></slot>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.2s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-active .card,
.modal-leave-active .card {
  transition: transform 0.2s ease;
}

.modal-enter-from .card,
.modal-leave-to .card {
  transform: scale(0.95);
}
</style>
