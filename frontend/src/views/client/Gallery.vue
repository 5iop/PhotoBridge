<script setup>
import { ref, onMounted, computed, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import * as api from '../../api'
import { getUploadUrl, getShareThumbSmallUrl, getShareThumbLargeUrl } from '../../api'

const route = useRoute()

const info = ref(null)
const photos = ref([])
const loading = ref(true)
const error = ref('')

const lightboxPhoto = ref(null)
const lightboxIndex = ref(0)
const lightboxExif = ref(null)
const loadingExif = ref(false)
const fullImageLoaded = ref(false)
const isFullscreen = ref(false)
const imageAspect = ref(1) // 图片宽高比

const showDownloadModal = ref(false)
const downloadType = ref('normal')

const token = computed(() => route.params.token)

onMounted(async () => {
  await fetchData()
  window.addEventListener('keydown', handleKeydown)
  document.addEventListener('fullscreenchange', handleFullscreenChange)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
  document.removeEventListener('fullscreenchange', handleFullscreenChange)
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
    error.value = err.response?.data?.error || '加载失败'
  } finally {
    loading.value = false
  }
}

function getPhotoUrl(photo) {
  return `${getUploadUrl()}${photo.normal_url}`
}

// 获取缩略图URL
function getThumbSmallUrl(photo) {
  return getShareThumbSmallUrl(token.value, photo.id)
}

function getThumbLargeUrl(photo) {
  return getShareThumbLargeUrl(token.value, photo.id)
}

// 当前预加载的照片ID（防止快速切换时状态混乱）
let currentPreloadingId = null

// 预加载原图并获取尺寸
function preloadFullImage(photo) {
  if (!photo.normal_url) return

  const photoId = photo.id
  currentPreloadingId = photoId

  const img = new Image()
  img.onload = () => {
    // 只有当仍然是当前预加载的照片时才更新状态
    if (currentPreloadingId === photoId) {
      fullImageLoaded.value = true
      // 计算宽高比
      if (img.width && img.height) {
        imageAspect.value = img.width / img.height
      }
    }
  }
  img.src = getPhotoUrl(photo)
}

// 获取当前照片的宽高比
function getCurrentAspect() {
  // 优先使用已加载的原图宽高比
  if (imageAspect.value !== 1) {
    return imageAspect.value
  }

  // 其次使用照片数据中的缩略图尺寸
  if (lightboxPhoto.value?.thumb_width && lightboxPhoto.value?.thumb_height) {
    return lightboxPhoto.value.thumb_width / lightboxPhoto.value.thumb_height
  }

  // 最后尝试从当前显示的图片元素获取
  const imgElement = document.querySelector('#fullscreen-container img')
  if (imgElement && imgElement.naturalWidth && imgElement.naturalHeight) {
    return imgElement.naturalWidth / imgElement.naturalHeight
  }

  return 1 // 默认值
}

// 全屏模式
async function enterFullscreen() {
  const container = document.getElementById('fullscreen-container')
  if (!container) return

  try {
    if (container.requestFullscreen) {
      await container.requestFullscreen()
    } else if (container.webkitRequestFullscreen) {
      await container.webkitRequestFullscreen()
    }
    isFullscreen.value = true

    // 根据图片宽高比锁定屏幕方向
    if (screen.orientation && screen.orientation.lock) {
      try {
        const aspect = getCurrentAspect()
        if (aspect > 1) {
          // 横向图片，锁定横屏
          await screen.orientation.lock('landscape')
        } else {
          // 竖向图片，锁定竖屏
          await screen.orientation.lock('portrait')
        }
      } catch (e) {
        // 部分浏览器不支持方向锁定
        console.log('Orientation lock not supported')
      }
    }
  } catch (e) {
    console.log('Fullscreen not supported')
  }
}

function exitFullscreen() {
  if (document.exitFullscreen) {
    document.exitFullscreen()
  } else if (document.webkitExitFullscreen) {
    document.webkitExitFullscreen()
  }
  isFullscreen.value = false

  // 解锁屏幕方向
  if (screen.orientation && screen.orientation.unlock) {
    try {
      screen.orientation.unlock()
    } catch (e) {}
  }
}

// 监听全屏变化
function handleFullscreenChange() {
  isFullscreen.value = !!document.fullscreenElement
  if (!document.fullscreenElement && screen.orientation && screen.orientation.unlock) {
    try {
      screen.orientation.unlock()
    } catch (e) {}
  }
}

// 缩略图加载失败时，尝试用原图或显示占位图
function handleThumbError(event, photo) {
  const img = event.target
  // 如果有原图，降级使用原图（仅一次）
  if (photo.normal_url && !img.dataset.fallback) {
    img.dataset.fallback = 'true'
    img.src = getPhotoUrl(photo)
  } else if (!img.dataset.failed) {
    // 最终降级：标记为失败，显示空状态
    img.dataset.failed = 'true'
    img.style.display = 'none'
    // 父元素会显示只有RAW的提示样式（通过隐藏图片触发）
  }
}

// 灯箱缩略图加载失败处理
function handleLightboxThumbError(event) {
  const img = event.target
  if (lightboxPhoto.value?.normal_url && !img.dataset.fallback) {
    img.dataset.fallback = 'true'
    img.src = getPhotoUrl(lightboxPhoto.value)
    fullImageLoaded.value = true // 直接显示原图
  } else if (!img.dataset.failed) {
    // 最终降级：标记为失败
    img.dataset.failed = 'true'
  }
}

// Get files for current lightbox photo
function getPhotoFiles(photo) {
  if (!photo) return []
  const files = []
  if (photo.normal_url) {
    files.push({
      type: 'normal',
      filename: photo.base_name + (photo.normal_ext || '.jpg'),
      url: photo.normal_url,
      ext: photo.normal_ext || '.jpg'
    })
  }
  if (photo.has_raw && photo.raw_url && info.value?.allow_raw) {
    files.push({
      type: 'raw',
      filename: photo.base_name + photo.raw_ext,
      url: photo.raw_url,
      ext: photo.raw_ext
    })
  }
  return files
}

function getExtLabel(ext) {
  const labels = {
    '.jpg': 'JPG',
    '.jpeg': 'JPEG',
    '.png': 'PNG',
    '.gif': 'GIF',
    '.webp': 'WebP',
    '.arw': 'ARW (Sony RAW)',
    '.cr2': 'CR2 (Canon RAW)',
    '.cr3': 'CR3 (Canon RAW)',
    '.nef': 'NEF (Nikon RAW)',
    '.dng': 'DNG (Adobe RAW)',
    '.orf': 'ORF (Olympus RAW)',
    '.rw2': 'RW2 (Panasonic RAW)',
    '.pef': 'PEF (Pentax RAW)',
    '.raf': 'RAF (Fuji RAW)',
    '.raw': 'RAW'
  }
  return labels[ext?.toLowerCase()] || ext?.toUpperCase().replace('.', '') || 'FILE'
}

function downloadFile(url, filename) {
  const a = document.createElement('a')
  a.href = getUploadUrl() + url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
}

function downloadPhotoPackage(photo) {
  const url = `${getUploadUrl()}/api/share/${token.value}/photo/${photo.id}/download`
  const a = document.createElement('a')
  a.href = url
  a.download = ''
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
}

async function openLightbox(index) {
  lightboxIndex.value = index
  lightboxPhoto.value = photos.value[index]
  lightboxExif.value = null
  loadingExif.value = true
  fullImageLoaded.value = false

  // 开始预加载原图
  preloadFullImage(photos.value[index])

  try {
    const res = await api.getPhotoExif(token.value, photos.value[index].id)
    lightboxExif.value = res.data
  } catch (err) {
    lightboxExif.value = {}
  } finally {
    loadingExif.value = false
  }
}

function closeLightbox() {
  if (isFullscreen.value) {
    exitFullscreen()
  }
  lightboxPhoto.value = null
  lightboxExif.value = null
  imageAspect.value = 1
}

async function prevPhoto() {
  const newIndex = (lightboxIndex.value - 1 + photos.value.length) % photos.value.length
  await openLightbox(newIndex)
}

async function nextPhoto() {
  const newIndex = (lightboxIndex.value + 1) % photos.value.length
  await openLightbox(newIndex)
}

function handleKeydown(e) {
  if (!lightboxPhoto.value) return
  if (e.key === 'ArrowLeft') prevPhoto()
  if (e.key === 'ArrowRight') nextPhoto()
  if (e.key === 'Escape') closeLightbox()
}

function download() {
  const url = `${getUploadUrl()}/api/share/${token.value}/download?type=${downloadType.value}`

  const a = document.createElement('a')
  a.href = url
  a.download = ''
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)

  // 关闭模态框（无法追踪实际下载进度，所以直接关闭）
  showDownloadModal.value = false
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
        <div class="inline-flex items-center justify-center w-16 h-16 rounded-xl bg-red-500/20 mb-4">
          <svg class="w-8 h-8 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
        </div>
        <h2 class="text-xl font-bold text-white mb-2">出错了</h2>
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
              <p class="text-sm text-gray-400 mt-1">{{ info.photo_count }} 张照片</p>
            </div>
            <button @click="showDownloadModal = true" class="btn btn-primary">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
              </svg>
              <span class="hidden sm:inline">下载全部</span>
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
            class="aspect-square rounded-lg sm:rounded-xl overflow-hidden bg-dark-300 cursor-pointer group relative"
            @click="openLightbox(index)"
          >
            <!-- 有普通图片时显示缩略图 -->
            <img
              v-if="photo.normal_url"
              :src="getThumbSmallUrl(photo)"
              class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
              loading="lazy"
              @error="handleThumbError($event, photo)"
            />
            <!-- 只有RAW时显示提示 -->
            <div v-else class="w-full h-full flex flex-col items-center justify-center text-gray-400">
              <svg class="w-8 h-8 mb-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
              </svg>
              <span class="text-xs">只有RAW</span>
            </div>
            <div v-if="photo.has_raw && info.allow_raw" class="absolute top-2 right-2 px-2 py-0.5 rounded-full bg-primary-500/80 text-white text-xs font-medium">
              RAW
            </div>
          </div>
        </div>
      </main>
    </div>

    <!-- Lightbox with EXIF and Files -->
    <div
      v-if="lightboxPhoto"
      id="fullscreen-container"
      class="fixed inset-0 z-50 bg-black"
      :class="isFullscreen ? 'flex items-center justify-center' : ''"
    >
      <!-- 全屏模式 -->
      <template v-if="isFullscreen">
        <div class="relative w-full h-full flex items-center justify-center" @click="exitFullscreen">
          <!-- 退出全屏按钮 -->
          <button
            class="absolute top-4 right-4 p-2 rounded-full bg-white/20 text-white z-10"
            @click.stop="exitFullscreen"
          >
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>

          <!-- 左右切换按钮 -->
          <button class="absolute left-4 top-1/2 -translate-y-1/2 p-3 rounded-full bg-white/20 text-white z-10" @click.stop="prevPhoto">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
            </svg>
          </button>
          <button class="absolute right-4 top-1/2 -translate-y-1/2 p-3 rounded-full bg-white/20 text-white z-10" @click.stop="nextPhoto">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
            </svg>
          </button>

          <!-- 图片 -->
          <img
            v-if="lightboxPhoto.normal_url && fullImageLoaded"
            :src="getPhotoUrl(lightboxPhoto)"
            class="max-w-full max-h-full object-contain"
            @click.stop
          />
          <img
            v-else-if="lightboxPhoto.normal_url"
            :src="getThumbLargeUrl(lightboxPhoto)"
            class="max-w-full max-h-full object-contain"
            @click.stop
          />

          <!-- 计数器 -->
          <div class="absolute bottom-4 left-1/2 -translate-x-1/2 px-4 py-2 rounded-full bg-white/20 text-white text-sm">
            {{ lightboxIndex + 1 }} / {{ photos.length }}
          </div>
        </div>
      </template>

      <!-- 移动端普通模式：上下布局 -->
      <template v-else>
        <div class="lg:hidden h-full flex flex-col overflow-hidden">
          <!-- 顶部图片区域 -->
          <div class="relative flex-shrink-0 bg-black" style="height: 45vh;">
            <!-- 关闭按钮 -->
            <button class="absolute top-3 right-3 p-2 rounded-full bg-white/20 text-white z-10" @click="closeLightbox">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>

            <!-- 全屏按钮 -->
            <button class="absolute top-3 left-3 p-2 rounded-full bg-white/20 text-white z-10" @click="enterFullscreen">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" />
              </svg>
            </button>

            <!-- 左右切换 -->
            <button class="absolute left-2 top-1/2 -translate-y-1/2 p-2 rounded-full bg-white/20 text-white z-10" @click="prevPhoto">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
              </svg>
            </button>
            <button class="absolute right-2 top-1/2 -translate-y-1/2 p-2 rounded-full bg-white/20 text-white z-10" @click="nextPhoto">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
              </svg>
            </button>

            <!-- 图片 -->
            <div class="w-full h-full flex items-center justify-center p-2">
              <div v-if="!lightboxPhoto.normal_url" class="flex flex-col items-center justify-center text-gray-400">
                <svg class="w-12 h-12 mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                </svg>
                <span class="text-sm">只有RAW文件</span>
              </div>
              <img
                v-else-if="fullImageLoaded"
                :src="getPhotoUrl(lightboxPhoto)"
                class="max-w-full max-h-full object-contain"
              />
              <img
                v-else
                :src="getThumbLargeUrl(lightboxPhoto)"
                class="max-w-full max-h-full object-contain"
                @error="handleLightboxThumbError"
              />
            </div>

            <!-- 计数器 -->
            <div class="absolute bottom-2 left-1/2 -translate-x-1/2 px-3 py-1 rounded-full bg-white/20 text-white text-xs">
              {{ lightboxIndex + 1 }} / {{ photos.length }}
            </div>
          </div>

          <!-- 底部信息区域（可滚动） -->
          <div class="flex-1 overflow-y-auto bg-dark-400">
            <div class="p-4">
              <!-- 文件名 -->
              <h3 class="text-base font-semibold text-white mb-3">{{ lightboxPhoto.base_name }}</h3>

              <!-- 下载文件 -->
              <div class="mb-4">
                <p class="text-xs text-gray-500 uppercase tracking-wide mb-2">下载</p>
                <div class="flex flex-wrap gap-2">
                  <button
                    v-for="file in getPhotoFiles(lightboxPhoto)"
                    :key="file.url"
                    @click="downloadFile(file.url, file.filename)"
                    class="flex items-center gap-2 px-3 py-2 rounded-lg bg-dark-300 text-white text-sm"
                  >
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                    </svg>
                    {{ getExtLabel(file.ext) }}
                  </button>
                  <button
                    v-if="getPhotoFiles(lightboxPhoto).length > 1"
                    @click="downloadPhotoPackage(lightboxPhoto)"
                    class="flex items-center gap-2 px-3 py-2 rounded-lg bg-primary-500 text-white text-sm"
                  >
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                    </svg>
                    打包全部
                  </button>
                </div>
              </div>

              <!-- EXIF -->
              <div>
                <p class="text-xs text-gray-500 uppercase tracking-wide mb-2">拍摄参数</p>
                <div v-if="loadingExif" class="flex justify-center py-4">
                  <svg class="w-5 h-5 text-primary-500 spinner" fill="none" viewBox="0 0 24 24">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
                  </svg>
                </div>
                <div v-else-if="lightboxExif" class="grid grid-cols-2 gap-2 text-sm">
                  <div v-if="lightboxExif.focal_length" class="bg-dark-300 rounded-lg p-2">
                    <p class="text-gray-500 text-xs">焦距</p>
                    <p class="text-white">{{ lightboxExif.focal_length }}</p>
                  </div>
                  <div v-if="lightboxExif.aperture" class="bg-dark-300 rounded-lg p-2">
                    <p class="text-gray-500 text-xs">光圈</p>
                    <p class="text-white">{{ lightboxExif.aperture }}</p>
                  </div>
                  <div v-if="lightboxExif.shutter_speed" class="bg-dark-300 rounded-lg p-2">
                    <p class="text-gray-500 text-xs">快门</p>
                    <p class="text-white">{{ lightboxExif.shutter_speed }}</p>
                  </div>
                  <div v-if="lightboxExif.iso" class="bg-dark-300 rounded-lg p-2">
                    <p class="text-gray-500 text-xs">ISO</p>
                    <p class="text-white">{{ lightboxExif.iso }}</p>
                  </div>
                  <div v-if="lightboxExif.date_time" class="bg-dark-300 rounded-lg p-2 col-span-2">
                    <p class="text-gray-500 text-xs">拍摄时间</p>
                    <p class="text-white">{{ lightboxExif.date_time }}</p>
                  </div>
                </div>
                <p v-else class="text-gray-500 text-sm">无 EXIF 信息</p>
              </div>
            </div>
          </div>
        </div>

        <!-- 桌面端：左右布局 -->
        <div class="hidden lg:flex h-full" @click="closeLightbox">
          <!-- 关闭按钮 -->
          <button class="absolute top-4 right-4 p-2 rounded-full bg-white/10 hover:bg-white/20 text-white z-20" @click.stop="closeLightbox">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>

          <!-- 图片区域 -->
          <div class="flex-1 flex items-center justify-center relative" @click.stop>
            <button class="absolute left-4 top-1/2 -translate-y-1/2 p-3 rounded-full bg-white/20 hover:bg-white/30 text-white" @click.stop="prevPhoto">
              <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
              </svg>
            </button>

            <div class="relative max-w-[calc(100%-100px)] max-h-[90vh]">
              <div v-if="!lightboxPhoto.normal_url" class="flex flex-col items-center justify-center text-gray-400 py-20">
                <svg class="w-16 h-16 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                </svg>
                <span class="text-lg">只有RAW文件</span>
                <span class="text-sm text-gray-500 mt-1">无法预览，请下载查看</span>
              </div>
              <img v-else-if="fullImageLoaded" :src="getPhotoUrl(lightboxPhoto)" class="max-w-full max-h-[90vh] object-contain" />
              <img v-else :src="getThumbLargeUrl(lightboxPhoto)" class="max-w-full max-h-[90vh] object-contain" @error="handleLightboxThumbError" />
              <div v-if="lightboxPhoto.normal_url && !fullImageLoaded" class="absolute bottom-2 right-2 px-2 py-1 rounded bg-black/50 text-white text-xs flex items-center gap-1">
                <svg class="w-3 h-3 spinner" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
                </svg>
                加载原图...
              </div>
            </div>

            <button class="absolute right-4 top-1/2 -translate-y-1/2 p-3 rounded-full bg-white/20 hover:bg-white/30 text-white" @click.stop="nextPhoto">
              <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
              </svg>
            </button>

            <div class="absolute bottom-4 left-1/2 -translate-x-1/2 px-4 py-2 rounded-full bg-white/20 text-white text-sm">
              {{ lightboxIndex + 1 }} / {{ photos.length }}
            </div>
          </div>

          <!-- 右侧信息面板 -->
          <div class="w-80 bg-dark-400 border-l border-dark-200 overflow-y-auto" @click.stop>
            <div class="p-6">
              <!-- File name header -->
              <h3 class="text-lg font-semibold text-white mb-1">{{ lightboxPhoto.base_name }}</h3>
              <p class="text-sm text-gray-500 mb-6">照片详情</p>

          <!-- Files Section -->
          <div class="mb-6">
            <p class="text-xs text-gray-500 uppercase tracking-wide mb-3">包含文件</p>
            <div class="space-y-2">
              <div
                v-for="file in getPhotoFiles(lightboxPhoto)"
                :key="file.url"
                class="flex items-center justify-between p-3 rounded-xl bg-dark-300"
              >
                <div class="flex items-center gap-3 min-w-0">
                  <div class="w-10 h-10 rounded-lg flex items-center justify-center" :class="file.type === 'raw' ? 'bg-primary-500/20' : 'bg-gray-500/20'">
                    <svg class="w-5 h-5" :class="file.type === 'raw' ? 'text-primary-400' : 'text-gray-400'" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                    </svg>
                  </div>
                  <div class="min-w-0">
                    <p class="text-white text-sm font-medium truncate">{{ getExtLabel(file.ext) }}</p>
                    <p class="text-xs text-gray-500 truncate">{{ file.filename }}</p>
                  </div>
                </div>
                <button
                  @click="downloadFile(file.url, file.filename)"
                  class="p-2 rounded-lg hover:bg-dark-200 text-gray-400 hover:text-white transition-colors"
                  title="下载"
                >
                  <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                  </svg>
                </button>
              </div>
            </div>

            <!-- Download All Button -->
            <button
              v-if="getPhotoFiles(lightboxPhoto).length > 1"
              @click="downloadPhotoPackage(lightboxPhoto)"
              class="w-full mt-3 btn btn-primary py-2.5"
            >
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
              打包下载全部文件
            </button>
          </div>

          <!-- Divider -->
          <div class="border-t border-dark-200 my-6"></div>

          <!-- EXIF Section -->
          <div>
            <p class="text-xs text-gray-500 uppercase tracking-wide mb-3">拍摄参数</p>

            <!-- Loading -->
            <div v-if="loadingExif" class="flex justify-center py-8">
              <svg class="w-6 h-6 text-primary-500 spinner" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
              </svg>
            </div>

            <!-- EXIF Data -->
            <div v-else-if="lightboxExif" class="space-y-4">
              <!-- Shooting params -->
              <div v-if="lightboxExif.focal_length || lightboxExif.aperture || lightboxExif.shutter_speed || lightboxExif.iso" class="grid grid-cols-2 gap-3">
                <div v-if="lightboxExif.focal_length">
                  <p class="text-xs text-gray-500 mb-1">焦距</p>
                  <p class="text-white text-sm">{{ lightboxExif.focal_length }}</p>
                </div>
                <div v-if="lightboxExif.aperture">
                  <p class="text-xs text-gray-500 mb-1">光圈</p>
                  <p class="text-white text-sm">{{ lightboxExif.aperture }}</p>
                </div>
                <div v-if="lightboxExif.shutter_speed">
                  <p class="text-xs text-gray-500 mb-1">快门</p>
                  <p class="text-white text-sm">{{ lightboxExif.shutter_speed }}</p>
                </div>
                <div v-if="lightboxExif.iso">
                  <p class="text-xs text-gray-500 mb-1">感光度</p>
                  <p class="text-white text-sm">{{ lightboxExif.iso }}</p>
                </div>
              </div>

              <!-- Dimensions -->
              <div v-if="lightboxExif.width && lightboxExif.height">
                <p class="text-xs text-gray-500 mb-1">尺寸</p>
                <p class="text-white text-sm">{{ lightboxExif.width }} x {{ lightboxExif.height }}</p>
              </div>

              <!-- Date -->
              <div v-if="lightboxExif.date_time">
                <p class="text-xs text-gray-500 mb-1">拍摄时间</p>
                <p class="text-white text-sm">{{ lightboxExif.date_time }}</p>
              </div>

              <!-- Other info -->
              <div v-if="lightboxExif.exposure_mode || lightboxExif.white_balance || lightboxExif.metering_mode || lightboxExif.flash" class="grid grid-cols-2 gap-3">
                <div v-if="lightboxExif.exposure_mode">
                  <p class="text-xs text-gray-500 mb-1">曝光模式</p>
                  <p class="text-white text-sm">{{ lightboxExif.exposure_mode }}</p>
                </div>
                <div v-if="lightboxExif.white_balance">
                  <p class="text-xs text-gray-500 mb-1">白平衡</p>
                  <p class="text-white text-sm">{{ lightboxExif.white_balance }}</p>
                </div>
                <div v-if="lightboxExif.metering_mode">
                  <p class="text-xs text-gray-500 mb-1">测光模式</p>
                  <p class="text-white text-sm">{{ lightboxExif.metering_mode }}</p>
                </div>
                <div v-if="lightboxExif.flash">
                  <p class="text-xs text-gray-500 mb-1">闪光灯</p>
                  <p class="text-white text-sm">{{ lightboxExif.flash }}</p>
                </div>
              </div>

              <!-- GPS -->
              <div v-if="lightboxExif.gps_latitude && lightboxExif.gps_longitude">
                <p class="text-xs text-gray-500 mb-1">GPS 位置</p>
                <p class="text-white text-sm">{{ lightboxExif.gps_latitude }}, {{ lightboxExif.gps_longitude }}</p>
              </div>

              <!-- Software -->
              <div v-if="lightboxExif.software">
                <p class="text-xs text-gray-500 mb-1">软件</p>
                <p class="text-white text-sm">{{ lightboxExif.software }}</p>
              </div>

              <!-- No EXIF -->
              <div v-if="!lightboxExif.focal_length && !lightboxExif.date_time" class="text-gray-500 text-sm">
                无可用的 EXIF 信息
              </div>
            </div>
          </div>
        </div>
      </div>
        </div>
      </template>
    </div>

    <!-- Download Modal -->
    <div v-if="showDownloadModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50">
      <div class="card p-6 w-full max-w-sm" @click.stop>
        <h3 class="text-lg font-semibold text-white mb-4">下载照片</h3>

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
              <p class="font-medium text-white">普通照片</p>
              <p class="text-sm text-gray-400">JPG 格式，适合网络分享</p>
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
              <p class="font-medium text-white">仅 RAW 文件</p>
              <p class="text-sm text-gray-400">原始画质 RAW 格式</p>
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
              <p class="font-medium text-white">全部文件</p>
              <p class="text-sm text-gray-400">普通照片 + RAW 文件</p>
            </div>
          </label>
        </div>

        <div class="flex gap-3 mt-6">
          <button @click="showDownloadModal = false" class="btn btn-secondary flex-1">
            取消
          </button>
          <button @click="download" class="btn btn-primary flex-1">
            下载
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
