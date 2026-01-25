<script setup>
import { ref, onMounted, computed, reactive, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import * as api from '../../api'
import { getUploadUrl, fetchAdminThumbSmall, fetchAdminThumbLarge, clearThumbCache, checkHashes } from '../../api'

// FilePond imports
import vueFilePond from 'vue-filepond'
import 'filepond/dist/filepond.min.css'
import 'filepond-plugin-image-preview/dist/filepond-plugin-image-preview.min.css'
import FilePondPluginFileValidateType from 'filepond-plugin-file-validate-type'
import FilePondPluginImagePreview from 'filepond-plugin-image-preview'

const FilePond = vueFilePond(FilePondPluginFileValidateType, FilePondPluginImagePreview)

const route = useRoute()
const router = useRouter()

const project = ref(null)
const photos = ref([])
const links = ref([])
const loading = ref(true)
const selectedPhotos = ref(new Set())

// FilePond
const pond = ref(null)
const uploadedCount = ref(0)
const failedFiles = ref([])
const skippedFiles = ref([])  // Files skipped due to duplicate hash

// File hash cache: filename -> hash
const fileHashCache = new Map()

// Link management
const showLinkModal = ref(false)
const editingLink = ref(null)
const newAlias = ref('')
const newAllowRaw = ref(true)
const newExclusions = ref(new Set())

// Photo preview with EXIF and files
const previewPhoto = ref(null)
const previewExif = ref(null)
const previewFiles = ref([])
const loadingExif = ref(false)
const fullImageLoaded = ref(false)

// Thumbnail blob URLs cache (reactive for Vue reactivity)
const thumbUrls = reactive({})
const largeThumbUrls = reactive({})

const projectId = computed(() => route.params.id)

onUnmounted(() => {
  // Clean up blob URLs when component unmounts
  clearThumbCache()
})

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

    // Load thumbnails asynchronously (don't block UI)
    for (const photo of photos.value) {
      if (photo.normal_ext) {
        loadThumbSmall(photo)
      }
    }
  } finally {
    loading.value = false
  }
}

function getPhotoUrl(photo) {
  if (photo.normal_ext) {
    // URL编码项目名称和文件名，防止特殊字符问题
    const encodedProject = encodeURIComponent(project.value.name)
    const encodedBaseName = encodeURIComponent(photo.base_name)
    return `${getUploadUrl()}/uploads/${encodedProject}/${encodedBaseName}${photo.normal_ext}`
  }
  return null
}

// 异步加载缩略图
async function loadThumbSmall(photo) {
  if (thumbUrls[photo.id]) return
  const url = await fetchAdminThumbSmall(photo.id)
  if (url) {
    thumbUrls[photo.id] = url
  } else {
    // Fallback to original image URL
    thumbUrls[photo.id] = getPhotoUrl(photo) || 'error'
  }
}

async function loadThumbLarge(photo) {
  if (largeThumbUrls[photo.id]) return
  const url = await fetchAdminThumbLarge(photo.id)
  if (url) {
    largeThumbUrls[photo.id] = url
  } else {
    // Fallback to original image URL
    largeThumbUrls[photo.id] = getPhotoUrl(photo) || 'error'
  }
}

// 获取已加载的缩略图URL
function getThumbSmallUrl(photo) {
  return thumbUrls[photo.id] || null
}

function getThumbLargeUrl(photo) {
  return largeThumbUrls[photo.id] || null
}

// 当前预加载的照片ID（防止快速切换时状态混乱）
let currentPreloadingId = null

// 预加载原图
function preloadFullImage(photo) {
  const url = getPhotoUrl(photo)
  if (!url) return

  const photoId = photo.id
  currentPreloadingId = photoId

  const img = new Image()
  img.onload = () => {
    // 只有当仍然是当前预加载的照片时才更新状态
    if (currentPreloadingId === photoId) {
      fullImageLoaded.value = true
    }
  }
  img.src = url
}

// 缩略图加载失败时，尝试用原图
function handleThumbError(event, photo) {
  const img = event.target
  const url = getPhotoUrl(photo)
  if (url && !img.dataset.fallback) {
    img.dataset.fallback = 'true'
    img.src = url
  } else if (!img.dataset.failed) {
    // 最终降级：标记为失败，隐藏图片
    img.dataset.failed = 'true'
    img.style.display = 'none'
  }
}

