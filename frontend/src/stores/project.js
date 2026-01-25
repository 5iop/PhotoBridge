import { defineStore } from 'pinia'
import { ref } from 'vue'
import * as api from '../api'

export const useProjectStore = defineStore('project', () => {
  const projects = ref([])
  const currentProject = ref(null)
  const loading = ref(false)

  async function fetchProjects() {
    loading.value = true
    try {
      const response = await api.getProjects()
      projects.value = response.data || []
    } finally {
      loading.value = false
    }
  }

  async function fetchProject(id) {
    loading.value = true
    try {
      const response = await api.getProject(id)
      currentProject.value = response.data
    } finally {
      loading.value = false
    }
  }

  async function createProject(data) {
    const response = await api.createProject(data)
    projects.value.push(response.data)
    return response.data
  }

  async function deleteProject(id) {
    await api.deleteProject(id)
    projects.value = projects.value.filter(p => p.id !== id)
  }

  return {
    projects,
    currentProject,
    loading,
    fetchProjects,
    fetchProject,
    createProject,
    deleteProject
  }
})
