<script setup>
defineProps({
  exif: {
    type: Object,
    default: null
  },
  loading: {
    type: Boolean,
    default: false
  },
  compact: {
    type: Boolean,
    default: false
  }
})

function hasExifData(exif) {
  if (!exif) return false
  return exif.focal_length || exif.aperture || exif.shutter_speed ||
         exif.iso || exif.date_time || exif.width || exif.height
}
</script>

<template>
  <div>
    <!-- Loading state -->
    <div v-if="loading" class="flex justify-center py-4">
      <svg class="w-5 h-5 text-primary-500 animate-spin" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
      </svg>
    </div>

    <!-- No EXIF data -->
    <div v-else-if="!hasExifData(exif)" class="text-gray-500 text-sm py-2">
      No EXIF data available
    </div>

    <!-- EXIF data display -->
    <div v-else :class="compact ? 'space-y-1' : 'space-y-2'">
      <!-- Focal Length -->
      <div v-if="exif.focal_length" class="flex items-center gap-2" :class="compact ? 'text-xs' : 'text-sm'">
        <svg class="w-4 h-4 text-gray-500 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
        </svg>
        <span class="text-gray-400">{{ exif.focal_length }}</span>
      </div>

      <!-- Aperture -->
      <div v-if="exif.aperture" class="flex items-center gap-2" :class="compact ? 'text-xs' : 'text-sm'">
        <svg class="w-4 h-4 text-gray-500 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <circle cx="12" cy="12" r="10" stroke-width="2" />
          <circle cx="12" cy="12" r="3" stroke-width="2" />
        </svg>
        <span class="text-gray-400">{{ exif.aperture }}</span>
      </div>

      <!-- Shutter Speed -->
      <div v-if="exif.shutter_speed" class="flex items-center gap-2" :class="compact ? 'text-xs' : 'text-sm'">
        <svg class="w-4 h-4 text-gray-500 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <span class="text-gray-400">{{ exif.shutter_speed }}</span>
      </div>

      <!-- ISO -->
      <div v-if="exif.iso" class="flex items-center gap-2" :class="compact ? 'text-xs' : 'text-sm'">
        <span class="w-4 h-4 text-gray-500 flex-shrink-0 text-center text-xs font-bold">ISO</span>
        <span class="text-gray-400">{{ exif.iso }}</span>
      </div>

      <!-- Date/Time -->
      <div v-if="exif.date_time" class="flex items-center gap-2" :class="compact ? 'text-xs' : 'text-sm'">
        <svg class="w-4 h-4 text-gray-500 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
        </svg>
        <span class="text-gray-400">{{ exif.date_time }}</span>
      </div>

      <!-- Dimensions -->
      <div v-if="exif.width && exif.height" class="flex items-center gap-2" :class="compact ? 'text-xs' : 'text-sm'">
        <svg class="w-4 h-4 text-gray-500 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" />
        </svg>
        <span class="text-gray-400">{{ exif.width }} x {{ exif.height }}</span>
      </div>
    </div>
  </div>
</template>
