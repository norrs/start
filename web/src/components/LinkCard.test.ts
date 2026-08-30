import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import LinkCard from './LinkCard.vue'
import type { Link, Preferences } from '../types'

const link:Link={id:1,slug:'mail',title:'Mail',url:'mailto:help@example.com',description:'Ask for help',iconProvider:'',iconName:'',iconColor:'',categoryId:1,categorySlug:'support',categoryName:'Support',categoryColor:'#fff',ownerGroupId:1,ownerGroupSlug:'platform',ownerGroupName:'Platform',tags:['help'],roles:[],teams:[],hidden:false,canManage:true,revision:1}
const preferences:Preferences={theme:'dark',viewMode:'categorized',showDescriptions:true,showCategories:true,showIcons:true,coloredIcons:true,showHidden:false,openNewTab:false,hotkeysEnabled:true,revision:1}

describe('LinkCard',()=>{
  it('renders non-HTTP destinations without throwing and exposes management controls',()=>{
    const wrapper=mount(LinkCard,{props:{link,preferences,active:true}})
    expect(wrapper.get('a').attributes('href')).toBe('mailto:help@example.com')
    expect(wrapper.text()).toContain('Ask for help')
    expect(wrapper.classes()).toContain('active')
  })
  it('emits the selected link when hidden',async()=>{
    const wrapper=mount(LinkCard,{props:{link,preferences,active:false}})
    await wrapper.get('button').trigger('click')
    expect(wrapper.emitted('hide')?.[0]).toEqual([link])
  })
})
