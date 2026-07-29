import { describe, expect, it } from 'vitest'

import { ximoAIUserNavItems } from '@/extensions/navigation'
import { FeatureFlags } from '@/utils/featureFlags'

describe('XimoAI model plaza navigation', () => {
  it('uses the upstream label for both entries and flags only the custom entry independently', () => {
    const upstreamFlag = () => true
    const customFlag = () => false
    const items = ximoAIUserNavItems({
      modelPlaza: 'upstream-icon',
      customModelPlaza: 'custom-icon',
      downloadCenter: 'download-icon',
      modelPlazaFeatureFlag: upstreamFlag,
      ximoAIModelPlazaEntryFeatureFlag: customFlag,
    })

    expect(items[0]).toMatchObject({
      path: '/model-plaza',
      labelKey: 'nav.modelPlaza',
      featureFlag: upstreamFlag,
    })
    expect(items[1]).toMatchObject({
      path: '/ximoai-model-plaza',
      labelKey: 'nav.modelPlaza',
      featureFlag: customFlag,
    })
    expect(FeatureFlags.ximoaiModelPlazaEntry).toMatchObject({
      key: 'ximoai_model_plaza_entry_enabled',
      mode: 'opt-out',
    })
  })
})
