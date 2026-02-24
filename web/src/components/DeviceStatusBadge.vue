<template>
  <a-tag :color="deviceStatusMap[status]?.color || '#999'">
    {{ deviceStatusMap[status]?.text || '未知' }}
  </a-tag>
</template>

<script setup lang="ts">
import { deviceStatusMap } from '@/utils/format'

defineProps<{ status: number }>()
</script>
