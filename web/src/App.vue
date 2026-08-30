<script setup lang="ts">
import Fuse from 'fuse.js'
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { api } from './api'
import type { Bootstrap, Link, LinkHistoryEntry, LinkInput, Preferences, RestoreTarget, Revision } from './types'
import AdminHistoryDialog from './components/AdminHistoryDialog.vue'
import CatalogImportDialog from './components/CatalogImportDialog.vue'
import HistoryDialog from './components/HistoryDialog.vue'
import LinkCard from './components/LinkCard.vue'
import LinkDialog from './components/LinkDialog.vue'
import PreferencesDialog from './components/PreferencesDialog.vue'
import ShortcutDialog from './components/ShortcutDialog.vue'

const data = ref<Bootstrap>()
const loading = ref(true)
const failure = ref('')
const query = ref('')
const search = ref<HTMLInputElement>()
const settingsMenu = ref<HTMLDetailsElement>()
const activeIndex = ref(0)
const editing = ref<Link | null>()
const adding = ref(false)
const preferencesOpen = ref(false)
const shortcutsOpen = ref(false)
const historyLink = ref<Link | null>()
const revisions = ref<Revision[]>([])
const adminHistoryOpen = ref(false)
const catalogImportOpen = ref(false)
const historyEntries = ref<LinkHistoryEntry[]>([])
const historyLoading = ref(false)
const busy = ref(false)
const notice = ref('')
let saveTimer: number | undefined

const preferences = computed(() => data.value?.preferences)
const fuse = computed(() => new Fuse(data.value?.links ?? [], {
  threshold: .34,
  ignoreLocation: true,
  keys: [{name:'title',weight:5},{name:'tags',weight:4},{name:'url',weight:3},{name:'description',weight:2}],
}))
const filtered = computed(() => query.value.trim() ? fuse.value.search(query.value.trim()).map(result => result.item) : (data.value?.links ?? []))
const sections = computed(() => {
  if (!data.value || data.value.preferences.viewMode === 'flat' || query.value.trim()) return [{ id:0, name:query.value.trim() ? 'Search results' : 'All links', color:data.value?.branding.accentColor ?? '#38bdf8', links:filtered.value }]
  return data.value.categories.map(category => ({ id:category.id, name:category.name, color:category.color, links:filtered.value.filter(link => link.categoryId === category.id) })).filter(section => section.links.length)
})
const activeLink = computed(() => filtered.value[Math.min(activeIndex.value, Math.max(0, filtered.value.length - 1))])
const canAdd = computed(() => data.value?.ownerGroups.some(group => group.canCreate) || false)
const creatableGroups = computed(() => data.value?.ownerGroups.filter(group => group.canCreate) ?? [])
const dialogOpen = computed(() => adding.value || !!editing.value || preferencesOpen.value || shortcutsOpen.value || !!historyLink.value || adminHistoryOpen.value || catalogImportOpen.value)
function preferenceKey(){return `start.preferences.${data.value?.user.email.toLowerCase() || 'anonymous'}`}

watch(filtered, links => { if (activeIndex.value >= links.length) activeIndex.value = Math.max(0, links.length - 1) })
watch(() => preferences.value?.theme, theme => { if (theme) document.documentElement.dataset.theme = theme }, { immediate:true })

