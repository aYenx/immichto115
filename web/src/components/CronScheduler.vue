<template>
  <div class="cron-scheduler">
    <!-- Frequency Tabs -->
    <div class="field-group">
      <span class="field-label">备份频率</span>
      <div class="tab-bar">
        <button
          v-for="f in frequencies"
          :key="f.value"
          :class="['tab-btn', frequency === f.value ? 'active' : '']"
          @click="frequency = f.value"
        >
          {{ f.label }}
        </button>
      </div>
    </div>

    <!-- Time Picker (for daily / weekly) -->
    <div v-if="frequency === 'daily' || frequency === 'weekly'" class="field-group">
      <span class="field-label">执行时间</span>
      <div class="time-picker">
        <div class="time-select-wrapper">
          <select class="time-select" v-model="hour">
            <option v-for="h in hours" :key="h" :value="h">{{ padZero(h) }}</option>
          </select>
          <span class="time-unit">时</span>
        </div>
        <span class="time-colon">:</span>
        <div class="time-select-wrapper">
          <select class="time-select" v-model="minute">
            <option v-for="m in minutes" :key="m" :value="m">{{ padZero(m) }}</option>
          </select>
          <span class="time-unit">分</span>
        </div>
      </div>
    </div>

    <!-- Day of Week Picker (for weekly) -->
    <div v-if="frequency === 'weekly'" class="field-group">
      <span class="field-label">执行日</span>
      <div class="weekday-bar">
        <button
          v-for="d in weekdays"
          :key="d.value"
          :class="['weekday-btn', dayOfWeek === d.value ? 'active' : '']"
          @click="dayOfWeek = d.value"
        >
          {{ d.label }}
        </button>
      </div>
    </div>

    <!-- Interval Picker (for interval) -->
    <div v-if="frequency === 'interval'" class="field-group">
      <span class="field-label">间隔时间</span>
      <div class="interval-picker">
        <span class="interval-prefix">每</span>
        <select class="time-select" v-model="intervalHours">
          <option v-for="i in intervalOptions" :key="i" :value="i">{{ i }}</option>
        </select>
        <span class="interval-suffix">小时执行一次</span>
      </div>
    </div>

    <!-- Custom Expression (for custom) -->
    <div v-if="frequency === 'custom'" class="field-group">
      <span class="field-label">Cron 表达式</span>
      <input
        class="cron-input"
        type="text"
        v-model="customExpression"
        placeholder="例如: 0 3 * * *"
      />
      <span class="field-hint">格式: 分 时 日 月 星期 (5段标准 cron)</span>
    </div>

    <!-- Preview -->
    <div class="preview-box">
      <div class="preview-row">
        <LucideClock :size="16" class="preview-icon" />
        <span class="preview-label">Cron 表达式：</span>
        <code class="preview-value">{{ generatedExpression }}</code>
      </div>
      <div class="preview-row">
        <LucideCalendarClock :size="16" class="preview-icon" />
        <span class="preview-label">执行说明：</span>
        <span class="preview-desc">{{ humanDescription }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { LucideClock, LucideCalendarClock } from 'lucide-vue-next'

const props = defineProps<{
  modelValue: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
}>()

type FrequencyType = 'daily' | 'weekly' | 'interval' | 'custom'

const frequencies = [
  { value: 'daily' as FrequencyType, label: '每天' },
  { value: 'weekly' as FrequencyType, label: '每周' },
  { value: 'interval' as FrequencyType, label: '每隔N小时' },
  { value: 'custom' as FrequencyType, label: '自定义' },
]

const weekdays = [
  { value: 1, label: '一' },
  { value: 2, label: '二' },
  { value: 3, label: '三' },
  { value: 4, label: '四' },
  { value: 5, label: '五' },
  { value: 6, label: '六' },
  { value: 0, label: '日' },
]

const weekdayNames: Record<number, string> = {
  0: '星期日', 1: '星期一', 2: '星期二', 3: '星期三',
  4: '星期四', 5: '星期五', 6: '星期六',
}

const hours = Array.from({ length: 24 }, (_, i) => i)
const minutes = Array.from({ length: 12 }, (_, i) => i * 5) // 0, 5, 10, ..., 55
const intervalOptions = [1, 2, 3, 4, 6, 8, 12]

const frequency = ref<FrequencyType>('daily')
const hour = ref(3)
const minute = ref(0)
const dayOfWeek = ref(0) // Sunday
const intervalHours = ref(6)
const customExpression = ref('0 3 * * *')

const padZero = (n: number) => String(n).padStart(2, '0')

// Parse incoming cron expression to set UI state
const parseCronExpression = (expr: string) => {
  if (!expr) return
  const parts = expr.trim().split(/\s+/)
  if (parts.length !== 5) {
    frequency.value = 'custom'
    customExpression.value = expr
    return
  }

  const minPart = parts[0]!
  const hourPart = parts[1]!
  const dayPart = parts[2]!
  const monPart = parts[3]!
  const dowPart = parts[4]!

  // Check for interval pattern: "0 */N * * *"
  if (hourPart.startsWith('*/') && dayPart === '*' && monPart === '*' && dowPart === '*') {
    const interval = parseInt(hourPart.substring(2))
    if (intervalOptions.includes(interval)) {
      frequency.value = 'interval'
      intervalHours.value = interval
      return
    }
  }

  // Check for weekly pattern: "M H * * D" where D is a single digit
  if (dayPart === '*' && monPart === '*' && /^\d$/.test(dowPart)) {
    const m = parseInt(minPart)
    const h = parseInt(hourPart)
    const d = parseInt(dowPart)
    if (!isNaN(m) && !isNaN(h) && d >= 0 && d <= 6) {
      frequency.value = 'weekly'
      minute.value = m
      hour.value = h
      dayOfWeek.value = d
      return
    }
  }

  // Check for daily pattern: "M H * * *"
  if (dayPart === '*' && monPart === '*' && dowPart === '*') {
    const m = parseInt(minPart)
    const h = parseInt(hourPart)
    if (!isNaN(m) && !isNaN(h)) {
      frequency.value = 'daily'
      minute.value = m
      hour.value = h
      return
    }
  }

  // Fallback to custom
  frequency.value = 'custom'
  customExpression.value = expr
}

// Generate cron expression from UI state
const generatedExpression = computed(() => {
  switch (frequency.value) {
    case 'daily':
      return `${minute.value} ${hour.value} * * *`
    case 'weekly':
      return `${minute.value} ${hour.value} * * ${dayOfWeek.value}`
    case 'interval':
      return `0 */${intervalHours.value} * * *`
    case 'custom':
      return customExpression.value
    default:
      return '0 3 * * *'
  }
})

// Human-readable description
const humanDescription = computed(() => {
  switch (frequency.value) {
    case 'daily':
      return `每天 ${padZero(hour.value)}:${padZero(minute.value)} 执行备份`
    case 'weekly':
      return `每${weekdayNames[dayOfWeek.value]} ${padZero(hour.value)}:${padZero(minute.value)} 执行备份`
    case 'interval':
      return `每 ${intervalHours.value} 小时执行一次备份`
    case 'custom':
      return '使用自定义 Cron 表达式'
    default:
      return ''
  }
})

// Emit changes
watch(generatedExpression, (val) => {
  emit('update:modelValue', val)
})

// Parse initial value
onMounted(() => {
  parseCronExpression(props.modelValue)
})

// Watch for external changes
watch(() => props.modelValue, (val) => {
  if (val !== generatedExpression.value) {
    parseCronExpression(val)
  }
})
</script>

<style scoped>
.cron-scheduler {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.field-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.field-label {
  color: var(--text-primary);
  font-size: 14px;
  font-weight: 600;
}

.field-hint {
  color: var(--text-tertiary);
  font-size: 12px;
}

/* Frequency Tabs */
.tab-bar {
  display: flex;
  gap: 0;
  background-color: var(--bg-primary);
  border-radius: 10px;
  padding: 3px;
  border: 1px solid var(--border-strong);
}

.tab-btn {
  flex: 1;
  height: 36px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-secondary);
  background: transparent;
  transition: all 0.2s ease;
  cursor: pointer;
}

.tab-btn:hover {
  color: var(--text-primary);
}

.tab-btn.active {
  background-color: var(--text-primary);
  color: var(--text-inverted);
  font-weight: 600;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

/* Time Picker */
.time-picker {
  display: flex;
  align-items: center;
  gap: 8px;
}

.time-select-wrapper {
  display: flex;
  align-items: center;
  gap: 6px;
}

.time-select {
  height: 44px;
  width: 80px;
  border-radius: 10px;
  border: 1px solid var(--border-strong);
  background-color: transparent;
  padding: 0 12px;
  color: var(--text-primary);
  font-size: 18px;
  font-weight: 600;
  font-family: var(--font-primary);
  appearance: none;
  -webkit-appearance: none;
  cursor: pointer;
  text-align: center;
}

.time-select:focus {
  outline: 2px solid var(--text-primary);
  outline-offset: -1px;
}

.time-unit {
  color: var(--text-secondary);
  font-size: 14px;
}

.time-colon {
  color: var(--text-primary);
  font-size: 24px;
  font-weight: 700;
  margin: 0 2px;
}

/* Weekday Bar */
.weekday-bar {
  display: flex;
  gap: 6px;
}

.weekday-btn {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  font-size: 14px;
  font-weight: 600;
  color: var(--text-secondary);
  border: 1px solid var(--border-strong);
  background: transparent;
  transition: all 0.2s ease;
  cursor: pointer;
}

.weekday-btn:hover {
  background-color: var(--border-subtle);
  color: var(--text-primary);
}

.weekday-btn.active {
  background-color: var(--text-primary);
  color: var(--text-inverted);
  border-color: var(--text-primary);
}

/* Interval Picker */
.interval-picker {
  display: flex;
  align-items: center;
  gap: 10px;
}

.interval-prefix,
.interval-suffix {
  color: var(--text-secondary);
  font-size: 15px;
}

/* Custom Expression Input */
.cron-input {
  height: 48px;
  border-radius: 12px;
  border: 1px solid var(--border-strong);
  background-color: transparent;
  padding: 0 16px;
  color: var(--text-primary);
  font-size: 16px;
  font-family: 'Consolas', 'Monaco', monospace;
  letter-spacing: 2px;
}

.cron-input::placeholder {
  color: var(--text-tertiary);
  letter-spacing: 1px;
}

.cron-input:focus {
  outline: 2px solid var(--text-primary);
  outline-offset: -1px;
}

/* Preview Box */
.preview-box {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 16px;
  background-color: var(--bg-primary);
  border-radius: 12px;
  border: 1px solid var(--border-strong);
}

.preview-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.preview-icon {
  color: var(--text-tertiary);
  flex-shrink: 0;
}

.preview-label {
  color: var(--text-secondary);
  font-size: 13px;
  white-space: nowrap;
}

.preview-value {
  color: var(--text-primary);
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 13px;
  background-color: var(--border-subtle);
  padding: 2px 8px;
  border-radius: 4px;
}

.preview-desc {
  color: var(--text-primary);
  font-size: 13px;
  font-weight: 500;
}
</style>
