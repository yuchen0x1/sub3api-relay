import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'

import Toast from '../Toast.vue'
import { useAppStore } from '@/stores/app'

describe('Toast', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('renders no action button for a plain toast', async () => {
    const store = useAppStore()
    store.showToast('success', 'saved')

    mount(Toast)
    await Promise.resolve()

    expect(document.body.querySelector('[data-test="toast-action"]')).toBeNull()
  })

  it('renders the action label and dismisses the toast when invoked', async () => {
    const store = useAppStore()
    const onUndo = vi.fn()
    store.showToast('success', 'exit enabled', 6000, { label: 'Undo', onClick: onUndo })

    const wrapper = mount(Toast)
    await wrapper.vm.$nextTick()

    const button = document.body.querySelector<HTMLButtonElement>('[data-test="toast-action"]')
    expect(button).not.toBeNull()
    expect(button?.textContent?.trim()).toBe('Undo')

    button?.click()
    await wrapper.vm.$nextTick()

    expect(onUndo).toHaveBeenCalledTimes(1)
    expect(store.toasts).toHaveLength(0)
  })
})
