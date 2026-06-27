import type { Component } from 'vue'
import {
  ArrowDown,
  ArrowUp,
  Bookmark,
  Bug,
  ChevronsDown,
  ChevronsUp,
  Circle,
  Equal,
  FileText,
  SquareCheckBig,
  Zap
} from 'lucide-vue-next'

export interface JiraMeta {
  issueType: string
  jiraStatus: string
  jiraPriority: string
  releaseRequested: string
  description: string
}

export interface JiraVisual {
  label: string
  icon: Component
  tone: 'blue' | 'green' | 'red' | 'purple' | 'gray'
}

export interface PriorityVisual {
  label: string
  icon: Component
  tone: 'red' | 'orange' | 'blue' | 'gray'
}

export function parseJiraMeta(markdown: string): JiraMeta {
  return {
    issueType: readBullet(markdown, 'Type'),
    jiraStatus: readBullet(markdown, 'Jira status'),
    jiraPriority: readBullet(markdown, 'Jira priority'),
    releaseRequested: readFirstBullet(markdown, [
      '发布请求',
      'ReleaseRequested',
      'Release Requested',
      'Fix versions'
    ]),
    description: readSection(markdown, 'Jira Description')
  }
}

export function priorityDisplayName(localPriority: string, jiraPriority: string) {
  if (jiraPriority) return jiraPriority
  return {
    urgent: 'Highest',
    high: 'High',
    medium: 'Medium',
    low: 'Low'
  }[localPriority] ?? localPriority
}

export function statusDisplayName(localStatus: string, markdown: string) {
  return parseJiraMeta(markdown).jiraStatus || localStatus || '-'
}

export function issueTypeVisual(issueType: string): JiraVisual | null {
  if (!issueType) return null
  const name = issueType.toLowerCase()
  if (name.includes('bug')) return { label: issueType, icon: Bug, tone: 'red' }
  if (name.includes('story')) return { label: issueType, icon: Bookmark, tone: 'green' }
  if (name.includes('sub')) return { label: issueType, icon: FileText, tone: 'blue' }
  if (name.includes('task')) return { label: issueType, icon: SquareCheckBig, tone: 'blue' }
  if (name.includes('epic')) return { label: issueType, icon: Zap, tone: 'purple' }
  return { label: issueType, icon: Circle, tone: 'gray' }
}

export function priorityVisual(localPriority: string, jiraPriority: string): PriorityVisual {
  const label = priorityDisplayName(localPriority, jiraPriority)
  const name = label.toLowerCase()
  if (name.includes('highest') || name.includes('blocker') || name.includes('critical') || name.includes('urgent')) {
    return { label, icon: ChevronsUp, tone: 'red' }
  }
  if (name.includes('high') || name.includes('major')) return { label, icon: ArrowUp, tone: 'orange' }
  if (name.includes('lowest') || name.includes('trivial')) return { label, icon: ChevronsDown, tone: 'blue' }
  if (name.includes('low') || name.includes('minor')) return { label, icon: ArrowDown, tone: 'blue' }
  if (name.includes('medium')) return { label, icon: Equal, tone: 'gray' }
  return { label, icon: Equal, tone: 'gray' }
}

export interface StatusVisual {
  label: string
  tone: 'green' | 'blue' | 'orange' | 'gray'
}

export function statusVisual(localStatus: string, markdown: string): StatusVisual {
  const label = statusDisplayName(localStatus, markdown)
  const haystack = `${localStatus} ${label}`.toLowerCase()
  if (/done|closed|resolved|complete/.test(haystack)) return { label, tone: 'green' }
  if (/block|suspend|hold|pending|waiting/.test(haystack)) return { label, tone: 'orange' }
  if (!label || label === '-' || /analysis|todo|to do|new|open|backlog/.test(haystack)) {
    return { label, tone: 'gray' }
  }
  return { label, tone: 'blue' }
}

export function toneColor(tone: string): { color: string; textColor: string; borderColor: string } {
  switch (tone) {
    case 'blue':
      return { color: '#dbeafe', textColor: '#2563eb', borderColor: '#dbeafe' }
    case 'green':
      return { color: '#dcfce7', textColor: '#16a34a', borderColor: '#dcfce7' }
    case 'red':
      return { color: '#fee2e2', textColor: '#dc2626', borderColor: '#fee2e2' }
    case 'orange':
      return { color: '#ffedd5', textColor: '#ea580c', borderColor: '#ffedd5' }
    case 'purple':
      return { color: '#ede9fe', textColor: '#7c3aed', borderColor: '#ede9fe' }
    default:
      return { color: '#f3f4f6', textColor: '#4b5563', borderColor: '#f3f4f6' }
  }
}

function readBullet(markdown: string, label: string) {
  const escaped = label.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const match = markdown.match(new RegExp(`^- ${escaped}:\\s*(.+)$`, 'im'))
  return match?.[1]?.trim() ?? ''
}

function readFirstBullet(markdown: string, labels: string[]) {
  for (const label of labels) {
    const value = readBullet(markdown, label)
    if (value) return value
  }
  return ''
}

function readSection(markdown: string, title: string) {
  const escaped = title.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const heading = new RegExp(`^##\\s+${escaped}\\s*$`, 'im')
  const match = heading.exec(markdown)
  if (!match) return ''
  const rest = markdown.slice(match.index + match[0].length)
  const nextHeading = rest.search(/^##\s+/m)
  return rest.slice(0, nextHeading === -1 ? undefined : nextHeading).trim()
}
