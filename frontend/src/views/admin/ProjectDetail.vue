<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import * as api from '../../api'
import { getUploadUrl } from '../../api'

const route = useRoute()
const router = useRouter()

const project = ref(null)
const photos = ref([])
const loading = ref(true)
const uploading = ref(false)
const uploadProgress = ref(0)
const dragOver = ref(false)
const selectedPhotos = ref(new Set())

const projectId = computed(() => route.params.id)

onMounted(async () => {
  await fetchData()
})

async function fetchData() {
  loading.value = true
  try {
    const [projectRes, photosRes] = await Promise.all([
      api.getProject(projectId.value),
      api.getProjectPhotos(projectId.value)
    ])
    project.value = projectRes.data
    photos.value = photosRes.data || []
  } finally {
    loading.value = false
  }
}

function getPhotoUrl(photo) {
  if (photo.normal_ext) {
    return `${getUploadUrl()}/uploads/${project.value.name}/${photo.base_name}${photo.normal_ext}`
  }
  return null
}

async function handleFiles(files) {
  if (!files.length) return

  uploading.value = true
  uploadProgress.value = 0

  try {
    await api.uploadPhotos(projectId.value, Array.from(files), (e) => {
      uploadProgress.value = Math.round((e.loaded * 100) / e.total)
    })
    await fetchData()
  } finally {
    uploading.value = false
    uploadProgress.value = 0
  }
}

function handleDrop(e) {
  e.preventDefault()
  dragOver.value = false
  handleFiles(e.dataTransfer.files)
}

function handleFileSelect(e) {
  handleFiles(e.target.files)
  e.target.value = ''
}

function toggleSelect(photoId) {
  if (selectedPhotos.value.has(photoId)) {
    selectedPhotos.value.delete(photoId)
  } else {
    selectedPhotos.value.add(photoId)
  }
  selectedPhotos.value = new Set(selectedPhotos.value)
}

function selectAll() {
  if (selectedPhotos.value.size === photos.value.length) {
    selectedPhotos.value.clear()
  } else {
    selectedPhotos.value = new Set(photos.value.map(p => p.id))
  }
  selectedPhotos.value = new Set(selectedPhotos.value)
}

async function deleteSelected() {
  if (!selectedPhotos.value.size) return
  if (!confirm(`Delete ${selectedPhotos.value.size} photo(s)?`)) return

  for (const id of selectedPhotos.value) {
    await api.deletePhoto(id)
  }
  selectedPhotos.value.clear()
  await fetchData()
}

async function setCover(photo) {
  await api.updateProject(projectId.value, {
    cover_photo: photo.base_name + photo.normal_ext
  })
  project.value.cover_photo = photo.base_name + photo.normal_ext
}
</script>