async function load() {
  try {
    const value = await api.bootstrap()
    data.value = value
    localStorage.setItem(preferenceKey(), JSON.stringify(value.preferences))
  } catch (error) { failure.value = message(error) }
  finally { loading.value = false }
}
function message(error:unknown){return error instanceof Error ? error.message : 'Something went wrong'}
function flash(value:string){notice.value=value;window.setTimeout(()=>{if(notice.value===value)notice.value=''},3000)}
function updatePreferences(value:Preferences) {
  if (!data.value) return
  const hiddenVisibilityChanged = value.showHidden !== data.value.preferences.showHidden
  data.value.preferences = value
  localStorage.setItem(preferenceKey(), JSON.stringify(value))
  window.clearTimeout(saveTimer)
  saveTimer = window.setTimeout(async () => {
    try { if(data.value) { data.value.preferences = await api.savePreferences(data.value.preferences); if(hiddenVisibilityChanged) await reload() } }
    catch(error){flash(`Could not save settings: ${message(error)}`)}
  }, 250)
}
function toggleView(){if(!preferences.value)return;updatePreferences({...preferences.value,viewMode:preferences.value.viewMode==='flat'?'categorized':'flat'})}
function openLink(link?:Link){if(!link||!preferences.value)return;window.open(link.url,preferences.value.openNewTab?'_blank':'_self',preferences.value.openNewTab?'noopener,noreferrer':undefined)}
async function hide(link:Link){
  try { await api.hide(link.id,!link.hidden);link.hidden=!link.hidden;if(!preferences.value?.showHidden&&link.hidden&&data.value)data.value.links=data.value.links.filter(item=>item.id!==link.id);flash(link.hidden?'Link hidden':'Link shown') }
  catch(error){flash(message(error))}
}
async function saveLink(input:LinkInput){
  busy.value=true
  try { if(editing.value)await api.update(editing.value.id,input);else await api.create(input);editing.value=null;adding.value=false;await reload();flash('Link saved') }
  catch(error){flash(message(error))}finally{busy.value=false}
}
async function remove(link:Link){
  if(!confirm(`Delete “${link.title}”?\n\nThe link remains recoverable from its revision history.`))return
  busy.value=true
  try{await api.remove(link.id,link.revision);if(data.value)data.value.links=data.value.links.filter(item=>item.id!==link.id);flash('Link deleted')}
  catch(error){flash(message(error))}finally{busy.value=false}
}
async function showHistory(link:Link){historyLink.value=link;revisions.value=[];try{revisions.value=await api.revisions(link.id)}catch(error){flash(message(error));historyLink.value=null}}
async function restore(revision:number){
  if(!historyLink.value||!confirm(`Restore revision ${revision} of “${historyLink.value.title}”?`))return
  busy.value=true
  try{await api.restore(historyLink.value.id,revision);historyLink.value=null;await reload();flash('Revision restored')}
  catch(error){flash(message(error))}finally{busy.value=false}
}
async function showAdminHistory(){
  adminHistoryOpen.value=true
  historyLoading.value=true
  try{historyEntries.value=await api.linkHistory()}
  catch(error){flash(message(error));adminHistoryOpen.value=false}
  finally{historyLoading.value=false}
}
async function restoreHistory(items:RestoreTarget[]){
  if(!confirm(`Restore the selected snapshot for ${items.length} ${items.length===1?'link':'links'}?\n\nUnselected links will not change.`))return
  busy.value=true
  try{await api.restoreHistory(items);await reload();historyEntries.value=await api.linkHistory();flash(`${items.length} ${items.length===1?'link':'links'} restored`)}
  catch(error){flash(message(error));try{historyEntries.value=await api.linkHistory()}catch{/* keep the original error */}}
  finally{busy.value=false}
}
async function importCatalog(file:File){
  busy.value=true
  try{
    const result=await api.importCatalog(file)
    catalogImportOpen.value=false
    await reload()
    flash(`Catalog imported: ${result.created} created, ${result.updated} updated, ${result.restored} restored, ${result.unchanged} unchanged`)
  }catch(error){flash(message(error))}
  finally{busy.value=false}
}
async function reload(){const value=await api.bootstrap();if(data.value)value.preferences=data.value.preferences;data.value=value}
function openDisplaySettings(){if(settingsMenu.value)settingsMenu.value.open=false;preferencesOpen.value=true}
function openCatalogImport(){if(settingsMenu.value)settingsMenu.value.open=false;catalogImportOpen.value=true}
function isTyping(target:EventTarget|null){const el=target as HTMLElement|null;return !!el&&(el.matches('input,textarea,select')||el.isContentEditable)}
async function keydown(event:KeyboardEvent){
  const command=event.ctrlKey||event.metaKey
  if(command&&event.key.toLowerCase()==='k'){event.preventDefault();search.value?.focus();return}
  if(event.key==='Escape'&&!dialogOpen.value&&query.value){event.preventDefault();query.value='';search.value?.focus();return}
  if(event.key==='Enter'&&event.target===search.value&&filtered.value.length===1){event.preventDefault();openLink(filtered.value[0]);return}
  if(isTyping(event.target)||dialogOpen.value||!preferences.value?.hotkeysEnabled)return
  if(event.key==='/' ){event.preventDefault();search.value?.focus();return}
  if(event.key==='?'){event.preventDefault();shortcutsOpen.value=true;return}
  if(event.key==='j'||event.key==='ArrowDown'){event.preventDefault();activeIndex.value=Math.min(filtered.value.length-1,activeIndex.value+1);await reveal();return}
  if(event.key==='k'||event.key==='ArrowUp'){event.preventDefault();activeIndex.value=Math.max(0,activeIndex.value-1);await reveal();return}
  if(event.key==='Enter'){event.preventDefault();openLink(filtered.value.length===1?filtered.value[0]:activeLink.value);return}
  if(event.key.toLowerCase()==='v'){event.preventDefault();toggleView();return}
  if(event.key.toLowerCase()==='h'&&activeLink.value){event.preventDefault();await hide(activeLink.value);return}
  if(event.key.toLowerCase()==='e'&&activeLink.value?.canManage){event.preventDefault();editing.value=activeLink.value;return}
  if(event.key.toLowerCase()==='n'&&canAdd.value){event.preventDefault();adding.value=true}
  if(event.key.toLowerCase()==='g'&&data.value?.user.canManageLinks){event.preventDefault();await showAdminHistory()}
  if(event.key.toLowerCase()==='i'&&data.value?.user.systemAdmin){event.preventDefault();openCatalogImport()}
}
async function reveal(){await nextTick();document.querySelector(`[data-link-id="${activeLink.value?.id}"]`)?.scrollIntoView({block:'nearest'})}
async function logout(){try{const result=await api.logout();location.assign(result.redirectUrl)}catch(error){flash(message(error))}}
function retry(){window.location.reload()}
onMounted(()=>{window.addEventListener('keydown',keydown);void load()})
onUnmounted(()=>{window.removeEventListener('keydown',keydown);window.clearTimeout(saveTimer)})
</script>

