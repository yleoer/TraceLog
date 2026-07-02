<template>
  <n-config-provider :locale="zhCN" :date-locale="dateZhCN">
    <n-dialog-provider>
      <n-message-provider>
        <AppLayout />
      </n-message-provider>
    </n-dialog-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import { dateZhCN, NConfigProvider, NDialogProvider, NMessageProvider, zhCN } from 'naive-ui'
import AppLayout from './layouts/AppLayout.vue'
</script>
