<template>
  <div class="page-content mb-5">
    <el-row :gutter="20">
      <!-- 内存 信息 -->
      <el-col :span="24" class="mb-4">
        <el-card class="art-table-card" shadow="never">
          <template #header>
            <span class="text-lg font-medium">内存信息</span>
          </template>
          <div class="flex justify-between">
            <div class="flex-1">
              <el-descriptions class="memory-desc" :column="1" border>
                <el-descriptions-item label="总内存">
                  {{ formatBytes(serverInfo.memory.total) }}
                </el-descriptions-item>
                <el-descriptions-item label="已使用内存">
                  {{ formatBytes(serverInfo.memory.used) }}
                </el-descriptions-item>
                <el-descriptions-item label="空闲内存">
                  {{ formatBytes(serverInfo.memory.free) }}
                </el-descriptions-item>
                <el-descriptions-item label="使用率">
                  {{ memoryUsage.toFixed(2) }}%
                </el-descriptions-item>
              </el-descriptions>
            </div>
            <div class="w-80 p-4 text-center">
              <div class="pb-3.5">
                <span class="text-base font-medium">内存使用率</span>
              </div>
              <el-progress type="dashboard" :percentage="memoryUsage" />
            </div>
          </div>
        </el-card>
      </el-col>

      <!-- CPU 与负载（真实指标：Windows 取 GetSystemTimes，Linux 取 /proc/stat 与 /proc/loadavg） -->
      <el-col :span="24" class="mb-4">
        <el-card class="art-table-card" shadow="never">
          <template #header>
            <div class="card-header">
              <span class="text-lg font-medium">CPU 与负载</span>
              <span class="text-g-600 text-sm">
                每 5 秒自动刷新{{ lastUpdated ? `，最后更新 ${lastUpdated}` : '' }}
              </span>
            </div>
          </template>
          <div class="flex justify-between">
            <div class="flex-1">
              <el-descriptions class="memory-desc" :column="1" border>
                <el-descriptions-item label="CPU 核心数">
                  {{ serverInfo.cpu.cores }}
                </el-descriptions-item>
                <el-descriptions-item label="CPU 使用率">
                  {{ cpuUsageText }}
                </el-descriptions-item>
                <el-descriptions-item label="已用 / 空闲">
                  {{ serverInfo.cpu.used }} / {{ serverInfo.cpu.idle }}
                </el-descriptions-item>
                <el-descriptions-item label="系统负载（1 / 5 / 15 分钟）">
                  <template v-if="serverInfo.load.available">
                    {{ serverInfo.load.one.toFixed(2) }} /
                    {{ serverInfo.load.five.toFixed(2) }} /
                    {{ serverInfo.load.fifteen.toFixed(2) }}
                  </template>
                  <template v-else>
                    <span>该平台不提供</span>
                    <span class="text-g-600 text-xs ml-2">
                      （Windows 无 load average 指标，参考 CPU 使用率）
                    </span>
                  </template>
                </el-descriptions-item>
              </el-descriptions>
            </div>
            <div class="w-80 p-4 text-center">
              <div class="pb-3.5">
                <span class="text-base font-medium">CPU 使用率</span>
              </div>
              <el-progress type="dashboard" :percentage="cpuUsage" />
            </div>
          </div>
        </el-card>
      </el-col>

      <!-- Go 运行时信息 -->
      <el-col :span="24" class="mb-4">
        <el-card class="art-table-card" shadow="never">
          <template #header>
            <span class="text-lg font-medium">Go 运行时信息</span>
          </template>
          <div class="py-2">
            <el-descriptions :column="2" border>
              <el-descriptions-item label="Go 版本">
                {{ serverInfo.server.go }}
              </el-descriptions-item>
              <el-descriptions-item label="操作系统">
                {{ serverInfo.server.os }}
              </el-descriptions-item>
              <el-descriptions-item label="系统架构">
                {{ serverInfo.server.arch }}
              </el-descriptions-item>
              <el-descriptions-item label="主机名">
                {{ serverInfo.server.hostname }}
              </el-descriptions-item>
              <el-descriptions-item label="CPU 核心数">
                {{ serverInfo.cpu.cores }}
              </el-descriptions-item>
              <el-descriptions-item label="Goroutine 数量">
                {{ serverInfo.goroutines }}
              </el-descriptions-item>
              <el-descriptions-item label="缓存键数量">
                {{ serverInfo.cache_keys }}
              </el-descriptions-item>
              <el-descriptions-item label="启动时间">
                {{ serverInfo.server.start_time }}
              </el-descriptions-item>
              <el-descriptions-item label="运行时长">
                {{ formatUptime(serverInfo.server.uptime) }}
              </el-descriptions-item>
              <el-descriptions-item label="当前时间">
                {{ serverInfo.server.now_time }}
              </el-descriptions-item>
            </el-descriptions>
          </div>
        </el-card>
      </el-col>

      <!-- 磁盘 信息 -->
      <el-col :span="24" class="mb-4">
        <el-card class="art-table-card" shadow="never">
          <template #header>
            <div class="card-header">
              <span><i class="el-icon-disk"></i> 磁盘监控</span>
            </div>
          </template>

          <el-table :data="diskRows" style="width: 100%">
            <el-table-column prop="path" label="路径" />
            <el-table-column prop="total" label="总大小" />
            <el-table-column prop="used" label="已用空间" />
            <el-table-column prop="free" label="可用空间" />
            <el-table-column prop="usage" label="使用率">
              <template #default="{ row }">
                <el-progress :percentage="row.usage" :stroke-width="12" :show-text="true" />
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
  import api from '@/api/safeguard/server'
  import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'

  /** 后端 GET /core/server/monitor 返回结构 */
  interface MonitorCpu {
    cores: number
    used: string
    idle: string
    usage: number
    /** 沿用旧契约的键名，值为 Go 版本 */
    go_version: string
  }

  interface MonitorMemory {
    total: number
    used: number
    free: number
    usage: number
    /** 沿用旧契约的键名，值为 Go 版本 */
    php_version: string
  }

  interface MonitorDisk {
    total: number
    used: number
    free: number
    usage: number
    path: string
  }

  interface MonitorLoad {
    one: number
    five: number
    fifteen: number
    /** 平台是否提供 load average（Windows 为 false） */
    available: boolean
  }

  interface MonitorServer {
    hostname: string
    os: string
    arch: string
    go: string
    uptime: number
    start_time: string
    now_time: string
  }

  interface MonitorData {
    cpu: MonitorCpu
    memory: MonitorMemory
    disk: MonitorDisk
    load: MonitorLoad
    server: MonitorServer
    cache_keys: number
    goroutines: number
  }

  const loading = ref(false)
  /** 最后一次成功刷新的时间（HH:mm:ss） */
  const lastUpdated = ref('')
  /** 刷新间隔（毫秒）：CPU 使用率是两次采样之间的差值，间隔稳定数值才有意义 */
  const REFRESH_INTERVAL = 5000
  let timer: ReturnType<typeof setInterval> | undefined

  const serverInfo = reactive<MonitorData>({
    cpu: { cores: 0, used: '-', idle: '-', usage: 0, go_version: '' },
    memory: { total: 0, used: 0, free: 0, usage: 0, php_version: '' },
    disk: { total: 0, used: 0, free: 0, usage: 0, path: '' },
    load: { one: 0, five: 0, fifteen: 0, available: false },
    server: { hostname: '', os: '', arch: '', go: '', uptime: 0, start_time: '', now_time: '' },
    cache_keys: 0,
    goroutines: 0
  })

  /** 内存使用率（保留两位小数） */
  const memoryUsage = computed(() => Number(Number(serverInfo.memory.usage).toFixed(2)))

  /** CPU 使用率（保留两位小数，用于仪表盘） */
  const cpuUsage = computed(() => Number(Number(serverInfo.cpu.usage).toFixed(2)))

  /** CPU 使用率文本：后端不可采集时显示占位符 */
  const cpuUsageText = computed(() => (serverInfo.cpu.used === '-' ? '该平台不提供' : `${cpuUsage.value.toFixed(2)}%`))

  /** 磁盘信息（Go 后端返回的是单个对象，转为单行表格数据；采集不到时显示占位符） */
  const diskRows = computed(() => {
    const available = Number(serverInfo.disk.total) > 0
    return [
      {
        path: available ? serverInfo.disk.path : '该平台不提供',
        total: available ? formatBytes(serverInfo.disk.total) : '—',
        used: available ? formatBytes(serverInfo.disk.used) : '—',
        free: available ? formatBytes(serverInfo.disk.free) : '—',
        usage: available ? Number(Number(serverInfo.disk.usage).toFixed(2)) : 0
      }
    ]
  })

  /**
   * 字节数格式化
   * @param bytes 字节数
   */
  const formatBytes = (bytes?: number): string => {
    const size = Number(bytes)
    if (!size || size <= 0) return '0 B'
    const units = ['B', 'KB', 'MB', 'GB', 'TB']
    const index = Math.min(Math.floor(Math.log(size) / Math.log(1024)), units.length - 1)
    return `${(size / 1024 ** index).toFixed(2)} ${units[index]}`
  }

  /**
   * 运行时长格式化
   * @param seconds 运行秒数
   */
  const formatUptime = (seconds?: number): string => {
    const total = Math.max(0, Math.floor(Number(seconds) || 0))
    const days = Math.floor(total / 86400)
    const hours = Math.floor((total % 86400) / 3600)
    const minutes = Math.floor((total % 3600) / 60)
    const secs = total % 60
    return days > 0 ? `${days}天${hours}小时${minutes}分` : `${hours}小时${minutes}分${secs}秒`
  }

  /**
   * 更新服务器信息
   */
  const updateServer = async () => {
    loading.value = true
    try {
      const data = await api.monitor({})
      serverInfo.cpu = { ...serverInfo.cpu, ...data?.cpu }
      serverInfo.memory = { ...serverInfo.memory, ...data?.memory }
      serverInfo.disk = { ...serverInfo.disk, ...data?.disk }
      serverInfo.load = { ...serverInfo.load, ...data?.load }
      serverInfo.server = { ...serverInfo.server, ...data?.server }
      serverInfo.cache_keys = data?.cache_keys ?? 0
      serverInfo.goroutines = data?.goroutines ?? 0

      const now = new Date()
      const pad = (n: number): string => String(n).padStart(2, '0')
      lastUpdated.value = `${pad(now.getHours())}:${pad(now.getMinutes())}:${pad(now.getSeconds())}`
    } finally {
      loading.value = false
    }
  }

  onMounted(() => {
    updateServer()
    // 定时刷新：CPU 使用率取两次采样差值，间隔固定才能反映「最近 5 秒」的真实占用
    timer = setInterval(updateServer, REFRESH_INTERVAL)
  })

  onUnmounted(() => {
    if (timer) clearInterval(timer)
  })
</script>

<style lang="scss" scoped>
  .memory-desc :deep(.el-descriptions__label) {
    width: 200px;
  }
  .memory-desc :deep(.el-descriptions__content) {
    width: 400px;
  }
</style>
