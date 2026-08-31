import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import CatalogImportDialog from './CatalogImportDialog.vue'

describe('CatalogImportDialog',()=>{
  it('requires review confirmation before importing a YAML file',async()=>{
    const wrapper=mount(CatalogImportDialog,{props:{busy:false},attachTo:document.body})
    const file=new File(['version: 1\ncategories: []\ngroups: []\nlinks: []\n'],'links.yaml',{type:'application/yaml'})
    const input=wrapper.get('input[type="file"]')
    Object.defineProperty(input.element,'files',{value:[file]})
    await input.trigger('change')
    await vi.waitFor(()=>expect(wrapper.get('pre').text()).toContain('version: 1'))
    expect(wrapper.get('button[type="submit"]').attributes('disabled')).toBeDefined()
    await wrapper.get('input[type="checkbox"]').setValue(true)
    await wrapper.get('form').trigger('submit')
    expect(wrapper.emitted('import')?.[0]?.[0]).toBe(file)
    wrapper.unmount()
  })
})
