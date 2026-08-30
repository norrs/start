<script setup lang="ts">
import { ref } from 'vue'
import BaseDialog from './BaseDialog.vue'

defineProps<{busy:boolean}>()
const emit=defineEmits<{close:[];import:[file:File]}>()
const file=ref<File>()
const preview=ref('')
const confirmed=ref(false)
const error=ref('')

function readFile(selected:File){
  return new Promise<string>((resolve,reject)=>{
    const reader=new FileReader()
    reader.onload=()=>resolve(String(reader.result??''))
    reader.onerror=()=>reject(reader.error??new Error('Could not read file'))
    reader.readAsText(selected)
  })
}

async function selectFile(event:Event){
  const selected=(event.target as HTMLInputElement).files?.[0]
  file.value=undefined;preview.value='';confirmed.value=false;error.value=''
  if(!selected)return
  if(selected.size>1024*1024){error.value='Catalog YAML must be 1 MiB or smaller';return}
  if(!/\.ya?ml$/i.test(selected.name)){error.value='Choose a .yaml or .yml file';return}
  file.value=selected
  try{preview.value=await readFile(selected)}
  catch{error.value='Could not read the selected file';file.value=undefined;return}
  if(!preview.value.trim()){error.value='Catalog YAML is empty';file.value=undefined}
}

function submit(){if(file.value&&confirmed.value)emit('import',file.value)}
</script>

<template>
  <BaseDialog title="Import link catalog" wide @close="$emit('close')">
    <form class="catalog-import" @submit.prevent="submit">
      <p>Merge a <code>links.yaml</code> catalog into the current database. Matching slugs are updated, deleted matching links are restored, and omitted links are unchanged.</p>
      <label>Catalog YAML
        <input type="file" accept=".yaml,.yml,application/yaml,text/yaml" :disabled="busy" @change="selectFile">
      </label>
      <p v-if="error" class="danger-text" role="alert">{{ error }}</p>
      <template v-if="file">
        <div class="catalog-file-summary"><strong>{{ file.name }}</strong><span>{{ Math.ceil(file.size/1024) }} KiB</span></div>
        <details open>
          <summary>Review YAML before importing</summary>
          <pre>{{ preview }}</pre>
        </details>
        <label class="catalog-confirm"><input v-model="confirmed" type="checkbox" :disabled="busy"> I understand that matching catalog records will be changed and recorded in link history.</label>
      </template>
      <footer>
        <button type="button" class="secondary" :disabled="busy" @click="$emit('close')">Cancel</button>
        <button type="submit" :disabled="busy||!file||!confirmed">{{ busy?'Importing…':'Import catalog' }}</button>
      </footer>
    </form>
  </BaseDialog>
</template>
