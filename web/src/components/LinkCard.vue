<script setup lang="ts">
import type { Link, Preferences } from '../types'
import IconRenderer from './IconRenderer.vue'
defineProps<{ link:Link; preferences:Preferences; active:boolean }>()
defineEmits<{ hide:[link:Link]; edit:[link:Link]; remove:[link:Link]; history:[link:Link] }>()
function destination(value:string){try{return new URL(value).hostname||value}catch{return value}}
</script>
<template>
  <article class="link-card" :class="{ hidden:link.hidden, active }" :data-link-id="link.id">
    <a class="card-main" :href="link.url" :target="preferences.openNewTab ? '_blank' : undefined" :rel="preferences.openNewTab ? 'noopener noreferrer' : undefined">
      <IconRenderer v-if="preferences.showIcons" :title="link.title" :url="link.url" :provider="link.iconProvider" :name="link.iconName" :color="link.iconColor" :colored="preferences.coloredIcons" />
      <span class="card-copy">
        <strong>{{ link.title }}</strong>
        <small>{{ destination(link.url) }}</small>
        <span v-if="preferences.showDescriptions && link.description" class="description">{{ link.description }}</span>
        <span v-if="link.tags.length" class="tags"><span v-for="tag in link.tags" :key="tag">{{ tag }}</span></span>
      </span>
    </a>
    <div class="card-actions">
      <button type="button" @click="$emit('hide',link)" :aria-label="link.hidden ? `Show ${link.title}` : `Hide ${link.title}`">{{ link.hidden ? 'Show' : 'Hide' }}</button>
      <button v-if="link.canManage" type="button" @click="$emit('edit',link)">Edit</button>
      <button v-if="link.canManage" type="button" @click="$emit('history',link)">History</button>
      <button v-if="link.canManage" class="danger" type="button" @click="$emit('remove',link)">Delete</button>
    </div>
  </article>
</template>
