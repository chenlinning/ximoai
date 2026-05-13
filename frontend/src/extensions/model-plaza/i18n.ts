/**
 * 模型广场 - 独立 i18n
 *
 * 将模型广场翻译合并到源项目 i18n 实例
 * 通过 window.__APP_I18N__ 获取源项目 i18n 实例
 */

const zhMessages = {
  modelPlaza: {
    title: '模型广场',
    subtitle: '查看模型定价与分组倍率',
    search: '搜索模型名、平台或分组...',
    filterAll: '全部平台',
    inputPrice: '输入价格',
    outputPrice: '输出价格',
    cachePrice: '缓存价格',
    rateMultiplier: '倍率',
    unit: '/1M tokens',
    noPricing: '暂无定价',
    noModels: '暂无可用模型',
    copied: '已复制',
    copy: '复制模型名',
    groupRates: '分组倍率',
    defaultRate: '默认倍率',
    activeChannels: '个渠道',
    totalModels: '个模型',
  },
}

const enMessages = {
  modelPlaza: {
    title: 'Model Plaza',
    subtitle: 'View model pricing and group rates',
    search: 'Search model, platform or group...',
    filterAll: 'All Platforms',
    inputPrice: 'Input Price',
    outputPrice: 'Output Price',
    cachePrice: 'Cache Price',
    rateMultiplier: 'Rate',
    unit: '/1M tokens',
    noPricing: 'No pricing',
    noModels: 'No models available',
    copied: 'Copied',
    copy: 'Copy model name',
    groupRates: 'Group Rates',
    defaultRate: 'Default Rate',
    activeChannels: 'channels',
    totalModels: 'models',
  },
}

export type ModelPlazaMessages = typeof zhMessages

/** 将模型广场 i18n 合并到源项目 i18n 实例 */
export function injectModelPlazaI18n(): void {
  const i18n = (window as any).__APP_I18N__
  if (!i18n) {
    console.warn('[ModelPlaza] i18n instance not found')
    return
  }

  try {
    // 合并中文
    i18n.global.mergeLocaleMessage('zh-CN', zhMessages)
    // 合并英文
    i18n.global.mergeLocaleMessage('en', enMessages)
    console.log('[ModelPlaza] i18n messages injected')
  } catch (e) {
    console.warn('[ModelPlaza] i18n inject failed:', e)
  }
}
