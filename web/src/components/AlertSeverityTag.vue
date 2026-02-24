<template>
  <a-tag :color="alertSeverityMap[severity]?.color || '#999'">
    {{ alertSeverityMap[severity]?.text || '未知' }}
  </a-tag>
</template>

<script setup lang="ts">
import { alertSeverityMap } from '@/utils/format'

defineProps<{ severity: number }>()
</script>
