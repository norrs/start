<script setup lang="ts">
import { nextTick, onMounted, onUnmounted, ref, useId } from 'vue'
const props=defineProps<{ title:string; wide?:boolean }>();const emit=defineEmits<{ close:[] }>();const panel=ref<HTMLElement>()
const titleId=useId();let previousFocus:HTMLElement|null=null
function focusable(){return [...(panel.value?.querySelectorAll<HTMLElement>('button:not([disabled]),input:not([disabled]),select:not([disabled]),textarea:not([disabled]),a[href],[tabindex]:not([tabindex="-1"])')??[])]}
function keydown(e:KeyboardEvent){
  if(e.key==='Escape'){e.preventDefault();emit('close');return}
  if(e.key!=='Tab')return
  const items=focusable();if(!items.length){e.preventDefault();return}
  const first=items[0],last=items[items.length-1]
  if(e.shiftKey&&document.activeElement===first){e.preventDefault();last.focus()}
  else if(!e.shiftKey&&document.activeElement===last){e.preventDefault();first.focus()}
}
onMounted(async()=>{
  previousFocus=document.activeElement as HTMLElement|null
  document.addEventListener('keydown',keydown)
  await nextTick()
  const firstField=panel.value?.querySelector<HTMLElement>('input:not([disabled]),select:not([disabled]),textarea:not([disabled])')
  if(firstField)firstField.focus()
  else focusable()[0]?.focus()
})
onUnmounted(()=>{document.removeEventListener('keydown',keydown);previousFocus?.focus()})
</script>
<template>
  <div class="dialog-backdrop" role="presentation" @mousedown.self="$emit('close')">
    <section ref="panel" class="dialog" :class="{wide}" role="dialog" aria-modal="true" :aria-labelledby="titleId">
      <header><h2 :id="titleId">{{ title }}</h2><button type="button" class="icon-button" aria-label="Close" @click="$emit('close')">×</button></header>
      <slot />
    </section>
  </div>
</template>
