// Turnstile verification utilities

let verificationComponent = null

// Register the verification component instance
export function registerVerificationComponent(component) {
  verificationComponent = component
}

// Handle API error responses that require verification
export function handleVerificationRequired(error) {
  if (error.response && error.response.status === 403) {
    const data = error.response.data
    if (data.error === 'verification_required' && data.turnstile_key) {
      // Show verification dialog
      if (verificationComponent) {
        verificationComponent.show()
        return new Promise((resolve, reject) => {
          // Wait for verification to complete
          const handler = () => {
            verificationComponent.$off('verified', handler)
            resolve()
          }
          verificationComponent.$on('verified', handler)
        })
      }
    }
  }
  return Promise.reject(error)
}

// Create axios interceptor
export function setupAxiosInterceptor(axios) {
  axios.interceptors.response.use(
    response => response,
    error => handleVerificationRequired(error)
  )
}
