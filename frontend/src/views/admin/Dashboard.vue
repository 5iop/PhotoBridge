<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useProjectStore } from '../../stores/project'
import { useAuthStore } from '../../stores/auth'
import { getUploadUrl } from '../../api'
import Modal from '../../components/Modal.vue'

const router = useRouter()
const projectStore = useProjectStore()
const auth = useAuthStore()

const showCreateModal = ref(false)
const newProjectName = ref('')
const newProjectDesc = ref('')
const creating = ref(false)

onMounted(() => {
  projectStore.fetchProjects()
})

async function createProject() {
  if (!newProjectName.value.trim()) return

  creating.value = true
  try {
    const project = await projectStore.createProject({
      name: newProjectName.value.trim(),
      description: newProjectDesc.value.trim()
    })
    showCreateModal.value = false
    newProjectName.value = ''
    newProjectDesc.value = ''
    router.push(`/admin/project/${project.id}`)
  } finally {
    creating.value = false
  }
}

async function handleDelete(project) {
  if (confirm(`确定要删除项目 "${project.name}" 吗？`)) {
    await projectStore.deleteProject(project.id)
  }
}

function logout() {
  auth.logout()
  router.push('/login')
}

function getCoverUrl(project) {
  if (project.cover_photo) {
    const encodedName = encodeURIComponent(project.name)
    const encodedCover = encodeURIComponent(project.cover_photo)
    return `${getUploadUrl()}/uploads/${encodedName}/${encodedCover}`
  }
  return null
}
</script>

<template>
  <div class="min-h-screen">
    <!-- Header -->
    <header class="bg-white border-b border-cf-border">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4 flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-lg bg-primary-500 flex items-center justify-center">
            <svg class="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
          </div>
          <h1 class="text-xl font-bold text-cf-text">PhotoBridge</h1>
        </div>
        <button @click="logout" class="btn btn-secondary text-sm">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
          </svg>
          退出登录
        </button>
      </div>
    </header>

    <!-- Main content -->
    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <div class="flex items-center justify-between mb-8">
        <h2 class="text-2xl font-bold text-cf-text">项目列表</h2>
        <button @click="showCreateModal = true" class="btn btn-primary">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
          </svg>
          新建项目
        </button>
      </div>

      <!-- Loading -->
      <div v-if="projectStore.loading" class="flex justify-center py-12">
        <svg class="w-8 h-8 text-primary-500 spinner" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
      </div>

      <!-- Empty state -->
      <div v-else-if="!projectStore.projects.length" class="text-center py-16">
        <div class="inline-flex items-center justify-center w-16 h-16 rounded-xl bg-gray-100 mb-4">
          <svg class="w-8 h-8 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
          </svg>
        </div>
        <h3 class="text-lg font-medium text-cf-text mb-2">暂无项目</h3>
        <p class="text-cf-muted mb-4">创建您的第一个项目开始使用</p>
        <button @click="showCreateModal = true" class="btn btn-primary">
          创建项目
        </button>
      </div>

      <!-- Project grid -->
      <div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
        <div
          v-for="project in projectStore.projects"
          :key="project.id"
          class="card group cursor-pointer hover:border-primary-500 hover:shadow-md transition-all"
          @click="router.push(`/admin/project/${project.id}`)"
        >
          <!-- Cover image -->
          <div class="aspect-[4/3] bg-gray-100 relative overflow-hidden">
            <img
              v-if="getCoverUrl(project)"
              :src="getCoverUrl(project)"
              class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
              alt=""
            />
            <div v-else class="w-full h-full flex items-center justify-center">
              <svg class="w-12 h-12 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
              </svg>
            </div>
          </div>

          <!-- Info -->
          <div class="p-4">
            <h3 class="font-semibold text-cf-text truncate">{{ project.name }}</h3>
            <p class="text-sm text-cf-muted mt-1">{{ project.photo_count || 0 }} 张照片</p>
          </div>

          <!-- Actions -->
          <div class="px-4 pb-4 flex gap-2">
            <button
              @click.stop="router.push(`/admin/project/${project.id}/links`)"
              class="btn btn-secondary text-sm flex-1"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
              </svg>
              分享链接
            </button>
            <button
              @click.stop="handleDelete(project)"
              class="btn btn-secondary text-sm text-red-500 hover:text-red-600 hover:bg-red-50"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
            </button>
          </div>
        </div>
      </div>
    </main>

    <!-- Create Modal -->
    <Modal :show="showCreateModal" title="新建项目" @close="showCreateModal = false">
      <div class="space-y-4">
        <div>
          <label class="label">项目名称</label>
          <input
            v-model="newProjectName"
            type="text"
            class="input"
            placeholder="例如：婚礼摄影 2024"
          />
        </div>

        <div>
          <label class="label">项目描述（可选）</label>
          <textarea
            v-model="newProjectDesc"
            class="input resize-none"
            rows="3"
            placeholder="输入项目描述..."
          ></textarea>
        </div>
      </div>

      <div class="flex gap-3 mt-6">
        <button @click="showCreateModal = false" class="btn btn-secondary flex-1">
          取消
        </button>
        <button @click="createProject" class="btn btn-primary flex-1" :disabled="creating">
          {{ creating ? '创建中...' : '创建' }}
        </button>
      </div>
    </Modal>
  </div>
</template>
