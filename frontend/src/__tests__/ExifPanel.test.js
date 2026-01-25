import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ExifPanel from '../components/ExifPanel.vue'

describe('ExifPanel', () => {
  const fullExifData = {
    focal_length: '50mm',
    aperture: 'f/1.8',
    shutter_speed: '1/125',
    iso: '100',
    date_time: '2024-01-15 14:30:00',
    width: 6000,
    height: 4000
  }

  it('shows loading state when loading is true', () => {
    const wrapper = mount(ExifPanel, {
      props: {
        loading: true
      }
    })
    expect(wrapper.find('.animate-spin').exists()).toBe(true)
  })

  it('shows no data message when exif is null', () => {
    const wrapper = mount(ExifPanel, {
      props: {
        exif: null,
        loading: false
      }
    })
    expect(wrapper.text()).toContain('No EXIF data available')
  })

  it('shows no data message when exif is empty', () => {
    const wrapper = mount(ExifPanel, {
      props: {
        exif: {},
        loading: false
      }
    })
    expect(wrapper.text()).toContain('No EXIF data available')
  })

  it('displays focal length when present', () => {
    const wrapper = mount(ExifPanel, {
      props: {
        exif: { focal_length: '50mm' }
      }
    })
    expect(wrapper.text()).toContain('50mm')
  })

  it('displays aperture when present', () => {
    const wrapper = mount(ExifPanel, {
      props: {
        exif: { aperture: 'f/2.8' }
      }
    })
    expect(wrapper.text()).toContain('f/2.8')
  })

  it('displays shutter speed when present', () => {
    const wrapper = mount(ExifPanel, {
      props: {
        exif: { shutter_speed: '1/250' }
      }
    })
    expect(wrapper.text()).toContain('1/250')
  })

  it('displays ISO when present', () => {
    const wrapper = mount(ExifPanel, {
      props: {
        exif: { iso: '400' }
      }
    })
    expect(wrapper.text()).toContain('400')
  })

  it('displays date/time when present', () => {
    const wrapper = mount(ExifPanel, {
      props: {
        exif: { date_time: '2024-01-15' }
      }
    })
    expect(wrapper.text()).toContain('2024-01-15')
  })

  it('displays dimensions when both width and height present', () => {
    const wrapper = mount(ExifPanel, {
      props: {
        exif: { width: 1920, height: 1080 }
      }
    })
    expect(wrapper.text()).toContain('1920 x 1080')
  })

  it('does not display dimensions when only width present', () => {
    const wrapper = mount(ExifPanel, {
      props: {
        exif: { width: 1920 }
      }
    })
    // Only width without height means dimensions won't show
    // But hasExifData will still return true if width exists
    expect(wrapper.text()).not.toContain('1920 x')
  })

  it('displays all EXIF data when provided', () => {
    const wrapper = mount(ExifPanel, {
      props: {
        exif: fullExifData
      }
    })
    expect(wrapper.text()).toContain('50mm')
    expect(wrapper.text()).toContain('f/1.8')
    expect(wrapper.text()).toContain('1/125')
    expect(wrapper.text()).toContain('100')
    expect(wrapper.text()).toContain('2024-01-15 14:30:00')
    expect(wrapper.text()).toContain('6000 x 4000')
  })

  it('applies compact styles when compact is true', () => {
    const wrapper = mount(ExifPanel, {
      props: {
        exif: { focal_length: '50mm' },
        compact: true
      }
    })
    expect(wrapper.find('.text-xs').exists()).toBe(true)
  })

  it('applies normal styles when compact is false', () => {
    const wrapper = mount(ExifPanel, {
      props: {
        exif: { focal_length: '50mm' },
        compact: false
      }
    })
    expect(wrapper.find('.text-sm').exists()).toBe(true)
  })

  it('hasExifData returns false for partial data without key fields', () => {
    const wrapper = mount(ExifPanel, {
      props: {
        exif: { unknown_field: 'value' }
      }
    })
    expect(wrapper.text()).toContain('No EXIF data available')
  })
})