// 预览缩略图加载失败处理
function handlePreviewThumbError(event) {
  const img = event.target
  const url = getPhotoUrl(previewPhoto.value)
  if (url && !img.dataset.fallback) {
    img.dataset.fallback = 'true'
    img.src = url
    fullImageLoaded.value = true
  } else if (!img.dataset.failed) {
    // 最终降级：标记为失败
    img.dataset.failed = 'true'
  }
}

// FilePond server configuration
const filePondServer = computed(() => ({
  process: async (fieldName, file, metadata, load, error, progress, abort) => {
    let aborted = false
    const abortController = { abort: () => { aborted = true } }

    try {
      // Calculate file hash
      progress(true, 0, 100)  // Show indeterminate progress during hash calculation
      const hash = await calculateFileHash(file)

      if (aborted) return abortController

      // Check if this hash already exists on server
      const response = await checkHashes(projectId.value, [hash])
      if (aborted) return abortController

      if (response.data.existing && response.data.existing.includes(hash)) {
        // File already exists, skip upload
        skippedFiles.value.push(file.name)
        load('skipped')  // Mark as completed but skipped
        return abortController
      }

      // Proceed with upload
      const formData = new FormData()
      formData.append('files', file)

      const token = localStorage.getItem('token')
      const xhr = new XMLHttpRequest()

      xhr.open('POST', `${getUploadUrl()}/api/admin/projects/${projectId.value}/photos`)
      xhr.setRequestHeader('Authorization', `Bearer ${token}`)

      xhr.upload.onprogress = (e) => {
        progress(e.lengthComputable, e.loaded, e.total)
      }

      xhr.onload = () => {
        if (xhr.status >= 200 && xhr.status < 300) {
          uploadedCount.value++
          load(xhr.responseText)
        } else {
          failedFiles.value.push(file.name)
          error('Upload failed')
        }
      }

      xhr.onerror = () => {
        failedFiles.value.push(file.name)
        error('Network error')
      }

      abortController.abort = () => {
        xhr.abort()
        abort()
      }

      xhr.send(formData)
    } catch (err) {
      failedFiles.value.push(file.name)
      error('Hash check failed')
    }

    return abortController
  }
}))

// FilePond accepted file types
const acceptedFileTypes = [
  'image/*',
  'image/x-canon-cr2',
  'image/x-canon-cr3',
  'image/x-nikon-nef',
  'image/x-sony-arw',
  'image/x-adobe-dng',
  'image/x-olympus-orf',
  'image/x-panasonic-rw2',
  'image/x-pentax-pef',
  'image/x-fuji-raf',
  'image/x-samsung-srw',
  'image/x-sigma-x3f'
]

// Calculate SHA-256 hash of a file
async function calculateFileHash(file) {
  const buffer = await file.arrayBuffer()
  const hashBuffer = await crypto.subtle.digest('SHA-256', buffer)
  const hashArray = Array.from(new Uint8Array(hashBuffer))
  return hashArray.map(b => b.toString(16).padStart(2, '0')).join('')
}

function handleFilePondInit() {
  // Reset counters when FilePond initializes
  uploadedCount.value = 0
  failedFiles.value = []
  skippedFiles.value = []
  fileHashCache.clear()
}

