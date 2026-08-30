<script setup lang="ts">
import { reactive } from 'vue'
import BaseDialog from './BaseDialog.vue'
import IconRenderer from './IconRenderer.vue'
import type { Category, Link, LinkInput, OwnerGroup } from '../types'
const props=defineProps<{ link?:Link; categories:Category[]; groups:OwnerGroup[]; busy:boolean }>()
const emit=defineEmits<{ close:[]; save:[input:LinkInput] }>()
const form=reactive<LinkInput>(props.link?{slug:props.link.slug,title:props.link.title,url:props.link.url,description:props.link.description,iconProvider:props.link.iconProvider,iconName:props.link.iconName,iconColor:props.link.iconColor,categoryId:props.link.categoryId,ownerGroupId:props.link.ownerGroupId,tags:[...props.link.tags],roles:[...props.link.roles],teams:[...props.link.teams],enabled:true,revision:props.link.revision}:{slug:'',title:'',url:'https://',description:'',iconProvider:'',iconName:'',iconColor:'',categoryId:props.categories[0]?.id??0,ownerGroupId:props.groups[0]?.id??0,tags:[],roles:[],teams:[],enabled:true,revision:0})
const arrays={tags:form.tags.join(', '),roles:form.roles.join(', '),teams:form.teams.join(', ')}
function save(){emit('save',{...form,tags:split(arrays.tags),roles:split(arrays.roles),teams:split(arrays.teams)})}
function split(v:string){return [...new Set(v.split(',').map(x=>x.trim()).filter(Boolean))]}
</script>
<template>
  <BaseDialog :title="link ? `Edit ${link.title}` : 'Add link'" wide @close="$emit('close')">
    <form class="link-form" @submit.prevent="save">
      <label>Title<input v-model.trim="form.title" required maxlength="120"></label>
      <label>Slug<input v-model.trim="form.slug" required pattern="[a-z0-9][a-z0-9-]{0,62}" placeholder="stirling-pdf"></label>
      <label class="span-2">URL<input v-model.trim="form.url" required maxlength="2048"></label>
      <label class="span-2">Description<textarea v-model.trim="form.description" rows="3" maxlength="500"></textarea></label>
      <label>Category<select v-model.number="form.categoryId" required><option v-for="c in categories" :key="c.id" :value="c.id">{{ c.name }}</option></select></label>
      <label>Owner group<select v-model.number="form.ownerGroupId" required><option v-for="g in groups" :key="g.id" :value="g.id">{{ g.name }}</option></select></label>
      <label>Icon provider<select v-model="form.iconProvider"><option value="">Automatic</option><option value="devicon">Devicon</option><option value="fontawesome">Font Awesome</option><option value="material">Material</option></select></label>
      <label>Icon name<input v-model.trim="form.iconName" placeholder="go-plain, server, home"><small>Optional; automatic icons use the destination.</small></label>
      <label>Icon color<input v-model.trim="form.iconColor" type="color"></label>
      <div class="icon-preview" aria-live="polite"><span>Preview</span><IconRenderer :title="form.title || 'Link'" :url="form.url" :provider="form.iconProvider" :name="form.iconName" :color="form.iconColor" colored /><small>{{ form.iconProvider || 'Automatic' }} · {{ form.iconName || 'inferred from URL' }}</small></div>
      <label>Tags<input v-model="arrays.tags" placeholder="pdf, documents"></label>
      <label>Roles<input v-model="arrays.roles" placeholder="developer, operator"></label>
      <label>Teams<input v-model="arrays.teams" placeholder="platform, security"></label>
      <label class="checkbox"><input v-model="form.enabled" type="checkbox"> Enabled</label>
      <footer class="span-2"><button type="button" class="secondary" @click="$emit('close')">Cancel</button><button type="submit" :disabled="busy">{{ busy ? 'Saving…' : 'Save link' }}</button></footer>
    </form>
  </BaseDialog>
</template>
