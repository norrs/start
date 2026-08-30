<script setup lang="ts">
import BaseDialog from './BaseDialog.vue'
import type { Preferences } from '../types'
defineProps<{ preferences:Preferences }>();defineEmits<{ close:[]; change:[value:Preferences] }>()
function update(p:Preferences,key:keyof Preferences,value:unknown){return {...p,[key]:value}}
</script>
<template>
  <BaseDialog title="Display settings" @close="$emit('close')">
    <div class="settings-list">
      <label>Theme<select :value="preferences.theme" @change="$emit('change',update(preferences,'theme',($event.target as HTMLSelectElement).value))"><option value="dark">Dark</option><option value="light">Light</option><option value="hacker">Hacker</option><option value="1984">1984</option></select></label>
      <label><input type="checkbox" :checked="preferences.showDescriptions" @change="$emit('change',update(preferences,'showDescriptions',($event.target as HTMLInputElement).checked))"> Show descriptions</label>
      <label><input type="checkbox" :checked="preferences.showCategories" @change="$emit('change',update(preferences,'showCategories',($event.target as HTMLInputElement).checked))"> Show category headings</label>
      <label><input type="checkbox" :checked="preferences.showIcons" @change="$emit('change',update(preferences,'showIcons',($event.target as HTMLInputElement).checked))"> Show icons</label>
      <label><input type="checkbox" :checked="preferences.coloredIcons" @change="$emit('change',update(preferences,'coloredIcons',($event.target as HTMLInputElement).checked))"> Colored icons</label>
      <label><input type="checkbox" :checked="preferences.showHidden" @change="$emit('change',update(preferences,'showHidden',($event.target as HTMLInputElement).checked))"> Show links I have hidden</label>
      <label><input type="checkbox" :checked="preferences.openNewTab" @change="$emit('change',update(preferences,'openNewTab',($event.target as HTMLInputElement).checked))"> Open links in a new tab</label>
      <label><input type="checkbox" :checked="preferences.hotkeysEnabled" @change="$emit('change',update(preferences,'hotkeysEnabled',($event.target as HTMLInputElement).checked))"> Single-key hotkeys</label>
    </div>
  </BaseDialog>
</template>