function handleProcessFiles() {
  // Called when all files have been processed
  let message = `上传完成: ${uploadedCount.value} 个文件成功`
  if (skippedFiles.value.length > 0) {
    message += `, ${skippedFiles.value.length} 个文件已存在(跳过)`
  }
  if (failedFiles.value.length > 0) {
    message += `, ${failedFiles.value.length} 个文件失败`
  }
  if (skippedFiles.value.length > 0 || failedFiles.value.length > 0) {
    alert(message)
  }
  // Refresh photo list
  fetchData()
  // Reset counters
  uploadedCount.value = 0
  failedFiles.value = []
  skippedFiles.value = []
  fileHashCache.clear()
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
  if (!confirm(`确定要删除 ${selectedPhotos.value.size} 张照片吗？`)) return

  const ids = Array.from(selectedPhotos.value)
  const results = await Promise.allSettled(ids.map(id => api.deletePhoto(id)))

  // 检查失败的删除
  const failed = results.filter(r => r.status === 'rejected')
  if (failed.length > 0) {
    alert(`${failed.length} 张照片删除失败`)
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

// Preview with EXIF and files
async function openPreview(photo) {
  previewPhoto.value = photo
  previewExif.value = null
  previewFiles.value = []
  loadingExif.value = true
  fullImageLoaded.value = false

  // 加载大缩略图
  loadThumbLarge(photo)

  // 开始预加载原图
  preloadFullImage(photo)

  try {
    const [exifRes, filesRes] = await Promise.all([
      api.getAdminPhotoExif(photo.id),
      api.getAdminPhotoFiles(photo.id)
    ])
    previewExif.value = exifRes.data
    previewFiles.value = filesRes.data || []
  } catch (err) {
    previewExif.value = {}
    previewFiles.value = []
  } finally {
    loadingExif.value = false
  }
}

function closePreview() {
  previewPhoto.value = null
  previewExif.value = null
  previewFiles.value = []
  fullImageLoaded.value = false
}

// File download helpers
function getExtLabel(ext) {
  const labels = {
    '.jpg': 'JPG',
    '.jpeg': 'JPEG',
    '.png': 'PNG',
    '.gif': 'GIF',
    '.webp': 'WebP',
    '.tiff': 'TIFF',
    '.tif': 'TIFF',
    '.arw': 'ARW (Sony RAW)',
    '.cr2': 'CR2 (Canon RAW)',
    '.cr3': 'CR3 (Canon RAW)',
    '.nef': 'NEF (Nikon RAW)',
    '.dng': 'DNG (Adobe RAW)',
    '.orf': 'ORF (Olympus RAW)',
    '.rw2': 'RW2 (Panasonic RAW)',
    '.pef': 'PEF (Pentax RAW)',
    '.raf': 'RAF (Fujifilm RAW)',
    '.srw': 'SRW (Samsung RAW)',
    '.x3f': 'X3F (Sigma RAW)',
    '.raw': 'RAW'
  }
  return labels[ext.toLowerCase()] || ext.toUpperCase().replace('.', '')
}

function downloadFile(url, filename) {
  const fullUrl = getUploadUrl() + url
  const a = document.createElement('a')
  a.href = fullUrl
  a.download = filename
  a.target = '_blank'
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
}

async function downloadAllFiles() {
  if (!previewFiles.value.length) return
  // Download each file with a small delay
  for (const file of previewFiles.value) {
    downloadFile(file.url, file.filename)
    await new Promise(r => setTimeout(r, 300))
  }
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
  if (!confirm(`确定要删除链接 "${link.alias || link.token}" 吗？`)) return
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
            <h1 class="text-xl font-bold text-white">{{ project?.name || '加载中...' }}</h1>
            <p class="text-sm text-gray-400">{{ photos.length }} 张照片 · {{ links.length }} 个链接</p>
          </div>
        </div>
      </div>
    </header>

    <!-- Main content - Two columns -->
    <main class="max-w-[1800px] mx-auto px-4 sm:px-6 lg:px-8 py-6">
      <div class="flex gap-6">
        <!-- Left: Photos -->
        <div class="flex-1 min-w-0">
          <!-- FilePond Upload area -->
          <div class="filepond-dark mb-6">
            <FilePond
              ref="pond"
              name="files"
              class-name="filepond--panel-root filepond--drop-label"
              label-idle="拖拽文件到此处或 <span class='filepond--label-action'>浏览文件</span>"
              :allow-multiple="true"
              :allow-reorder="true"
              :server="filePondServer"
              :accepted-file-types="acceptedFileTypes"
              :max-parallel-uploads="3"
              :instant-upload="true"
              credits=""
              @init="handleFilePondInit"
              @processfiles="handleProcessFiles"
            />
          </div>

          <!-- Toolbar -->
          <div v-if="photos.length" class="flex items-center justify-between mb-4">
            <div class="flex items-center gap-3">
              <button @click="selectAll" class="btn btn-secondary text-sm py-1.5">
                {{ selectedPhotos.size === photos.length ? '取消全选' : '全选' }}
              </button>
              <span v-if="selectedPhotos.size" class="text-sm text-gray-400">已选择 {{ selectedPhotos.size }} 张</span>
            </div>
            <button v-if="selectedPhotos.size" @click="deleteSelected" class="btn btn-danger text-sm py-1.5">
              删除
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
              <!-- 有缩略图URL时显示图片 -->
              <img v-if="photo.normal_ext && getThumbSmallUrl(photo)" :src="getThumbSmallUrl(photo)" class="w-full h-full object-cover" loading="lazy" @error="handleThumbError($event, photo)" />
              <!-- 正在加载缩略图时显示加载器 -->
              <div v-else-if="photo.normal_ext && !getThumbSmallUrl(photo)" class="w-full h-full flex items-center justify-center bg-dark-300">
                <svg class="w-6 h-6 text-gray-500 spinner" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
                </svg>
              </div>
              <!-- 只有RAW时显示提示 -->
              <div v-else class="w-full h-full flex flex-col items-center justify-center text-gray-500">
                <svg class="w-6 h-6 mb-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                </svg>
                <span class="text-[10px]">只有RAW</span>
              </div>

              <div class="absolute inset-0 bg-black/50 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center gap-2">
                <button @click.stop="openPreview(photo)" class="p-1.5 rounded bg-white/20 hover:bg-white/30 text-white" title="预览">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                  </svg>
                </button>
                <button @click.stop="setCover(photo)" class="p-1.5 rounded bg-white/20 hover:bg-white/30 text-white" title="设为封面">
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
            <p class="text-gray-400">暂无照片</p>
          </div>
        </div>

        <!-- Right: Links -->
        <div class="w-80 flex-shrink-0">
          <div class="card p-4 sticky top-4">
            <div class="flex items-center justify-between mb-4">
              <h2 class="font-semibold text-white">分享链接</h2>
              <button @click="openCreateModal" class="btn btn-primary text-sm py-1.5 px-3">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
                </svg>
                新建
              </button>
            </div>

            <!-- Links list -->
            <div v-if="links.length" class="space-y-3">
              <div v-for="link in links" :key="link.id" class="p-3 rounded-xl bg-dark-300 group">
                <div class="flex items-start justify-between gap-2 mb-2">
                  <div class="min-w-0">
                    <p class="font-medium text-white text-sm truncate">{{ link.alias || '未命名' }}</p>
                    <p class="text-xs text-gray-500 font-mono truncate">/share/{{ link.token }}</p>
                  </div>
                  <div class="flex gap-1">
                    <button @click="copyLink(link)" class="p-1.5 rounded hover:bg-dark-200 text-gray-400 hover:text-white" title="复制链接">
                      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 5H6a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2v-1M8 5a2 2 0 002 2h2a2 2 0 002-2M8 5a2 2 0 012-2h2a2 2 0 012 2m0 0h2a2 2 0 012 2v3m2 4H10m0 0l3-3m-3 3l3 3" />
                      </svg>
                    </button>
                    <button @click="openEditModal(link)" class="p-1.5 rounded hover:bg-dark-200 text-gray-400 hover:text-white" title="编辑">
                      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                      </svg>
                    </button>
                    <button @click="deleteLink(link)" class="p-1.5 rounded hover:bg-dark-200 text-gray-400 hover:text-red-400" title="删除">
                      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                      </svg>
                    </button>
                  </div>
                </div>
                <div class="flex items-center gap-2 text-xs">
                  <span v-if="link.allow_raw" class="text-primary-400">允许RAW</span>
                  <span v-else class="text-gray-500">禁止RAW</span>
                  <span v-if="link.exclusions?.length" class="text-gray-500">· {{ link.exclusions.length }} 张隐藏</span>
                </div>
              </div>
            </div>

            <div v-else class="text-center py-8 text-gray-500 text-sm">
              <p>暂无链接</p>
            </div>
          </div>
        </div>
      </div>
    </main>

    <!-- Photo Preview Modal with EXIF -->
    <div v-if="previewPhoto" class="fixed inset-0 z-50 bg-black/90 flex" @click="closePreview">
      <!-- Close button -->
      <button class="absolute top-4 right-4 p-2 rounded-full bg-white/10 hover:bg-white/20 text-white z-10" @click.stop="closePreview">
        <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>

      <!-- Left: Image -->
      <div class="flex-1 flex items-center justify-center p-8" @click.stop>
        <div class="relative max-w-full max-h-full">
          <!-- 只有RAW时显示提示 -->
          <div v-if="!previewPhoto.normal_ext" class="flex flex-col items-center justify-center text-gray-400 py-20">
            <svg class="w-16 h-16 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
            <span class="text-lg">只有RAW文件</span>
            <span class="text-sm text-gray-500 mt-1">无法预览，请下载查看</span>
          </div>
          <!-- 大缩略图作为占位（有URL时显示） -->
          <img
            v-else-if="!fullImageLoaded && getThumbLargeUrl(previewPhoto)"
            :src="getThumbLargeUrl(previewPhoto)"
            class="max-w-full max-h-full object-contain"
            @error="handlePreviewThumbError"
          />
          <!-- 大缩略图正在加载时显示加载器 -->
          <div v-else-if="!fullImageLoaded && !getThumbLargeUrl(previewPhoto)" class="flex items-center justify-center py-20">
            <svg class="w-12 h-12 text-primary-500 spinner" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
            </svg>
          </div>
          <!-- 原图 (加载完成后显示) -->
          <img
            v-else
            :src="getPhotoUrl(previewPhoto)"
            class="max-w-full max-h-full object-contain"
          />
          <!-- 加载指示器 -->
          <div v-if="previewPhoto.normal_ext && !fullImageLoaded" class="absolute bottom-2 right-2 px-2 py-1 rounded bg-black/50 text-white text-xs flex items-center gap-1">
            <svg class="w-3 h-3 spinner" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
            </svg>
            加载原图...
          </div>
        </div>
      </div>

      <!-- Right: EXIF Info -->
      <div class="w-80 bg-dark-400 border-l border-dark-200 overflow-y-auto" @click.stop>
        <div class="p-6">
          <h3 class="text-lg font-semibold text-white mb-4">照片信息</h3>

          <div class="space-y-4">
            <!-- File name -->
            <div>
              <p class="text-xs text-gray-500 uppercase tracking-wide mb-1">文件名</p>
              <p class="text-white text-sm">{{ previewPhoto.base_name }}{{ previewPhoto.normal_ext }}</p>
            </div>

            <!-- Files download section -->
            <div v-if="previewFiles.length" class="pt-2 pb-2 border-t border-b border-dark-200">
              <p class="text-xs text-gray-500 uppercase tracking-wide mb-2">可下载文件</p>
              <div class="space-y-2">
                <div v-for="file in previewFiles" :key="file.type" class="flex items-center justify-between gap-2 p-2 rounded-lg bg-dark-300">
                  <div class="min-w-0">
                    <p class="text-white text-sm truncate">{{ file.filename }}</p>
                    <p class="text-xs text-gray-500">{{ getExtLabel(file.ext) }}</p>
                  </div>
                  <button @click="downloadFile(file.url, file.filename)" class="flex-shrink-0 p-1.5 rounded bg-primary-500/20 hover:bg-primary-500/40 text-primary-400" title="下载">
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                    </svg>
                  </button>
                </div>
              </div>
              <!-- Download all button -->
              <button v-if="previewFiles.length > 1" @click="downloadAllFiles" class="w-full mt-3 btn btn-primary text-sm py-2">
                <svg class="w-4 h-4 mr-1.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                </svg>
                下载全部文件
              </button>
            </div>

            <!-- Loading -->
            <div v-if="loadingExif" class="flex justify-center py-8">
              <svg class="w-6 h-6 text-primary-500 spinner" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
              </svg>
            </div>

            <!-- EXIF Data -->
            <template v-else-if="previewExif">
              <!-- Shooting params -->
              <div v-if="previewExif.focal_length || previewExif.aperture || previewExif.shutter_speed || previewExif.iso" class="grid grid-cols-2 gap-3">
                <div v-if="previewExif.focal_length">
                  <p class="text-xs text-gray-500 uppercase tracking-wide mb-1">焦距</p>
                  <p class="text-white text-sm">{{ previewExif.focal_length }}</p>
                </div>
                <div v-if="previewExif.aperture">
                  <p class="text-xs text-gray-500 uppercase tracking-wide mb-1">光圈</p>
                  <p class="text-white text-sm">{{ previewExif.aperture }}</p>
                </div>
                <div v-if="previewExif.shutter_speed">
                  <p class="text-xs text-gray-500 uppercase tracking-wide mb-1">快门</p>
                  <p class="text-white text-sm">{{ previewExif.shutter_speed }}</p>
                </div>
                <div v-if="previewExif.iso">
                  <p class="text-xs text-gray-500 uppercase tracking-wide mb-1">感光度</p>
                  <p class="text-white text-sm">{{ previewExif.iso }}</p>
                </div>
              </div>

              <!-- Dimensions -->
              <div v-if="previewExif.width && previewExif.height">
                <p class="text-xs text-gray-500 uppercase tracking-wide mb-1">尺寸</p>
                <p class="text-white text-sm">{{ previewExif.width }} x {{ previewExif.height }}</p>
              </div>

              <!-- Date -->
              <div v-if="previewExif.date_time">
                <p class="text-xs text-gray-500 uppercase tracking-wide mb-1">拍摄时间</p>
                <p class="text-white text-sm">{{ previewExif.date_time }}</p>
              </div>

              <!-- Other info -->
              <div v-if="previewExif.exposure_mode || previewExif.white_balance || previewExif.metering_mode" class="grid grid-cols-2 gap-3">
                <div v-if="previewExif.exposure_mode">
                  <p class="text-xs text-gray-500 uppercase tracking-wide mb-1">曝光模式</p>
                  <p class="text-white text-sm">{{ previewExif.exposure_mode }}</p>
                </div>
                <div v-if="previewExif.white_balance">
                  <p class="text-xs text-gray-500 uppercase tracking-wide mb-1">白平衡</p>
                  <p class="text-white text-sm">{{ previewExif.white_balance }}</p>
                </div>
                <div v-if="previewExif.metering_mode">
                  <p class="text-xs text-gray-500 uppercase tracking-wide mb-1">测光模式</p>
                  <p class="text-white text-sm">{{ previewExif.metering_mode }}</p>
                </div>
                <div v-if="previewExif.flash">
                  <p class="text-xs text-gray-500 uppercase tracking-wide mb-1">闪光灯</p>
                  <p class="text-white text-sm">{{ previewExif.flash }}</p>
                </div>
              </div>

              <!-- GPS -->
              <div v-if="previewExif.gps_latitude && previewExif.gps_longitude">
                <p class="text-xs text-gray-500 uppercase tracking-wide mb-1">GPS 位置</p>
                <p class="text-white text-sm">{{ previewExif.gps_latitude }}, {{ previewExif.gps_longitude }}</p>
              </div>

              <!-- Software -->
              <div v-if="previewExif.software">
                <p class="text-xs text-gray-500 uppercase tracking-wide mb-1">软件</p>
                <p class="text-white text-sm">{{ previewExif.software }}</p>
              </div>

              <!-- No EXIF -->
              <div v-if="!previewExif.focal_length && !previewExif.date_time" class="text-gray-500 text-sm">
                无可用的 EXIF 信息
              </div>
            </template>
          </div>
        </div>
      </div>
    </div>

    <!-- Link Modal -->
    <div v-if="showLinkModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60">
      <div class="card p-5 w-full max-w-lg max-h-[80vh] overflow-y-auto" @click.stop>
        <h3 class="text-lg font-semibold text-white mb-4">{{ editingLink ? '编辑链接' : '创建链接' }}</h3>

        <div class="space-y-4">
          <div>
            <label class="label">链接名称</label>
            <input v-model="newAlias" type="text" class="input" placeholder="客户名称" />
          </div>

          <div class="flex items-center gap-3">
            <button @click="newAllowRaw = !newAllowRaw" class="relative w-10 h-5 rounded-full transition-colors" :class="newAllowRaw ? 'bg-primary-500' : 'bg-dark-200'">
              <span class="absolute top-0.5 w-4 h-4 rounded-full bg-white transition-transform" :class="newAllowRaw ? 'left-5' : 'left-0.5'"></span>
            </button>
            <span class="text-sm text-gray-300">允许下载 RAW 文件</span>
          </div>

          <div>
            <label class="label">隐藏的照片</label>
            <div class="grid grid-cols-6 gap-1.5 max-h-48 overflow-y-auto p-1">
              <div
                v-for="photo in photos"
                :key="photo.id"
                class="aspect-square rounded overflow-hidden cursor-pointer relative"
                :class="newExclusions.has(photo.id) ? 'ring-2 ring-red-500' : 'ring-1 ring-dark-100'"
                @click="toggleExclusion(photo.id)"
              >
                <img v-if="photo.normal_ext && getThumbSmallUrl(photo)" :src="getThumbSmallUrl(photo)" class="w-full h-full object-cover" :class="newExclusions.has(photo.id) ? 'opacity-40' : ''" @error="handleThumbError($event, photo)" />
                <div v-else-if="photo.normal_ext && !getThumbSmallUrl(photo)" class="w-full h-full bg-dark-300 flex items-center justify-center">
                  <svg class="w-4 h-4 text-gray-500 spinner" fill="none" viewBox="0 0 24 24">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
                  </svg>
                </div>
                <div v-else class="w-full h-full bg-dark-300 flex items-center justify-center text-[8px] text-gray-500">只有RAW</div>
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
          <button @click="showLinkModal = false" class="btn btn-secondary flex-1">取消</button>
          <button @click="saveLink" class="btn btn-primary flex-1">{{ editingLink ? '保存' : '创建' }}</button>
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

<style>
/* FilePond dark theme */
.filepond-dark .filepond--panel-root {
  background-color: #1e1e2e;
  border: 2px dashed #3d3d5c;
  border-radius: 0.75rem;
}

.filepond-dark .filepond--drop-label {
  color: #9ca3af;
  font-size: 0.9rem;
}

.filepond-dark .filepond--drop-label label {
  cursor: pointer;
}

.filepond-dark .filepond--label-action {
  color: #8b5cf6;
  text-decoration: underline;
  text-decoration-color: rgba(139, 92, 246, 0.4);
}

.filepond-dark .filepond--label-action:hover {
  color: #a78bfa;
}

.filepond-dark .filepond--item-panel {
  background-color: #2d2d44;
  border-radius: 0.5rem;
}

.filepond-dark .filepond--file-action-button {
  cursor: pointer;
  color: white;
  background-color: rgba(0, 0, 0, 0.5);
}

.filepond-dark .filepond--file-action-button:hover {
  background-color: rgba(0, 0, 0, 0.7);
}

.filepond-dark .filepond--file {
  color: white;
}

.filepond-dark .filepond--file-info {
  color: rgba(255, 255, 255, 0.85);
}

.filepond-dark .filepond--file-info .filepond--file-info-sub {
  color: rgba(255, 255, 255, 0.5);
}

.filepond-dark .filepond--file-status {
  color: rgba(255, 255, 255, 0.65);
}

.filepond-dark .filepond--drip-blob {
  background-color: #8b5cf6;
}

/* Processing state */
.filepond-dark .filepond--item[data-filepond-item-state='processing'] .filepond--item-panel {
  background-color: #3730a3;
}

.filepond-dark .filepond--item[data-filepond-item-state='processing-complete'] .filepond--item-panel {
  background-color: #065f46;
}

/* Error state */
.filepond-dark [data-filepond-item-state='processing-error'] .filepond--item-panel,
.filepond-dark [data-filepond-item-state='load-error'] .filepond--item-panel {
  background-color: #991b1b;
}

/* Progress indicator */
.filepond-dark .filepond--progress-indicator {
  color: white;
}

.filepond-dark .filepond--load-indicator {
  color: white;
}

/* Image preview plugin */
.filepond-dark .filepond--image-preview-overlay-idle {
  color: rgba(255, 255, 255, 0.85);
}

.filepond-dark .filepond--image-preview-wrapper {
  border-radius: 0.375rem;
  overflow: hidden;
}
</style>
