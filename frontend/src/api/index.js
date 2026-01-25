import axios from 'axios'

const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || 'http://localhost:8080/api',
  timeout: 30000,
})

// Request interceptor
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

// Response interceptor
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      if (!window.location.pathname.startsWith('/share/')) {
        window.location.href = '/login'
      }
    }
    return Promise.reject(error)
  }
)

// Auth
export const login = (username, password) =>
  api.post('/admin/login', { username, password })

// Projects
export const getProjects = () => api.get('/admin/projects')
export const createProject = (data) => api.post('/admin/projects', data)
export const getProject = (id) => api.get(`/admin/projects/${id}`)
export const updateProject = (id, data) => api.put(`/admin/projects/${id}`, data)
export const deleteProject = (id) => api.delete(`/admin/projects/${id}`)

// Photos
export const getProjectPhotos = (projectId) => api.get(`/admin/projects/${projectId}/photos`)
export const uploadPhotos = (projectId, files, onProgress) => {
  const formData = new FormData()
  files.forEach((file) => formData.append('files', file))
  return api.post(`/admin/projects/${projectId}/photos`, formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
    onUploadProgress: onProgress,
  })
}
export const deletePhoto = (id) => api.delete(`/admin/photos/${id}`)

// Share links
export const getShareLinks = (projectId) => api.get(`/admin/projects/${projectId}/links`)
export const createShareLink = (projectId, data) => api.post(`/admin/projects/${projectId}/links`, data)
export const updateShareLink = (id, data) => api.put(`/admin/links/${id}`, data)
export const deleteShareLink = (id) => api.delete(`/admin/links/${id}`)

// Public share
export const getShareInfo = (token) => api.get(`/share/${token}`)
export const getSharePhotos = (token) => api.get(`/share/${token}/photos`)

export const getUploadUrl = () => import.meta.env.VITE_API_URL || 'http://localhost:8080'

export default api