<template>
  <div class="app-shell" :style="data ? {'--brand-accent':data.branding.accentColor} : undefined">
    <header class="topbar">
      <a class="brand" href="/" aria-label="Start home">
        <img v-if="data?.branding.logoUrl" :src="data.branding.logoUrl" alt="">
        <span v-else class="brand-mark">S</span>
        <span><strong>{{ data?.branding.appName || 'Start' }}</strong><small>internal launchpad</small></span>
      </a>
      <nav v-if="data" aria-label="Account and settings">
        <button type="button" class="secondary shortcut-button" @click="shortcutsOpen=true" title="Keyboard shortcuts (?)">?</button>
        <button v-if="data.user.canManageLinks" type="button" class="secondary history-button" @click="showAdminHistory">History</button>
        <details ref="settingsMenu" class="settings-menu"><summary aria-label="Settings">Settings</summary><div class="account-menu"><button type="button" @click="openDisplaySettings">Display settings</button><button v-if="data.user.systemAdmin" type="button" @click="openCatalogImport">Import YAML</button></div></details>
        <details class="account-details"><summary :aria-label="`Account for ${data.user.displayName || data.user.email}`">Welcome {{ data.user.displayName || data.user.email }}</summary><div class="account-menu"><small>{{ data.user.email }}</small><button type="button" @click="logout">Sign out</button></div></details>
      </nav>
    </header>

    <main>
      <section v-if="loading" class="state-panel"><span class="spinner"></span><p>Opening your launchpad…</p></section>
      <section v-else-if="failure" class="state-panel error"><h1>Start could not load</h1><p>{{ failure }}</p><button type="button" @click="retry">Try again</button></section>
      <template v-else-if="data && preferences">
        <section class="toolbar" aria-label="Find and arrange links">
          <label class="search-box"><span>⌕</span><input ref="search" v-model="query" type="search" autocomplete="off" placeholder="Search titles, descriptions, URLs and tags…" aria-label="Search links" @input="activeIndex=0"><kbd>/</kbd></label>
          <div class="view-switch" role="group" aria-label="View mode"><button type="button" :class="{selected:preferences.viewMode==='categorized'}" @click="updatePreferences({...preferences,viewMode:'categorized'})">Categories</button><button type="button" :class="{selected:preferences.viewMode==='flat'}" @click="updatePreferences({...preferences,viewMode:'flat'})">All links</button></div>
          <button v-if="canAdd" type="button" class="add-link-button" @click="adding=true"><span>＋</span> Add link <kbd>N</kbd></button>
        </section>

        <div v-if="!filtered.length" class="empty-state"><span>⌕</span><h2>No links match “{{ query }}”</h2><p>Try a shorter phrase or clear the search.</p><button type="button" class="secondary" @click="query='';search?.focus()">Clear search</button></div>
        <section v-for="section in sections" v-else :key="section.id" class="link-section">
          <header v-if="preferences.showCategories"><span class="category-dot" :style="{background:section.color}"></span><h2>{{ section.name }}</h2><small>{{ section.links.length }}</small></header>
          <div class="link-grid">
            <LinkCard v-for="link in section.links" :key="link.id" :link="link" :preferences="preferences" :active="activeLink?.id===link.id" @mouseenter="activeIndex=filtered.findIndex(item=>item.id===link.id)" @hide="hide" @edit="editing=$event" @remove="remove" @history="showHistory" />
          </div>
        </section>
      </template>
    </main>
    <footer class="page-footer"><span><i></i> Start is ready</span><span>Press <kbd>?</kbd> for keyboard help</span></footer>
    <div v-if="notice" class="toast" role="status">{{ notice }}</div>
    <LinkDialog v-if="data&&(adding||editing)" :link="editing||undefined" :categories="data.categories" :groups="editing ? data.ownerGroups : creatableGroups" :busy="busy" @close="adding=false;editing=null" @save="saveLink" />
    <PreferencesDialog v-if="preferences&&preferencesOpen" :preferences="preferences" @close="preferencesOpen=false" @change="updatePreferences" />
    <ShortcutDialog v-if="shortcutsOpen" @close="shortcutsOpen=false" />
    <HistoryDialog v-if="historyLink" :link="historyLink" :revisions="revisions" :busy="busy" @close="historyLink=null" @restore="restore" />
    <AdminHistoryDialog v-if="adminHistoryOpen" :entries="historyEntries" :loading="historyLoading" :busy="busy" @close="adminHistoryOpen=false" @restore="restoreHistory" />
    <CatalogImportDialog v-if="catalogImportOpen" :busy="busy" @close="catalogImportOpen=false" @import="importCatalog" />
  </div>
</template>
