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

const showCreateModal = ref(false)
const showEditModal = ref(false)
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

function getShareUrl(link) {
  return `${window.location.origin}/share/${link.token}`
}

function copyLink(link) {
  navigator.clipboard.writeText(getShareUrl(link))
}

async function createLink() {
  try {
    await api.createShareLink(projectId.value, {
      alias: newAlias.value.trim(),
      allow_raw: newAllowRaw.value,
      exclusions: Array.from(newExclusions.value)
    })
    showCreateModal.value = false
    resetForm()
    await fetchData()
  } catch (err) {
    console.error(err)
  }
}

function openEditModal(link) {
  editingLink.value = link
  newAlias.value = link.alias || ''
  newAllowRaw.value = link.allow_raw
  newExclusions.value = new Set((link.exclusions || []).map(e => e.photo_id))
  showEditModal.value = true
}

async function updateLink() {
  try {
    await api.updateShareLink(editingLink.value.id, {
      alias: newAlias.value.trim(),
      allow_raw: newAllowRaw.value,
      exclusions: Array.from(newExclusions.value)
    })
    showEditModal.value = false
    resetForm()
    await fetchData()
  } catch (err) {
    console.error(err)
  }
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

function resetForm() {
  newAlias.value = ''
  newAllowRaw.value = true
  newExclusions.value = new Set()
  editingLink.value = null
}

function openCreateModal() {
  resetForm()
  showCreateModal.value = true
}
</script>

<template>
  <div class="min-h-screen">
    <!-- Header -->
    <header class="bg-white border-b border-cf-border">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
        <div class="flex items-center gap-4">
          <button @click="router.push(`/admin/project/${projectId}`)" class="btn btn-secondary">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
            </svg>
          </button>
          <div class="flex-1">
            <h1 class="text-xl font-bold text-cf-text">分享链接</h1>
            <p class="text-sm text-cf-muted">{{ project?.name || '加载中...' }}</p>
          </div>
          <button @click="openCreateModal" class="btn btn-primary">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
            </svg>
            新建链接
          </button>
        </div>
      </div>
    </header>

    <!-- Main content -->
    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <!-- Loading -->
      <div v-if="loading" class="flex justify-center py-12">
        <svg class="w-8 h-8 text-primary-500 spinner" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
      </div>

      <!-- Empty state -->
      <div v-else-if="!links.length" class="text-center py-16">
        <div class="inline-flex items-center justify-center w-16 h-16 rounded-xl bg-gray-100 mb-4">
          <svg class="w-8 h-8 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
          </svg>
        </div>
        <h3 class="text-lg font-medium text-cf-text mb-2">暂无分享链接</h3>
        <p class="text-cf-muted mb-4">创建链接以便与客户分享此项目</p>
        <button @click="openCreateModal" class="btn btn-primary">
          创建链接
        </button>
      </div>

      <!-- Links list -->
      <div v-else class="space-y-4">
        <div v-for="link in links" :key="link.id" class="card p-6">
          <div class="flex items-start justify-between gap-4">
            <div class="flex-1 min-w-0">
              <h3 class="font-semibold text-cf-text">{{ link.alias || '未命名链接' }}</h3>
              <p class="text-sm text-cf-muted truncate mt-1">{{ getShareUrl(link) }}</p>
              <div class="flex items-center gap-4 mt-3">
                <span v-if="link.allow_raw" class="inline-flex items-center gap-1 text-xs text-primary-600">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                  </svg>
                  允许 RAW
                </span>
                <span v-else class="inline-flex items-center gap-1 text-xs text-cf-muted">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                  禁止 RAW
                </span>
                <span v-if="link.exclusions?.length" class="text-xs text-cf-muted">
                  {{ link.exclusions.length }} 张照片已隐藏
                </span>
              </div>
            </div>
            <div class="flex items-center gap-2">
              <button @click="copyLink(link)" class="btn btn-secondary text-sm">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 5H6a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2v-1M8 5a2 2 0 002 2h2a2 2 0 002-2M8 5a2 2 0 012-2h2a2 2 0 012 2m0 0h2a2 2 0 012 2v3m2 4H10m0 0l3-3m-3 3l3 3" />
                </svg>
                复制
              </button>
              <button @click="openEditModal(link)" class="btn btn-secondary text-sm">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                </svg>
                编辑
              </button>
              <button @click="deleteLink(link)" class="btn btn-secondary text-sm text-red-500 hover:text-red-600 hover:bg-red-50">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                </svg>
              </button>
            </div>
          </div>
        </div>
      </div>
    </main>

    <!-- Create/Edit Modal -->
    <div v-if="showCreateModal || showEditModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/30 overflow-y-auto">
      <div class="card p-6 w-full max-w-2xl my-8" @click.stop>
        <h3 class="text-lg font-semibold text-cf-text mb-4">
          {{ showEditModal ? '编辑链接' : '创建新链接' }}
        </h3>

        <div class="space-y-6">
          <div>
            <label class="label">链接名称</label>
            <input
              v-model="newAlias"
              type="text"
              class="input"
              placeholder="例如：客户名称"
            />
          </div>

          <div class="flex items-center gap-3">
            <button
              @click="newAllowRaw = !newAllowRaw"
              class="relative w-12 h-6 rounded-full transition-colors"
              :class="newAllowRaw ? 'bg-primary-500' : 'bg-gray-200'"
            >
              <span
                class="absolute top-1 w-4 h-4 rounded-full bg-white shadow transition-transform"
                :class="newAllowRaw ? 'left-7' : 'left-1'"
              ></span>
            </button>
            <span class="text-cf-text">允许下载 RAW 文件</span>
          </div>

          <div>
            <label class="label">隐藏的照片（点击切换）</label>
            <p class="text-sm text-cf-muted mb-3">选中的照片将不会在此链接中显示</p>
            <div class="grid grid-cols-4 sm:grid-cols-6 gap-2 max-h-64 overflow-y-auto p-1">
              <div
                v-for="photo in photos"
                :key="photo.id"
                class="aspect-square rounded-lg overflow-hidden cursor-pointer relative"
                :class="newExclusions.has(photo.id) ? 'ring-2 ring-red-500' : 'ring-1 ring-cf-border'"
                @click="toggleExclusion(photo.id)"
              >
                <img
                  v-if="getPhotoUrl(photo)"
                  :src="getPhotoUrl(photo)"
                  class="w-full h-full object-cover"
                  :class="newExclusions.has(photo.id) ? 'opacity-50' : ''"
                />
                <div v-else class="w-full h-full bg-gray-100 flex items-center justify-center text-xs text-cf-muted">
                  RAW
                </div>
                <div
                  v-if="newExclusions.has(photo.id)"
                  class="absolute inset-0 flex items-center justify-center bg-red-500/30"
                >
                  <svg class="w-6 h-6 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
                  </svg>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div class="flex gap-3 mt-6">
          <button
            @click="showCreateModal = false; showEditModal = false; resetForm()"
            class="btn btn-secondary flex-1"
          >
            取消
          </button>
          <button
            @click="showEditModal ? updateLink() : createLink()"
            class="btn btn-primary flex-1"
          >
            {{ showEditModal ? '保存更改' : '创建链接' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
