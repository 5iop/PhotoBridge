<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import * as api from '../../api'
import { getUploadUrl } from '../../api'

const route = useRoute()

const info = ref(null)
const photos = ref([])
const loading = ref(true)
const error = ref('')

const lightboxPhoto = ref(null)
const lightboxIndex = ref(0)

const showDownloadModal = ref(false)
const downloadType = ref('normal')
const downloading = ref(false)

const token = computed(() => route.params.token)

onMounted(async () => {
  await fetchData()
})

async function fetchData() {
  loading.value = true
  error.value = ''
  try {
    const [infoRes, photosRes] = await Promise.all([
      api.getShareInfo(token.value),
      api.getSharePhotos(token.value)
    ])
    info.value = infoRes.data
    photos.value = photosRes.data || []
  } catch (err) {
    error.value = err.response?.data?.error || 'Failed to load gallery'
  } finally {
    loading.value = false
  }
}

function getPhotoUrl(photo) {
  return `${getUploadUrl()}${photo.normal_url}`
}

function openLightbox(index) {
  lightboxIndex.value = index
  lightboxPhoto.value = photos.value[index]
}

function closeLightbox() {
  lightboxPhoto.value = null
}

function prevPhoto() {
  lightboxIndex.value = (lightboxIndex.value - 1 + photos.value.length) % photos.value.length
  lightboxPhoto.value = photos.value[lightboxIndex.value]
}

function nextPhoto() {
  lightboxIndex.value = (lightboxIndex.value + 1) % photos.value.length
  lightboxPhoto.value = photos.value[lightboxIndex.value]
}

function handleKeydown(e) {
  if (!lightboxPhoto.value) return
  if (e.key === 'ArrowLeft') prevPhoto()
  if (e.key === 'ArrowRight') nextPhoto()
  if (e.key === 'Escape') closeLightbox()
}

function download() {
  downloading.value = true
  const url = `${getUploadUrl()}/api/share/${token.value}/download?type=${downloadType.value}`

  const a = document.createElement('a')
  a.href = url
  a.download = ''
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)

  setTimeout(() => {
    downloading.value = false
    showDownloadModal.value = false
  }, 1000)
}

// Handle keyboard events
if (typeof window !== 'undefined') {
  window.addEventListener('keydown', handleKeydown)
}
</script>

