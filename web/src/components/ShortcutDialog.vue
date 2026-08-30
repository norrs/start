<script setup lang="ts">
import BaseDialog from './BaseDialog.vue'
defineEmits<{ close: [] }>()
</script>
<template><BaseDialog title="Keyboard shortcuts" @close="$emit('close')"><dl class="shortcut-list"><template v-for="[key,label] in [['/ or Ctrl/⌘ K','Focus search'],['J / K or ↓ / ↑','Move between links'],['Enter','Open focused link'],['V','Toggle categorized / flat'],['H','Hide or show focused link'],['E','Edit focused link'],['N','Add a link'],['G','Open link change history (admins)'],['I','Import link catalog (system admins)'],['Esc','Close or clear'],['?','Show shortcuts']]" :key="key"><dt><kbd>{{key}}</kbd></dt><dd>{{label}}</dd></template></dl></BaseDialog></template>