<template>
  <div class="min-h-screen">
    <!-- Header -->
    <header class="bg-dark-400 border-b border-dark-200">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
        <div class="flex items-center gap-4">
          <button @click="router.push('/admin')" class="btn btn-secondary">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
            </svg>
          </button>
          <div class="flex-1">
            <h1 class="text-xl font-bold text-white">{{ project?.name || 'Loading...' }}</h1>
            <p class="text-sm text-gray-400">{{ photos.length }} photos</p>
          </div>
          <button
            @click="router.push(`/admin/project/${projectId}/links`)"
            class="btn btn-secondary"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
            </svg>
            Manage Links
          </button>
        </div>
      </div>
    </header>

    <!-- Main content -->
    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <!-- Upload area -->
      <div
        class="card p-8 mb-8 border-2 border-dashed transition-all"
        :class="dragOver ? 'border-primary-500 bg-primary-500/10' : 'border-dark-100'"
        @dragover.prevent="dragOver = true"
        @dragleave="dragOver = false"
        @drop="handleDrop"
      >
        <div class="text-center">
          <svg class="w-12 h-12 mx-auto text-gray-500 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
          </svg>

          <div v-if="uploading" class="space-y-2">
            <div class="h-2 bg-dark-300 rounded-full overflow-hidden max-w-xs mx-auto">
              <div
                class="h-full bg-gradient-to-r from-primary-500 to-primary-400 transition-all"
                :style="{ width: `${uploadProgress}%` }"
              ></div>
            </div>
            <p class="text-gray-400">Uploading... {{ uploadProgress }}%</p>
          </div>

          <div v-else>
            <p class="text-gray-300 mb-2">Drag and drop photos here, or</p>
            <label class="btn btn-primary cursor-pointer">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
              </svg>
              Browse Files
              <input
                type="file"
                class="hidden"
                multiple
                accept="image/*,.raw,.cr2,.cr3,.nef,.arw,.dng,.orf,.rw2,.pef,.raf,.srw,.x3f"
                @change="handleFileSelect"
              />
            </label>
            <p class="text-sm text-gray-500 mt-2">Supports JPG, PNG, RAW formats</p>
          </div>
        </div>
      </div>

      <!-- Toolbar -->
      <div v-if="photos.length" class="flex items-center justify-between mb-4">
        <div class="flex items-center gap-4">
          <button @click="selectAll" class="btn btn-secondary text-sm">
            {{ selectedPhotos.size === photos.length ? 'Deselect All' : 'Select All' }}
          </button>
          <span v-if="selectedPhotos.size" class="text-gray-400">
            {{ selectedPhotos.size }} selected
          </span>
        </div>
        <button
          v-if="selectedPhotos.size"
          @click="deleteSelected"
          class="btn btn-danger text-sm"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
          </svg>
          Delete Selected
        </button>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="flex justify-center py-12">
        <svg class="w-8 h-8 text-primary-500 spinner" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
      </div>

      <!-- Photo grid -->
      <div v-else-if="photos.length" class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-4">
        <div
          v-for="photo in photos"
          :key="photo.id"
          class="group relative aspect-square rounded-xl overflow-hidden bg-dark-300 cursor-pointer"
          :class="selectedPhotos.has(photo.id) ? 'ring-2 ring-primary-500' : ''"
          @click="toggleSelect(photo.id)"
        >
          <!-- Image -->
          <img
            v-if="getPhotoUrl(photo)"
            :src="getPhotoUrl(photo)"
            class="w-full h-full object-cover"
            loading="lazy"
          />
          <div v-else class="w-full h-full flex items-center justify-center text-gray-500">
            RAW
          </div>

          <!-- Overlay -->
          <div class="absolute inset-0 bg-black/50 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center gap-2">
            <button
              @click.stop="setCover(photo)"
              class="p-2 rounded-lg bg-white/10 hover:bg-white/20 text-white"
              title="Set as cover"
            >
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
              </svg>
            </button>
          </div>

          <!-- Selection checkbox -->
          <div
            class="absolute top-2 left-2 w-6 h-6 rounded-full border-2 flex items-center justify-center transition-all"
            :class="selectedPhotos.has(photo.id) ? 'bg-primary-500 border-primary-500' : 'border-white/50 bg-black/30'"
          >
            <svg v-if="selectedPhotos.has(photo.id)" class="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
            </svg>
          </div>

          <!-- RAW badge -->
          <div v-if="photo.has_raw" class="absolute top-2 right-2 px-2 py-0.5 rounded-full bg-primary-500/80 text-white text-xs font-medium">
            RAW
          </div>

          <!-- Filename -->
          <div class="absolute bottom-0 inset-x-0 p-2 bg-gradient-to-t from-black/70 to-transparent">
            <p class="text-xs text-white truncate">{{ photo.base_name }}</p>
          </div>
        </div>
      </div>

      <!-- Empty state -->
      <div v-else class="text-center py-16">
        <div class="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-dark-300 mb-4">
          <svg class="w-8 h-8 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
          </svg>
        </div>
        <h3 class="text-lg font-medium text-white mb-2">No photos yet</h3>
        <p class="text-gray-400">Upload photos to get started</p>
      </div>
    </main>
  </div>
</template>
