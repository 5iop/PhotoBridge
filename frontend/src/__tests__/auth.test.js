import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

// Mock localStorage
const localStorageMock = {
  store: {},
  getItem: vi.fn((key) => localStorageMock.store[key] || null),
  setItem: vi.fn((key, value) => {
    localStorageMock.store[key] = value
  }),
  removeItem: vi.fn((key) => {
    delete localStorageMock.store[key]
  }),
  clear: vi.fn(() => {
    localStorageMock.store = {}
  })
}

vi.stubGlobal('localStorage', localStorageMock)

// Mock the API module
vi.mock('../api', () => ({
  login: vi.fn()
}))

describe('Auth Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorageMock.clear()
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  it('initializes with null token when localStorage is empty', async () => {
    const { useAuthStore } = await import('../stores/auth')
    const store = useAuthStore()
    expect(store.token).toBeNull()
    expect(store.isAuthenticated).toBe(false)
  })

  it('initializes with token from localStorage', async () => {
    localStorageMock.store['token'] = 'existing-token'

    // Need to re-import to get fresh store
    vi.resetModules()
    const { useAuthStore } = await import('../stores/auth')
    const store = useAuthStore()

    expect(store.token).toBe('existing-token')
    expect(store.isAuthenticated).toBe(true)
  })

  it('login stores token correctly', async () => {
    const { login: mockLogin } = await import('../api')
    mockLogin.mockResolvedValue({ data: { token: 'new-token' } })

    const { useAuthStore } = await import('../stores/auth')
    const store = useAuthStore()

    await store.login('admin', 'password')

    expect(mockLogin).toHaveBeenCalledWith('admin', 'password')
    expect(store.token).toBe('new-token')
    expect(localStorageMock.setItem).toHaveBeenCalledWith('token', 'new-token')
    expect(store.isAuthenticated).toBe(true)
  })

  it('logout clears token', async () => {
    localStorageMock.store['token'] = 'existing-token'

    const { useAuthStore } = await import('../stores/auth')
    const store = useAuthStore()
    store.token = 'existing-token'

    store.logout()

    expect(store.token).toBeNull()
    expect(localStorageMock.removeItem).toHaveBeenCalledWith('token')
    expect(store.isAuthenticated).toBe(false)
  })

  it('isAuthenticated returns true when token exists', async () => {
    const { useAuthStore } = await import('../stores/auth')
    const store = useAuthStore()
    store.token = 'valid-token'
    expect(store.isAuthenticated).toBe(true)
  })

  it('isAuthenticated returns false when token is null', async () => {
    const { useAuthStore } = await import('../stores/auth')
    const store = useAuthStore()
    store.token = null
    expect(store.isAuthenticated).toBe(false)
  })

  it('isAuthenticated returns false when token is empty string', async () => {
    const { useAuthStore } = await import('../stores/auth')
    const store = useAuthStore()
    store.token = ''
    expect(store.isAuthenticated).toBe(false)
  })
})
