export interface LabeledOption {
  label: string
  value: string
}

export const tempTaskStatusOptions: LabeledOption[] = [
  { label: '待处理', value: 'todo' },
  { label: '处理中', value: 'processing' },
  { label: '已完成', value: 'done' },
  { label: '挂起', value: 'suspended' }
]

export const tempTaskPriorityOptions: LabeledOption[] = [
  { label: '低', value: 'low' },
  { label: '中', value: 'medium' },
  { label: '高', value: 'high' },
  { label: '紧急', value: 'urgent' }
]

export function tempStatusLabel(value: string): string {
  return tempTaskStatusOptions.find((option) => option.value === value)?.label ?? value
}

export function tempPriorityLabel(value: string): string {
  return tempTaskPriorityOptions.find((option) => option.value === value)?.label ?? value
}
