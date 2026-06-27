import { computed, onMounted, watch, type Ref } from 'vue'

export function useDraftAutosave(key: Ref<string>, value: Ref<string>, options: { enabled?: Ref<boolean>; delay?: number } = {}) {
  const restored = computed(() => key.value ? localStorage.getItem(key.value) ?? '' : '')
  let timer: number | undefined

  onMounted(() => {
    if (!key.value) return
    const draft = localStorage.getItem(key.value)
    if (draft && !value.value) {
      value.value = draft
    }
  })

  watch(
    [key, value, options.enabled ?? computed(() => true)],
    ([currentKey, currentValue, enabled]) => {
      window.clearTimeout(timer)
      if (!currentKey || !enabled) return
      timer = window.setTimeout(() => {
        if (currentValue.trim()) {
          localStorage.setItem(currentKey, currentValue)
        } else {
          localStorage.removeItem(currentKey)
        }
      }, options.delay ?? 500)
    }
  )

  function clearDraft() {
    if (key.value) {
      localStorage.removeItem(key.value)
    }
  }

  return { restored, clearDraft }
}
