<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ title:string; url:string; provider:string; name:string; color:string; colored:boolean }>()
type Icon = { provider:'devicon'|'fontawesome'|'material'|'fallback'; name:string; style?:string }

const hostIcons:Record<string,Icon> = {
  'github.com': {provider:'devicon',name:'github-original'},
  'gitlab.com': {provider:'devicon',name:'gitlab-plain'},
  'grafana': {provider:'devicon',name:'grafana-plain'},
  'prometheus': {provider:'devicon',name:'prometheus-original'},
  'kubernetes': {provider:'devicon',name:'kubernetes-plain'},
  'docker': {provider:'devicon',name:'docker-plain'},
  'stirling': {provider:'fontawesome',name:'file-pdf'},
}
function safe(value:string){return value.toLowerCase().trim().replace(/[^a-z0-9_:-]/g,'')}
function infer():Icon {
  let hostname=''
  try {
    const parsed=new URL(props.url)
    hostname=parsed.hostname.toLowerCase()
    if(parsed.protocol==='mailto:')return {provider:'fontawesome',name:'envelope'}
  } catch { /* fall through to the generic icon */ }
  const match=Object.entries(hostIcons).find(([needle])=>hostname===needle||hostname.includes(needle))
  return match?.[1] ?? {provider:'fontawesome',name:'link'}
}
const icon=computed<Icon>(()=>{
  const provider=props.provider.toLowerCase().trim()
  const name=safe(props.name)
  if(!provider)return name ? {provider:'material',name} : infer()
  if(!name)return {provider:'fallback',name:props.title.trim().charAt(0).toUpperCase()||'↗'}
  if(provider==='devicon')return {provider,name:name.replace(/^devicon-/,'').replace(/-colored$/,'')}
  if(provider==='material')return {provider,name:name.replace(/^material-icons-/,'')}
  if(provider==='fontawesome'){
    const normalized=name.replace(/^fa-/,'')
    const [style,iconName]=normalized.includes(':') ? normalized.split(':',2) : ['solid',normalized]
    return {provider,name:iconName,style:['solid','regular','brands'].includes(style)?style:'solid'}
  }
  return {provider:'fallback',name:props.title.trim().charAt(0).toUpperCase()||'↗'}
})
const customColor=computed(()=>props.colored&&props.color?{color:props.color}:undefined)
const deviconClass=computed(()=>[`devicon-${icon.value.name}`,props.colored&&!props.color?'colored':''])
const fontawesomeClass=computed(()=>[`fa-${icon.value.style||'solid'}`,`fa-${icon.value.name}`])
</script>
<template>
  <span class="link-icon" :style="customColor" aria-hidden="true" :title="name || 'Automatic icon'">
    <i v-if="icon.provider==='devicon'" :class="deviconClass"></i>
    <i v-else-if="icon.provider==='fontawesome'" :class="fontawesomeClass"></i>
    <i v-else-if="icon.provider==='material'" class="material-icons">{{ icon.name }}</i>
    <span v-else>{{ icon.name }}</span>
  </span>
</template>
