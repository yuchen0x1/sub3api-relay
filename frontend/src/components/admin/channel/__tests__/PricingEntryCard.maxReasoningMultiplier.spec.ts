import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import PricingEntryCard from '../PricingEntryCard.vue'
import type { PricingFormEntry } from '../types'

vi.mock('vue-i18n', async importOriginal => ({
  ...await importOriginal<typeof import('vue-i18n')>(),
  useI18n: () => ({ t: (key: string) => key }),
}))

function createEntry(billingMode: PricingFormEntry['billing_mode'] = 'token'): PricingFormEntry {
  return {
    models: [],
    billing_mode: billingMode,
    input_price: null,
    output_price: null,
    cache_write_price: null,
    cache_read_price: null,
    image_input_price: null,
    image_output_price: null,
    per_request_price: null,
    max_reasoning_effort_multiplier: null,
    intervals: [],
  }
}

describe('PricingEntryCard max reasoning effort multiplier', () => {
  it('emits the typed multiplier for token pricing', async () => {
    const entry = createEntry('token')
    const wrapper = mount(PricingEntryCard, {
      props: { entry },
      attachTo: document.body,
    })

    const input = wrapper.find('input[placeholder="admin.channels.form.multiplierPlaceholder"]')
    expect(input.exists()).toBe(true)
    await input.setValue('2.5')

    const emitted = wrapper.emitted('update')
    expect(emitted).toHaveLength(1)
    expect((emitted?.[0]?.[0] as PricingFormEntry).max_reasoning_effort_multiplier).toBe('2.5')
    wrapper.unmount()
  })

  it('shows the fable default placeholder when a fable model is priced', () => {
    const entry = createEntry('token')
    entry.models = ['claude-fable-5-1']
    const wrapper = mount(PricingEntryCard, {
      props: { entry },
      attachTo: document.body,
    })

    expect(wrapper.find('input[placeholder="admin.channels.form.fable51DefaultMaxReasoningMultiplier"]').exists()).toBe(true)
    wrapper.unmount()
  })

  it('does not render the control for per-request pricing', () => {
    const wrapper = mount(PricingEntryCard, {
      props: { entry: createEntry('per_request') },
      attachTo: document.body,
    })

    expect(wrapper.find('input[placeholder="admin.channels.form.multiplierPlaceholder"]').exists()).toBe(false)
    wrapper.unmount()
  })
})
