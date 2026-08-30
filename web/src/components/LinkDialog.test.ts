import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import LinkDialog from './LinkDialog.vue'

describe('LinkDialog',()=>{
  it('previews an icon while its provider and name are edited',async()=>{
    const wrapper=mount(LinkDialog,{props:{categories:[{id:1,slug:'tools',name:'Tools',description:'',color:'#fff',sortOrder:1}],groups:[{id:1,slug:'platform',name:'Platform',description:'',canCreate:true}],busy:false}})
    await wrapper.get('input[placeholder="go-plain, server, home"]').setValue('go-plain')
    const selects=wrapper.findAll('select')
    await selects[2].setValue('devicon')
    expect(wrapper.get('.icon-preview i').classes()).toContain('devicon-go-plain')
  })
})
