<template>
  <n-date-picker
    ref="pickerRef"
    :value="value"
    type="datetime"
    :clearable="clearable"
    :size="size"
    :placeholder="placeholder"
    :disabled="disabled"
    :show="show"
    class="w-full"
    @update:value="onUpdateValue"
    @update:show="onUpdateShow"
  >
    <template #footer>
      <div class="time-preset-shortcuts" @mousedown.prevent>
        <n-button
          v-for="preset in timePresets"
          :key="preset.label"
          size="tiny"
          secondary
          block
          :disabled="disabled"
          @click="applyPresetTime(preset)"
        >
          {{ preset.label }}
        </n-button>
      </div>
    </template>
  </n-date-picker>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { NButton, NDatePicker } from 'naive-ui'

type DatePickerSize = 'small' | 'medium' | 'large'

interface TimePreset {
  label: string
  hour: number
  minute: number
}

type DatePickerValue = number | [number, number] | null
type DatePickerInst = InstanceType<typeof NDatePicker> & {
  pendingValue?: DatePickerValue
  handlePanelUpdateValue?: (value: number | null, doUpdate: boolean) => void
  handlePanelConfirm?: () => void
  handlePanelClose?: (disableUpdateOnClose?: boolean) => void
}

const props = withDefaults(defineProps<{
  value: number | null
  clearable?: boolean
  size?: DatePickerSize
  placeholder?: string
  disabled?: boolean
  show?: boolean
}>(), {
  clearable: true,
  size: 'small',
  placeholder: '选择时间',
  disabled: false
})

const emit = defineEmits<{
  'update:value': [value: number | null]
  'update:show': [show: boolean]
}>()

const pickerRef = ref<DatePickerInst | null>(null)
const timePresets: TimePreset[] = [
  { label: '上班 08:00', hour: 8, minute: 0 },
  { label: '上午 10:00', hour: 10, minute: 0 },
  { label: '中午 12:00', hour: 12, minute: 0 },
  { label: '下班 17:00', hour: 17, minute: 0 }
]

function onUpdateValue(value: number | null) {
  emit('update:value', value)
}

function onUpdateShow(show: boolean) {
  emit('update:show', show)
}

function applyPresetTime(preset: TimePreset) {
  const nextValue = presetTimestamp(preset.hour, preset.minute)
  const picker = pickerRef.value

  if (picker?.handlePanelUpdateValue && picker.handlePanelConfirm) {
    picker.handlePanelUpdateValue(nextValue, false)
    picker.handlePanelConfirm()
    picker.handlePanelClose?.(true)
    return
  }

  emit('update:value', nextValue)
  emit('update:show', false)
}

function presetTimestamp(hour: number, minute: number) {
  const date = new Date(currentPanelTimestamp())
  date.setHours(hour, minute, 0, 0)
  return date.getTime()
}

function currentPanelTimestamp() {
  const pendingValue = pickerRef.value?.pendingValue
  if (typeof pendingValue === 'number') return pendingValue
  if (typeof props.value === 'number') return props.value
  return Date.now()
}
</script>

<style scoped>
.time-preset-shortcuts {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 6px;
}
</style>
