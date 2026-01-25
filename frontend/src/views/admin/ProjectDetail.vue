<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import * as api from '../../api'
import { getUploadUrl } from '../../api'

const route = useRoute()
const router = useRouter()

const project = ref(null)
const photos = ref([])
const links = ref([])
const loading = ref(true)
const uploading = ref(false)
const uploadProgress = ref(0)
const dragOver = ref(false)
const selectedPhotos = ref(new Set())

// Link management
const showLinkModal = ref(false)
const editingLink = ref(null)
const newAlias = ref('')
const newAllowRaw = ref(true)
const newExclusions = ref(new Set())

const projectId = computed(() => route.params.id)

onMounted(async () => {
  await fetchData()
})

async function fetchData() {
  loading.value = true
  try {
    const [projectRes, photosRes, linksRes] = await Promise.all([
      api.getProject(projectId.value),
      api.getProjectPhotos(projectId.value),
      api.getShareLinks(projectId.value)
    ])
    project.value = projectRes.data
    photos.value = photosRes.data || []
    links.value = linksRes.data || []
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

// Link functions
function getShareUrl(link) {
  return `${window.location.origin}/share/${link.token}`
}

function copyLink(link) {
  navigator.clipboard.writeText(getShareUrl(link))
}

function openCreateModal() {
  editingLink.value = null
  newAlias.value = ''
  newAllowRaw.value = true
  newExclusions.value = new Set()
  showLinkModal.value = true
}

function openEditModal(link) {
  editingLink.value = link
  newAlias.value = link.alias || ''
  newAllowRaw.value = link.allow_raw
  newExclusions.value = new Set((link.exclusions || []).map(e => e.photo_id))
  showLinkModal.value = true
}

async function saveLink() {
  const data = {
    alias: newAlias.value.trim(),
    allow_raw: newAllowRaw.value,
    exclusions: Array.from(newExclusions.value)
  }

  if (editingLink.value) {
    await api.updateShareLink(editingLink.value.id, data)
  } else {
    await api.createShareLink(projectId.value, data)
  }

  showLinkModal.value = false
  await fetchData()
}

async function deleteLink(link) {
  if (!confirm(`Delete link "${link.alias || link.token}"?`)) return
  await api.deleteShareLink(link.id)
  await fetchData()
}

function toggleExclusion(photoId) {
  if (newExclusions.value.has(photoId)) {
    newExclusions.value.delete(photoId)
  } else {
    newExclusions.value.add(photoId)
  }
  newExclusions.value = new Set(newExclusions.value)
}
</script>

<template>
  <div class="min-h-screen">
    <!-- Header -->
    <header class="bg-dark-400 border-b border-dark-200">
      <div class="max-w-[1800px] mx-auto px-4 sm:px-6 lg:px-8 py-4">
        <div class="flex items-center gap-4">
          <button @click="router.push('/admin')" class="btn btn-secondary">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
            </svg>
          </button>
          <div class="flex-1">
            <h1 class="text-xl font-bold text-white">{{ project?.name || 'Loading...' }}</h1>
            <p class="text-sm text-gray-400">{{ photos.length }} photos · {{ links.length }} links</p>
          </div>
        </div>
      </div>
    </header>

    <!-- Main content - Two columns -->
    <main class="max-w-[1800px] mx-auto px-4 sm:px-6 lg:px-8 py-6">
      <div class="flex gap-6">
        <!-- Left: Photos -->
        <div class="flex-1 min-w-0">
          <!-- Upload area -->
          <div
            class="card p-6 mb-6 border-2 border-dashed transition-all"
            :class="dragOver ? 'border-primary-500 bg-primary-500/10' : 'border-dark-100'"
            @dragover.prevent="dragOver = true"
            @dragleave="dragOver = false"
            @drop="handleDrop"
          >
            <div class="text-center">
              <div v-if="uploading" class="space-y-2">
                <div class="h-2 bg-dark-300 rounded-full overflow-hidden max-w-xs mx-auto">
                  <div class="h-full bg-gradient-to-r from-primary-500 to-primary-400 transition-all" :style="{ width: `${uploadProgress}%` }"></div>
                </div>
                <p class="text-gray-400 text-sm">Uploading... {{ uploadProgress }}%</p>
              </div>
              <div v-else class="flex items-center justify-center gap-4">
                <svg class="w-8 h-8 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
                </svg>
                <span class="text-gray-400">Drop files or</span>
                <label class="btn btn-primary btn-sm cursor-pointer">
                  Browse
                  <input type="file" class="hidden" multiple accept="image/*,.raw,.cr2,.cr3,.nef,.arw,.dng,.orf,.rw2,.pef,.raf,.srw,.x3f" @change="handleFileSelect" />
                </label>
              </div>
            </div>
          </div>

          <!-- Toolbar -->
          <div v-if="photos.length" class="flex items-center justify-between mb-4">
            <div class="flex items-center gap-3">
              <button @click="selectAll" class="btn btn-secondary text-sm py-1.5">
                {{ selectedPhotos.size === photos.length ? 'Deselect' : 'Select All' }}
              </button>
              <span v-if="selectedPhotos.size" class="text-sm text-gray-400">{{ selectedPhotos.size }} selected</span>
            </div>
            <button v-if="selectedPhotos.size" @click="deleteSelected" class="btn btn-danger text-sm py-1.5">
              Delete
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
          <div v-else-if="photos.length" class="grid grid-cols-3 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-6 gap-3">
            <div
              v-for="photo in photos"
              :key="photo.id"
              class="group relative aspect-square rounded-lg overflow-hidden bg-dark-300 cursor-pointer"
              :class="selectedPhotos.has(photo.id) ? 'ring-2 ring-primary-500' : ''"
              @click="toggleSelect(photo.id)"
            >
              <img v-if="getPhotoUrl(photo)" :src="getPhotoUrl(photo)" class="w-full h-full object-cover" loading="lazy" />
              <div v-else class="w-full h-full flex items-center justify-center text-gray-500 text-xs">RAW</div>

              <div class="absolute inset-0 bg-black/50 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center">
                <button @click.stop="setCover(photo)" class="p-1.5 rounded bg-white/20 hover:bg-white/30 text-white" title="Set cover">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                  </svg>
                </button>
              </div>

              <div class="absolute top-1.5 left-1.5 w-5 h-5 rounded-full border-2 flex items-center justify-center" :class="selectedPhotos.has(photo.id) ? 'bg-primary-500 border-primary-500' : 'border-white/50 bg-black/30'">
                <svg v-if="selectedPhotos.has(photo.id)" class="w-3 h-3 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
                </svg>
              </div>

              <div v-if="photo.has_raw" class="absolute top-1.5 right-1.5 px-1.5 py-0.5 rounded bg-primary-500/80 text-white text-[10px] font-medium">RAW</div>
            </div>
          </div>

          <!-- Empty -->
          <div v-else class="text-center py-12">
            <svg class="w-12 h-12 mx-auto text-gray-600 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
            <p class="text-gray-400">No photos yet</p>
          </div>
        </div>

        <!-- Right: Links -->
        <div class="w-80 flex-shrink-0">
          <div class="card p-4 sticky top-4">
            <div class="flex items-center justify-between mb-4">
              <h2 class="font-semibold text-white">Share Links</h2>
              <button @click="openCreateModal" class="btn btn-primary text-sm py-1.5 px-3">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
                </svg>
                New
              </button>
            </div>

            <!-- Links list -->
            <div v-if="links.length" class="space-y-3">
              <div v-for="link in links" :key="link.id" class="p-3 rounded-xl bg-dark-300 group">
                <div class="flex items-start justify-between gap-2 mb-2">
                  <div class="min-w-0">
                    <p class="font-medium text-white text-sm truncate">{{ link.alias || 'Untitled' }}</p>
                    <p class="text-xs text-gray-500 font-mono truncate">/share/{{ link.token }}</p>
                  </div>
                  <div class="flex gap-1">
                    <button @click="copyLink(link)" class="p-1.5 rounded hover:bg-dark-200 text-gray-400 hover:text-white" title="Copy">
                      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 5H6a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2v-1M8 5a2 2 0 002 2h2a2 2 0 002-2M8 5a2 2 0 012-2h2a2 2 0 012 2m0 0h2a2 2 0 012 2v3m2 4H10m0 0l3-3m-3 3l3 3" />
                      </svg>
                    </button>
                    <button @click="openEditModal(link)" class="p-1.5 rounded hover:bg-dark-200 text-gray-400 hover:text-white" title="Edit">
                      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                      </svg>
                    </button>
                    <button @click="deleteLink(link)" class="p-1.5 rounded hover:bg-dark-200 text-gray-400 hover:text-red-400" title="Delete">
                      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                      </svg>
                    </button>
                  </div>
                </div>
                <div class="flex items-center gap-2 text-xs">
                  <span v-if="link.allow_raw" class="text-primary-400">RAW</span>
                  <span v-else class="text-gray-500">No RAW</span>
                  <span v-if="link.exclusions?.length" class="text-gray-500">· {{ link.exclusions.length }} hidden</span>
                </div>
              </div>
            </div>

            <div v-else class="text-center py-8 text-gray-500 text-sm">
              <p>No links yet</p>
            </div>
          </div>
        </div>
      </div>
    </main>

    <!-- Link Modal -->
    <div v-if="showLinkModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60">
      <div class="card p-5 w-full max-w-lg max-h-[80vh] overflow-y-auto" @click.stop>
        <h3 class="text-lg font-semibold text-white mb-4">{{ editingLink ? 'Edit Link' : 'Create Link' }}</h3>

        <div class="space-y-4">
          <div>
            <label class="label">Alias</label>
            <input v-model="newAlias" type="text" class="input" placeholder="Client name" />
          </div>

          <div class="flex items-center gap-3">
            <button @click="newAllowRaw = !newAllowRaw" class="relative w-10 h-5 rounded-full transition-colors" :class="newAllowRaw ? 'bg-primary-500' : 'bg-dark-200'">
              <span class="absolute top-0.5 w-4 h-4 rounded-full bg-white transition-transform" :class="newAllowRaw ? 'left-5' : 'left-0.5'"></span>
            </button>
            <span class="text-sm text-gray-300">Allow RAW downloads</span>
          </div>

          <div>
            <label class="label">Hidden Photos</label>
            <div class="grid grid-cols-6 gap-1.5 max-h-48 overflow-y-auto p-1">
              <div
                v-for="photo in photos"
                :key="photo.id"
                class="aspect-square rounded overflow-hidden cursor-pointer relative"
                :class="newExclusions.has(photo.id) ? 'ring-2 ring-red-500' : 'ring-1 ring-dark-100'"
                @click="toggleExclusion(photo.id)"
              >
                <img v-if="getPhotoUrl(photo)" :src="getPhotoUrl(photo)" class="w-full h-full object-cover" :class="newExclusions.has(photo.id) ? 'opacity-40' : ''" />
                <div v-else class="w-full h-full bg-dark-300 flex items-center justify-center text-[8px] text-gray-500">RAW</div>
                <div v-if="newExclusions.has(photo.id)" class="absolute inset-0 flex items-center justify-center">
                  <svg class="w-4 h-4 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div class="flex gap-3 mt-5">
          <button @click="showLinkModal = false" class="btn btn-secondary flex-1">Cancel</button>
          <button @click="saveLink" class="btn btn-primary flex-1">{{ editingLink ? 'Save' : 'Create' }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.btn-sm {
  @apply py-1.5 px-3 text-sm;
}
</style>
