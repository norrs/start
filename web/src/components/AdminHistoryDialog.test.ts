import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import AdminHistoryDialog from './AdminHistoryDialog.vue'
import type { LinkHistoryEntry, LinkInput } from '../types'

const input=(slug:string,title:string):LinkInput=>({slug,title,url:`https://${slug}.example`,description:'',iconProvider:'',iconName:'',iconColor:'',categoryId:1,ownerGroupId:1,tags:[],roles:[],teams:[],enabled:true,revision:1})
const entry=(linkId:number,title:string):LinkHistoryEntry=>({linkId,linkSlug:title.toLowerCase(),linkTitle:title,currentRevision:2,currentDeleted:false,revision:1,action:'create',snapshot:{input:input(title.toLowerCase(),title),deleted:false},actorName:'Admin',actorEmail:'admin@example.test',createdAt:'2026-08-30T00:00:00Z'})

describe('AdminHistoryDialog',()=>{
  it('selects independent snapshots and emits a selective restore',async()=>{
    const wrapper=mount(AdminHistoryDialog,{props:{entries:[entry(1,'One'),entry(2,'Two')],loading:false,busy:false}})
    const checks=wrapper.findAll('input[type="checkbox"]')
    await checks[0].setValue(true)
    await checks[1].setValue(true)
    const restore=wrapper.findAll('button').find(button=>button.text()==='Restore 2')
    expect(restore).toBeTruthy()
    await restore!.trigger('click')
    expect(wrapper.emitted('restore')?.[0]).toEqual([[
      {linkId:1,revision:1,currentRevision:2},
      {linkId:2,revision:1,currentRevision:2},
    ]])
  })
})
