import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { mount, config } from '@vue/test-utils'
import Modal from '../components/Modal.vue'

// Disable Teleport for testing
config.global.stubs = {
  Teleport: true
}

describe('Modal', () => {
  it('renders when show is true', () => {
    const wrapper = mount(Modal, {
      props: {
        show: true,
        title: 'Test Modal'
      }
    })
    expect(wrapper.text()).toContain('Test Modal')
  })

  it('does not render when show is false', () => {
    const wrapper = mount(Modal, {
      props: {
        show: false,
        title: 'Test Modal'
      }
    })
    expect(wrapper.text()).not.toContain('Test Modal')
  })

  it('emits close event when backdrop is clicked', async () => {
    const wrapper = mount(Modal, {
      props: {
        show: true,
        title: 'Test Modal'
      }
    })
    // Find the backdrop element and click it
    const backdrop = wrapper.find('.fixed')
    await backdrop.trigger('click')
    expect(wrapper.emitted('close')).toBeTruthy()
  })

  it('does not emit close when modal content is clicked', async () => {
    const wrapper = mount(Modal, {
      props: {
        show: true,
        title: 'Test Modal'
      }
    })
    // Find the card content and click it
    const card = wrapper.find('.card')
    await card.trigger('click')
    expect(wrapper.emitted('close')).toBeFalsy()
  })

  it('emits close when close button is clicked', async () => {
    const wrapper = mount(Modal, {
      props: {
        show: true,
        title: 'Test Modal'
      }
    })
    const closeBtn = wrapper.find('button')
    await closeBtn.trigger('click')
    expect(wrapper.emitted('close')).toBeTruthy()
  })

  it('renders slot content', () => {
    const wrapper = mount(Modal, {
      props: {
        show: true,
        title: 'Test Modal'
      },
      slots: {
        default: '<div class="slot-content">Slot Content</div>'
      }
    })
    expect(wrapper.find('.slot-content').exists()).toBe(true)
    expect(wrapper.text()).toContain('Slot Content')
  })

  it('applies custom maxWidth class', () => {
    const wrapper = mount(Modal, {
      props: {
        show: true,
        title: 'Test',
        maxWidth: 'max-w-lg'
      }
    })
    expect(wrapper.find('.max-w-lg').exists()).toBe(true)
  })

  it('hides title section when no title provided', () => {
    const wrapper = mount(Modal, {
      props: {
        show: true
      }
    })
    // When no title, the h3 element should not exist
    expect(wrapper.find('h3').exists()).toBe(false)
  })

  it('uses default maxWidth when not specified', () => {
    const wrapper = mount(Modal, {
      props: {
        show: true,
        title: 'Test'
      }
    })
    expect(wrapper.find('.max-w-md').exists()).toBe(true)
  })
})
