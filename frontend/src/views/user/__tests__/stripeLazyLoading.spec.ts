import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const frontendRoot = resolve(__dirname, '../../../..')
const stripeConsumers = [
  'src/views/user/StripePaymentView.vue',
  'src/views/user/StripePopupView.vue',
  'src/components/payment/StripePaymentInline.vue',
]

function readFrontendFile(path: string): string {
  return readFileSync(resolve(frontendRoot, path), 'utf8')
}

describe('Stripe lazy-loading contract', () => {
  it.each(stripeConsumers)('%s uses the side-effect-free Stripe loader', (path) => {
    const source = readFrontendFile(path)

    expect(source).toContain("await import('@stripe/stripe-js/pure')")
    expect(source).not.toMatch(/await import\(['"]@stripe\/stripe-js['"]\)/)
  })

  it('keeps Stripe out of hand-written vendor buckets', () => {
    // The hand-written vendor chunks were removed (they caused a circular
    // vendor-misc -> api/client dependency); Stripe stays out of the entry
    // chunk solely via the side-effect-free dynamic imports asserted above.
    // If the config ever pins Stripe into a shared chunk again, lazy loading
    // would silently regress, so guard against any Stripe reference here.
    const viteConfig = readFrontendFile('vite.config.ts')

    expect(viteConfig).not.toContain('@stripe/stripe-js')
  })
})
