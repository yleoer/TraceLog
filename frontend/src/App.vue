<template>
  <n-dialog-provider>
    <n-message-provider>
      <AppLayout />
    </n-message-provider>
  </n-dialog-provider>
</template>

<script setup lang="ts">
import { NDialogProvider, NMessageProvider } from 'naive-ui'
import AppLayout from './layouts/AppLayout.vue'
</script>
