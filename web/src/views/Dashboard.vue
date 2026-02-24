<template>
  <div>
    <a-row :gutter="[24, 24]">
      <a-col :span="6">
        <a-statistic-card class="stat-card">
          <a-statistic title="用户总数" :value="dashboard?.total_users || 0">
            <template #prefix><UserOutlined /></template>
          </a-statistic>
        </a-statistic-card>
      </a-col>
      <a-col :span="6">
        <a-statistic-card class="stat-card">
          <a-statistic title="设备总数" :value="dashboard?.total_devices || 0">
            <template #prefix><LockOutlined /></template>
          </a-statistic>
        </a-statistic-card>
      </a-col>
      <a-col :span="6">
        <a-statistic-card class="stat-card">
          <a-statistic title="活跃会话" :value="dashboard?.active_sessions || 0">
            <template #prefix><TeamOutlined /></template>
          </a-statistic>
        </a-statistic-card>
      </a-col>
      <a-col :span="6">
        <a-statistic-card class="stat-card stat-card-alert" :class="{ 'has-alerts': (dashboard?.pending_alerts || 0) > 0 }">
          <a-statistic title="待处置告警" :value="dashboard?.pending_alerts || 0" :value-style="{ color: (dashboard?.pending_alerts || 0) > 0 ? '#ff4d4f' : undefined }">
            <template #prefix><AlertOutlined /></template>
          </a-statistic>
        </a-statistic-card>
      </a-col>
    </a-row>

    <a-row :gutter="24" style="margin-top: 24px">
      <a-col :span="12">
        <a-card title="设备状态分布">
          <div ref="chartRef" style="height: 300px"></div>
        </a-card>
      </a-col>
      <a-col :span="12">
        <a-card title="最新告警" :body-style="{ padding: 0 }">
          <a-table
            :columns="alertColumns"
            :data-source="dashboard?.recent_alerts || []"
            :pagination="false"
            size="small"
            row-key="id"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'alert_type'">
                {{ alertTypeMap[record.alert_type] || record.alert_type }}
              </template>
              <template v-if="column.key === 'severity'">
                <a-tag :color="alertSeverityMap[record.severity]?.color">
                  {{ alertSeverityMap[record.severity]?.text }}
                </a-tag>
              </template>
              <template v-if="column.key === 'created_at'">
                {{ formatTime(record.created_at) }}
              </template>
            </template>
          </a-table>
        </a-card>
      </a-col>
    </a-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import { UserOutlined, LockOutlined, TeamOutlined, AlertOutlined } from '@ant-design/icons-vue'
import { getDashboard } from '@/api/admin'
import { formatTime, alertSeverityMap, alertTypeMap } from '@/utils/format'
import type { DashboardData } from '@/types'

const dashboard = ref<DashboardData | null>(null)
const chartRef = ref<HTMLElement>()

const alertColumns = [
  { title: '类型', key: 'alert_type', dataIndex: 'alert_type' },
  { title: '设备', dataIndex: 'device_id', key: 'device_id' },
  { title: '级别', key: 'severity', dataIndex: 'severity', width: 80 },
  { title: '时间', key: 'created_at', dataIndex: 'created_at', width: 170 },
]

onMounted(async () => {
  try {
    dashboard.value = await getDashboard()
    await nextTick()
    renderChart()
  } catch {
    // handled by interceptor
  }
})

async function renderChart() {
  if (!chartRef.value || !dashboard.value) return

  const echarts = await import('echarts')
  const chart = echarts.init(chartRef.value)

  const statusData = dashboard.value.devices_by_status
  const data = [
    { value: statusData.normal || 0, name: '正常', itemStyle: { color: '#52c41a' } },
    { value: statusData.disabled || 0, name: '已禁用', itemStyle: { color: '#d9d9d9' } },
    { value: statusData.alert_locked || 0, name: '告警锁定', itemStyle: { color: '#ff4d4f' } },
  ].filter(d => d.value > 0)

  chart.setOption({
    tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
    series: [{
      type: 'pie',
      radius: ['40%', '70%'],
      avoidLabelOverlap: false,
      label: { show: true, formatter: '{b}\n{c}台' },
      data,
    }],
  })

  window.addEventListener('resize', () => chart.resize())
}
</script>

<style scoped>
.stat-card {
  padding: 20px 24px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.stat-card-alert.has-alerts {
  border-left: 3px solid #ff4d4f;
}
</style>