<template>
  <div class="min-h-screen bg-dark-600">
    <!-- Loading -->
    <div v-if="loading" class="min-h-screen flex items-center justify-center">
      <svg class="w-12 h-12 text-primary-500 spinner" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
      </svg>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="min-h-screen flex items-center justify-center p-4">
      <div class="text-center">
        <div class="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-red-500/20 mb-4">
          <svg class="w-8 h-8 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
        </div>
        <h2 class="text-xl font-bold text-white mb-2">Oops!</h2>
        <p class="text-gray-400">{{ error }}</p>
      </div>
    </div>

    <!-- Gallery -->
    <div v-else>
      <!-- Header -->
      <header class="sticky top-0 z-40 bg-dark-500/80 backdrop-blur-lg border-b border-dark-200">
        <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
          <div class="flex items-center justify-between">
            <div>
              <h1 class="text-xl sm:text-2xl font-bold text-white">{{ info.project_name }}</h1>
              <p class="text-sm text-gray-400 mt-1">{{ info.photo_count }} photos</p>
            </div>
            <button @click="showDownloadModal = true" class="btn btn-primary">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
              </svg>
              <span class="hidden sm:inline">Download All</span>
            </button>
          </div>
        </div>
      </header>

      <!-- Photo grid -->
      <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-2 sm:gap-4">
          <div
            v-for="(photo, index) in photos"
            :key="photo.id"
            class="aspect-square rounded-lg sm:rounded-xl overflow-hidden bg-dark-300 cursor-pointer group"
            @click="openLightbox(index)"
          >
            <img
              v-if="photo.normal_url"
              :src="getPhotoUrl(photo)"
              class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
              loading="lazy"
            />
            <div v-if="photo.has_raw && info.allow_raw" class="absolute top-2 right-2 px-2 py-0.5 rounded-full bg-primary-500/80 text-white text-xs font-medium">
              RAW
            </div>
          </div>
        </div>
      </main>
    </div>

    <!-- Lightbox -->
    <div
      v-if="lightboxPhoto"
      class="fixed inset-0 z-50 bg-black/95 flex items-center justify-center"
      @click="closeLightbox"
    >
      <!-- Close button -->
      <button
        class="absolute top-4 right-4 p-2 rounded-full bg-white/10 hover:bg-white/20 text-white z-10"
        @click.stop="closeLightbox"
      >
        <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>

      <!-- Prev button -->
      <button
        class="absolute left-4 top-1/2 -translate-y-1/2 p-3 rounded-full bg-white/10 hover:bg-white/20 text-white"
        @click.stop="prevPhoto"
      >
        <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
        </svg>
      </button>

      <!-- Next button -->
      <button
        class="absolute right-4 top-1/2 -translate-y-1/2 p-3 rounded-full bg-white/10 hover:bg-white/20 text-white"
        @click.stop="nextPhoto"
      >
        <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
        </svg>
      </button>

      <!-- Image -->
      <img
        v-if="lightboxPhoto.normal_url"
        :src="getPhotoUrl(lightboxPhoto)"
        class="max-w-[90vw] max-h-[90vh] object-contain"
        @click.stop
      />

      <!-- Counter -->
      <div class="absolute bottom-4 left-1/2 -translate-x-1/2 px-4 py-2 rounded-full bg-white/10 text-white text-sm">
        {{ lightboxIndex + 1 }} / {{ photos.length }}
      </div>
    </div>

    <!-- Download Modal -->
    <div v-if="showDownloadModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50">
      <div class="card p-6 w-full max-w-sm" @click.stop>
        <h3 class="text-lg font-semibold text-white mb-4">Download Photos</h3>

        <div class="space-y-3">
          <label
            class="flex items-center gap-3 p-3 rounded-xl cursor-pointer transition-colors"
            :class="downloadType === 'normal' ? 'bg-primary-500/20 border border-primary-500' : 'bg-dark-300 border border-dark-100'"
          >
            <input type="radio" v-model="downloadType" value="normal" class="hidden" />
            <div class="w-5 h-5 rounded-full border-2 flex items-center justify-center"
              :class="downloadType === 'normal' ? 'border-primary-500' : 'border-gray-500'">
              <div v-if="downloadType === 'normal'" class="w-2.5 h-2.5 rounded-full bg-primary-500"></div>
            </div>
            <div>
              <p class="font-medium text-white">Standard Photos</p>
              <p class="text-sm text-gray-400">JPG format, optimized for web</p>
            </div>
          </label>

          <label
            v-if="info?.allow_raw"
            class="flex items-center gap-3 p-3 rounded-xl cursor-pointer transition-colors"
            :class="downloadType === 'raw' ? 'bg-primary-500/20 border border-primary-500' : 'bg-dark-300 border border-dark-100'"
          >
            <input type="radio" v-model="downloadType" value="raw" class="hidden" />
            <div class="w-5 h-5 rounded-full border-2 flex items-center justify-center"
              :class="downloadType === 'raw' ? 'border-primary-500' : 'border-gray-500'">
              <div v-if="downloadType === 'raw'" class="w-2.5 h-2.5 rounded-full bg-primary-500"></div>
            </div>
            <div>
              <p class="font-medium text-white">RAW Files Only</p>
              <p class="text-sm text-gray-400">Original quality RAW format</p>
            </div>
          </label>

          <label
            v-if="info?.allow_raw"
            class="flex items-center gap-3 p-3 rounded-xl cursor-pointer transition-colors"
            :class="downloadType === 'all' ? 'bg-primary-500/20 border border-primary-500' : 'bg-dark-300 border border-dark-100'"
          >
            <input type="radio" v-model="downloadType" value="all" class="hidden" />
            <div class="w-5 h-5 rounded-full border-2 flex items-center justify-center"
              :class="downloadType === 'all' ? 'border-primary-500' : 'border-gray-500'">
              <div v-if="downloadType === 'all'" class="w-2.5 h-2.5 rounded-full bg-primary-500"></div>
            </div>
            <div>
              <p class="font-medium text-white">All Files</p>
              <p class="text-sm text-gray-400">Both standard and RAW files</p>
            </div>
          </label>
        </div>

        <div class="flex gap-3 mt-6">
          <button @click="showDownloadModal = false" class="btn btn-secondary flex-1">
            Cancel
          </button>
          <button @click="download" class="btn btn-primary flex-1" :disabled="downloading">
            <svg v-if="downloading" class="w-5 h-5 spinner" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            <span v-else>Download</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
