import axios from 'axios'

const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || '/api',
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
export const deletePhoto = (id) => api.delete(`/admin/photos/${id}`)
export const checkHashes = (projectId, hashes) => api.post(`/admin/projects/${projectId}/photos/check-hashes`, { hashes })

// Share links
export const getShareLinks = (projectId) => api.get(`/admin/projects/${projectId}/links`)
export const createShareLink = (projectId, data) => api.post(`/admin/projects/${projectId}/links`, data)
export const updateShareLink = (id, data) => api.put(`/admin/links/${id}`, data)
export const deleteShareLink = (id) => api.delete(`/admin/links/${id}`)

// Public share
export const getShareInfo = (token) => api.get(`/share/${token}`)
export const getSharePhotos = (token) => api.get(`/share/${token}/photos`)
export const getPhotoExif = (token, photoId) => api.get(`/share/${token}/photo/${photoId}/exif`)
export const verifySharePassword = (token, password) =>
  api.post(`/share/${token}/verify-password`, { password })

// Admin EXIF and files
export const getAdminPhotoExif = (photoId) => api.get(`/admin/photos/${photoId}/exif`)
export const getAdminPhotoFiles = (photoId) => api.get(`/admin/photos/${photoId}/files`)

// Get base URL for uploads/downloads (without /api suffix)
// This is used for file paths like /uploads/... or /api/share/.../download
export const getUploadUrl = () => {
  let baseUrl = ''

  if (import.meta.env.VITE_API_URL) {
    baseUrl = import.meta.env.VITE_API_URL
  } else if (import.meta.env.DEV) {
    // 开发模式下默认使用后端地址
    baseUrl = 'http://localhost:8060'
  }

  // Remove trailing /api to avoid duplication (e.g., /api/api/share/...)
  // VITE_API_URL is used as axios baseURL which can include /api
  // But upload/download paths already include /api or /uploads
  if (baseUrl.endsWith('/api')) {
    baseUrl = baseUrl.slice(0, -4)
  }

  return baseUrl
}

// Thumbnail URLs (share routes don't need auth)
// cdnBaseUrl: optional CDN base URL (from backend cdn_base_url field)
export const getShareThumbSmallUrl = (token, photoId, cdnBaseUrl = '') => {
  const baseUrl = cdnBaseUrl || getUploadUrl()
  return `${baseUrl}/api/share/${token}/photo/${photoId}/thumb/small`
}
export const getShareThumbLargeUrl = (token, photoId, cdnBaseUrl = '') => {
  const baseUrl = cdnBaseUrl || getUploadUrl()
  return `${baseUrl}/api/share/${token}/photo/${photoId}/thumb/large`
}

// Admin thumbnail fetchers - return blob URLs with auth
const thumbCache = new Map()

export const fetchAdminThumbSmall = async (photoId) => {
  const cacheKey = `small-${photoId}`
  if (thumbCache.has(cacheKey)) {
    return thumbCache.get(cacheKey)
  }
  try {
    const response = await api.get(`/admin/photos/${photoId}/thumb/small`, {
      responseType: 'blob'
    })
    const blobUrl = URL.createObjectURL(response.data)
    thumbCache.set(cacheKey, blobUrl)
    return blobUrl
  } catch (err) {
    // Return null to trigger fallback
    return null
  }
}

export const fetchAdminThumbLarge = async (photoId) => {
  const cacheKey = `large-${photoId}`
  if (thumbCache.has(cacheKey)) {
    return thumbCache.get(cacheKey)
  }
  try {
    const response = await api.get(`/admin/photos/${photoId}/thumb/large`, {
      responseType: 'blob'
    })
    const blobUrl = URL.createObjectURL(response.data)
    thumbCache.set(cacheKey, blobUrl)
    return blobUrl
  } catch (err) {
    return null
  }
}

// Clear thumbnail cache (call when logging out or when photos change)
export const clearThumbCache = () => {
  for (const url of thumbCache.values()) {
    URL.revokeObjectURL(url)
  }
  thumbCache.clear()
}

export default api
