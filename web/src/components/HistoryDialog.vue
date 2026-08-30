<script setup lang="ts">
import BaseDialog from './BaseDialog.vue';import type{Link,Revision}from'../types';defineProps<{link:Link;revisions:Revision[];busy:boolean}>();defineEmits<{close:[];restore:[revision:number]}>()
</script>
<template><BaseDialog :title="`${link.title} history`" @close="$emit('close')"><ol class="history-list"><li v-for="r in revisions" :key="r.revision"><span><strong>Revision {{r.revision}}</strong><small>{{r.action}} · {{new Date(r.createdAt).toLocaleString()}}</small></span><button type="button" class="secondary" :disabled="busy||r.revision===link.revision" @click="$emit('restore',r.revision)">Restore</button></li></ol></BaseDialog></template>
