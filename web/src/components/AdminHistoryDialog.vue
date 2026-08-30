<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import type { LinkHistoryEntry, RestoreTarget } from '../types'
import BaseDialog from './BaseDialog.vue'

const props=defineProps<{entries:LinkHistoryEntry[];loading:boolean;busy:boolean}>()
const emit=defineEmits<{close:[];restore:[items:RestoreTarget[]]}>()
const query=ref('')
const selected=reactive<Record<number,RestoreTarget>>({})
const filtered=computed(()=>{
  const needle=query.value.trim().toLowerCase()
  if(!needle)return props.entries
  return props.entries.filter(entry=>[
    entry.linkTitle,entry.linkSlug,entry.action,entry.actorName,entry.actorEmail,
    entry.snapshot.input.title,entry.snapshot.input.url,
  ].some(value=>value.toLowerCase().includes(needle)))
})
const selectedItems=computed(()=>Object.values(selected))
function checked(entry:LinkHistoryEntry){return selected[entry.linkId]?.revision===entry.revision}
function toggle(entry:LinkHistoryEntry){
  if(checked(entry)){delete selected[entry.linkId];return}
  selected[entry.linkId]={linkId:entry.linkId,revision:entry.revision,currentRevision:entry.currentRevision}
}
function actor(entry:LinkHistoryEntry){return entry.actorName||entry.actorEmail||'Seed import'}
</script>

<template>
  <BaseDialog title="Link change history" wide @close="$emit('close')">
    <div class="history-toolbar">
      <label>Filter history<input v-model="query" type="search" placeholder="Link, action or administrator"></label>
      <p>Select one snapshot for each link you want to restore. Other links remain unchanged.</p>
    </div>
    <div v-if="loading" class="history-empty">Loading history…</div>
    <div v-else-if="!filtered.length" class="history-empty">No matching link changes.</div>
    <ol v-else class="admin-history-list">
      <li v-for="entry in filtered" :key="`${entry.linkId}-${entry.revision}`" :class="{selected:checked(entry)}">
        <label class="history-select">
          <input type="checkbox" :checked="checked(entry)" :disabled="entry.revision===entry.currentRevision" :aria-label="`Select revision ${entry.revision} of ${entry.linkTitle}`" @change="toggle(entry)">
          <span>
            <strong>{{entry.snapshot.input.title || entry.linkTitle}}</strong>
            <small>{{entry.action}} · revision {{entry.revision}} · {{new Date(entry.createdAt).toLocaleString()}}</small>
          </span>
        </label>
        <div class="history-meta">
          <span>{{actor(entry)}}</span>
          <span v-if="entry.snapshot.deleted" class="danger-text">Deleted snapshot</span>
          <span v-else-if="entry.currentDeleted">Currently deleted</span>
          <span v-if="entry.revision===entry.currentRevision">Current</span>
        </div>
        <details>
          <summary>Snapshot details</summary>
          <dl><dt>URL</dt><dd>{{entry.snapshot.input.url}}</dd><dt>Slug</dt><dd>{{entry.snapshot.input.slug}}</dd><dt>Description</dt><dd>{{entry.snapshot.input.description || '—'}}</dd></dl>
        </details>
      </li>
    </ol>
    <footer class="history-footer"><span>{{selectedItems.length}} {{selectedItems.length===1?'link':'links'}} selected</span><div><button type="button" class="secondary" @click="$emit('close')">Cancel</button><button type="button" :disabled="busy||!selectedItems.length" @click="emit('restore',selectedItems)">{{busy?'Restoring…':`Restore ${selectedItems.length || ''}`}}</button></div></footer>
  </BaseDialog>
</template>
