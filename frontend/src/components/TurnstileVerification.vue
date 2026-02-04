<template>
  <div class="verification-overlay" v-if="showVerification">
    <div class="verification-card">
      <div class="verification-header">
        <h2>安全验证</h2>
        <p>请完成验证以继续访问</p>
      </div>

      <div class="verification-body">
        <div v-if="loading" class="loading">
          <div class="spinner"></div>
          <p>正在加载验证...</p>
        </div>

        <div v-else-if="error" class="error-message">
          <p>{{ error }}</p>
          <button @click="retry" class="retry-button">重试</button>
        </div>

        <div v-else>
          <div :id="widgetId" class="turnstile-widget"></div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'TurnstileVerification',
  props: {
    siteKey: {
      type: String,
      required: true
    }
  },
  data() {
    return {
      showVerification: false,
      loading: true,
      error: null,
      widgetId: 'turnstile-widget-' + Math.random().toString(36).substr(2, 9),
      turnstileLoaded: false
    }
  },
  methods: {
    show() {
      this.showVerification = true
      this.loadTurnstile()
    },

    hide() {
      this.showVerification = false
    },

    loadTurnstile() {
      // Check if Turnstile script is already loaded
      if (window.turnstile) {
        this.renderWidget()
        return
      }

      // Load Turnstile script
      const script = document.createElement('script')
      script.src = 'https://challenges.cloudflare.com/turnstile/v0/api.js'
      script.async = true
      script.defer = true
      script.onload = () => {
        this.turnstileLoaded = true
        this.renderWidget()
      }
      script.onerror = () => {
        this.loading = false
        this.error = '验证组件加载失败，请刷新页面重试'
      }
      document.head.appendChild(script)
    },

    renderWidget() {
      this.loading = false

      // Wait for DOM to be ready
      this.$nextTick(() => {
        try {
          window.turnstile.render('#' + this.widgetId, {
            sitekey: this.siteKey,
            callback: (token) => {
              this.verifyToken(token)
            },
            'error-callback': () => {
              this.error = '验证失败，请重试'
            },
            theme: 'light',
            size: 'normal'
          })
        } catch (err) {
          console.error('Turnstile render error:', err)
          this.error = '验证组件渲染失败'
        }
      })
    },

    async verifyToken(token) {
      try {
        const response = await fetch('/api/verify', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          },
          body: JSON.stringify({ token })
        })

        const data = await response.json()

        if (data.success) {
          this.$emit('verified')
          this.hide()
        } else {
          this.error = data.message || '验证失败'
        }
      } catch (err) {
        console.error('Verification error:', err)
        this.error = '验证请求失败，请重试'
      }
    },

    retry() {
      this.error = null
      this.loading = true
      this.renderWidget()
    }
  }
}
</script>

<style scoped>
.verification-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
  backdrop-filter: blur(4px);
}

.verification-card {
  background: white;
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
  max-width: 400px;
  width: 90%;
  padding: 32px;
  animation: slideIn 0.3s ease-out;
}

@keyframes slideIn {
  from {
    opacity: 0;
    transform: translateY(-20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.verification-header {
  text-align: center;
  margin-bottom: 24px;
}

.verification-header h2 {
  margin: 0 0 8px 0;
  font-size: 24px;
  color: #333;
}

.verification-header p {
  margin: 0;
  color: #666;
  font-size: 14px;
}

.verification-body {
  min-height: 100px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.loading {
  text-align: center;
}

.spinner {
  width: 40px;
  height: 40px;
  border: 4px solid #f3f3f3;
  border-top: 4px solid #3498db;
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin: 0 auto 16px;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.loading p {
  color: #666;
  font-size: 14px;
}

.error-message {
  text-align: center;
}

.error-message p {
  color: #e74c3c;
  margin-bottom: 16px;
}

.retry-button {
  background: #3498db;
  color: white;
  border: none;
  padding: 10px 24px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  transition: background 0.2s;
}

.retry-button:hover {
  background: #2980b9;
}

.turnstile-widget {
  display: flex;
  justify-content: center;
  min-height: 65px;
}
</style>
