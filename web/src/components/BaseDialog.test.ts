import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { describe, expect, it } from 'vitest'
import BaseDialog from './BaseDialog.vue'

describe('BaseDialog', () => {
  it('focuses the first form field instead of the close button', async () => {
    const wrapper=mount(BaseDialog, {
      attachTo:document.body,
      props:{title:'Edit link'},
      slots:{default:'<input aria-label="First field"><button type="button">Last action</button>'},
    })
    await nextTick()
    expect(document.activeElement).toBe(wrapper.get('input').element)
    wrapper.unmount()
  })
})
