<template>
  <n-config-provider :locale="zhCN" :date-locale="dateZhCN" :theme-overrides="themeOverrides">
    <n-dialog-provider>
      <n-message-provider>
        <AppLayout />
      </n-message-provider>
    </n-dialog-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { dateZhCN, NConfigProvider, NDialogProvider, NMessageProvider, zhCN } from 'naive-ui'
import type { GlobalThemeOverrides } from 'naive-ui'
import AppLayout from './layouts/AppLayout.vue'
import { useWorkTheme } from './composables/useWorkTheme'

const { activeTheme } = useWorkTheme()
const themeOverrides = computed<GlobalThemeOverrides>(() => {
  const theme = activeTheme.value
  return {
  common: {
    primaryColor: theme.primary,
    primaryColorHover: theme.hover,
    primaryColorPressed: theme.pressed,
    primaryColorSuppl: theme.highlight,
    borderRadius: '9px',
    borderColor: 'rgba(71, 84, 103, 0.14)',
    dividerColor: 'rgba(71, 84, 103, 0.11)',
    textColorBase: '#172033',
    textColor1: '#172033',
    textColor2: '#475467',
    textColor3: '#7b8496',
    bodyColor: 'transparent',
    cardColor: 'rgba(255, 255, 255, 0.86)',
    modalColor: 'rgba(250, 252, 255, 0.92)',
    popoverColor: 'rgba(255, 255, 255, 0.94)',
    tableColor: 'transparent',
    inputColor: 'rgba(255, 255, 255, 0.72)',
    inputColorDisabled: 'rgba(238, 242, 248, 0.72)',
    hoverColor: `rgba(${theme.rgb}, 0.07)`,
    actionColor: 'rgba(244, 247, 251, 0.82)',
    boxShadow1: '0 12px 34px rgba(50, 69, 98, 0.12)',
    boxShadow2: '0 20px 60px rgba(43, 61, 88, 0.16)',
    boxShadow3: '0 28px 80px rgba(37, 54, 80, 0.2)',
    fontFamily: "Inter, -apple-system, BlinkMacSystemFont, 'SF Pro Text', 'Segoe UI', sans-serif",
    fontWeightStrong: '650'
  },
  Button: {
    borderRadiusSmall: '8px',
    borderRadiusMedium: '9px',
    heightSmall: '34px',
    fontWeight: '600'
  },
  Input: {
    borderRadius: '9px',
    heightSmall: '36px',
    boxShadowFocus: `0 0 0 3px rgba(${theme.rgb}, 0.14)`
  },
  Select: {
    peers: {
      InternalSelection: {
        borderRadius: '9px',
        heightSmall: '36px',
        boxShadowActive: `0 0 0 3px rgba(${theme.rgb}, 0.14)`
      }
    }
  },
  DataTable: {
    thColor: 'rgba(245, 248, 252, 0.72)',
    thColorHover: 'rgba(245, 248, 252, 0.92)',
    tdColor: 'transparent',
    tdColorHover: `rgba(${theme.rgb}, 0.055)`,
    borderColor: 'rgba(71, 84, 103, 0.09)',
    thTextColor: '#667085',
    tdTextColor: '#344054'
  },
  Dialog: {
    borderRadius: '14px'
  },
  Message: {
    borderRadius: '10px'
  },
  Tag: {
    borderRadius: '999px'
  }
  }
})
</script>
