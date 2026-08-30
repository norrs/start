import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import IconRenderer from './IconRenderer.vue'

describe('IconRenderer', () => {
  it('normalizes explicit Devicon names without allowing arbitrary classes', () => {
    const wrapper = mount(IconRenderer, { props: { title:'Go', url:'https://go.dev', provider:'devicon', name:'devicon-go-plain', color:'', colored:true } })
    expect(wrapper.get('i').classes()).toContain('devicon-go-plain')
    expect(wrapper.get('i').classes()).toContain('colored')
  })

  it('chooses a self-hosted icon from the destination when configured automatically', () => {
    const wrapper = mount(IconRenderer, { props: { title:'Source', url:'https://github.com/example/start', provider:'', name:'', color:'', colored:false } })
    expect(wrapper.get('i').classes()).toContain('devicon-github-original')
  })

  it('supports Font Awesome style-qualified icon names', () => {
    const wrapper = mount(IconRenderer, { props: { title:'GitHub', url:'https://example.test', provider:'fontawesome', name:'brands:github', color:'', colored:false } })
    expect(wrapper.get('i').classes()).toEqual(expect.arrayContaining(['fa-brands','fa-github']))
  })
})
