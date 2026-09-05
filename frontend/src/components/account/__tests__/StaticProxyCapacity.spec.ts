import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import AccountCapacityCell from '../AccountCapacityCell.vue'
import CapacityBadge from '../CapacityBadge.vue'
import QuotaBadge from '../QuotaBadge.vue'
import type { Account } from '@/types'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, values?: Record<string, unknown>) =>
      values ? `${key} ${Object.values(values).join(' ')}` : key
  })
}))

function makeAccount(overrides: Partial<Account> = {}): Account {
  return {
    id: 1,
    name: 'account',
    platform: 'openai',
    type: 'oauth',
    proxy_id: null,
    concurrency: 5,
    current_concurrency: 2,
    priority: 1,
    status: 'active',
    error_message: null,
    last_used_at: null,
    expires_at: null,
    auto_pause_on_expired: true,
    created_at: '2026-03-15T00:00:00Z',
    updated_at: '2026-03-15T00:00:00Z',
    schedulable: true,
    rate_limited_at: null,
    rate_limit_reset_at: null,
    overload_until: null,
    temp_unschedulable_until: null,
    temp_unschedulable_reason: null,
    session_window_start: null,
    session_window_end: null,
    session_window_status: null,
    ...overrides
  }
}

function staticCapacityAccount(overrides: Partial<Account> = {}): Account {
  return makeAccount({
    ...overrides,
    static_ip_capacity: {
      current: 29,
      max: 100,
      exits: [
        { id: 1, name: 'Exit one', host: 'one.example.com', status: 'healthy', current: 10, max: 10 },
        { id: 2, name: 'Exit two', host: 'two.example.com', status: 'healthy', current: 10, max: 10 },
        { id: 3, name: 'Exit three', host: 'three.example.com', status: 'healthy', current: 10, max: 10 }
      ]
    }
  })
}

describe('Static proxy exit capacity', () => {
  it('renders aggregate 29/100 and one 10/10 row for each exit', () => {
    const wrapper = mount(AccountCapacityCell, {
      props: { account: staticCapacityAccount() }
    })

    expect(wrapper.get('[data-test="static-capacity-total"]').text()).toContain('29/100')
    const rows = wrapper.findAll('[data-test^="static-capacity-exit-"]')
    expect(rows).toHaveLength(3)
    expect(rows.map((row) => row.text())).toEqual(['10/10', '10/10', '10/10'])
  })

  it('keeps the existing account capacity display when static capacity is absent', () => {
    const wrapper = mount(AccountCapacityCell, {
      props: { account: makeAccount() }
    })

    expect(wrapper.find('[data-test="static-capacity-total"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('2/5')
  })

  it('static capacity replaces the concurrency badge and quota badges still apply to apikey accounts', () => {
    const wrapper = mount(AccountCapacityCell, {
      props: {
        account: staticCapacityAccount()
      }
    })

    expect(wrapper.findAllComponents(CapacityBadge)).toHaveLength(4)
    expect(wrapper.findAllComponents(QuotaBadge)).toHaveLength(0)

    const quotaWrapper = mount(AccountCapacityCell, {
      props: {
        account: staticCapacityAccount({
          type: 'apikey',
          quota_limit: 10,
          quota_used: 2
        })
      }
    })

    expect(quotaWrapper.findAllComponents(QuotaBadge)).toHaveLength(1)
    expect(quotaWrapper.text()).toContain('$2.00/$10.00')
  })

  it('marks disabled and unhealthy exits with state-specific styling and tooltips', () => {
    const account = staticCapacityAccount()
    account.static_ip_capacity!.exits = [
      { id: 7, name: 'Disabled exit', host: 'disabled.example.com', status: 'disabled', current: 0, max: 10 },
      { id: 8, name: 'Unhealthy exit', host: 'unhealthy.example.com', status: 'unhealthy', current: 9, max: 10 }
    ]

    const wrapper = mount(AccountCapacityCell, { props: { account } })
    const disabled = wrapper.get('[data-test="static-capacity-exit-7"]')
    const unhealthy = wrapper.get('[data-test="static-capacity-exit-8"]')

    expect(disabled.classes()).toContain('opacity-60')
    expect(disabled.attributes('title')).toContain('disabled.example.com')
    expect(unhealthy.classes()).toContain('bg-red-100')
    expect(unhealthy.attributes('title')).toContain('unhealthy.example.com')
  })

  it('colors idle exits green and near-full exits amber', () => {
    const account = staticCapacityAccount()
    account.static_ip_capacity!.exits = [
      { id: 9, name: 'Idle exit', host: 'idle.example.com', status: 'healthy', current: 0, max: 10 },
      { id: 10, name: 'Near full exit', host: 'near.example.com', status: 'healthy', current: 8, max: 10 },
      { id: 11, name: 'Full exit', host: 'full.example.com', status: 'healthy', current: 10, max: 10 }
    ]

    const wrapper = mount(AccountCapacityCell, { props: { account } })
    expect(wrapper.get('[data-test="static-capacity-exit-9"]').classes()).toContain('bg-emerald-100')
    expect(wrapper.get('[data-test="static-capacity-exit-10"]').classes()).toContain('bg-amber-100')
    expect(wrapper.get('[data-test="static-capacity-exit-11"]').classes()).toContain('bg-red-100')
  })
})
