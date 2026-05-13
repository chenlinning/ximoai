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
    searchPlaceholder: '搜索模型名、平台或分组...',
    allPlatforms: '全部平台',
    inputPrice: '输入价格',
    outputPrice: '输出价格',
    cacheReadPrice: '缓存读取',
    perRequestPrice: '按次计费',
    imagePrice: '图片价格',
    perMillionUnit: '/百万 Token',
    perRequestUnit: '/次',
    perImageUnit: '/张',
    noPricing: '暂无定价',
    noModels: '暂无可用模型',
    noResults: '未找到匹配的模型',
    copied: '已复制',
    copyModelName: '复制模型名',
    groupRates: '分组倍率',
    defaultRate: '默认倍率',
  },
}

const enMessages = {
  modelPlaza: {
    title: 'Model Plaza',
    subtitle: 'View model pricing and group rates',
    searchPlaceholder: 'Search model, platform or group...',
    allPlatforms: 'All Platforms',
    inputPrice: 'Input Price',
    outputPrice: 'Output Price',
    cacheReadPrice: 'Cache Read',
    perRequestPrice: 'Per Request',
    imagePrice: 'Image Price',
    perMillionUnit: '/1M Tokens',
    perRequestUnit: '/Request',
    perImageUnit: '/Image',
    noPricing: 'No pricing info',
    noModels: 'No models available',
    noResults: 'No matching models found',
    copied: 'Copied',
    copyModelName: 'Copy model name',
    groupRates: 'Group Rates',
    defaultRate: 'Default Rate',
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
    // 合并中文（源项目 locale key 是 'zh'，不是 'zh-CN'）
    i18n.global.mergeLocaleMessage('zh', zhMessages)
    // 合并英文
    i18n.global.mergeLocaleMessage('en', enMessages)
    console.log('[ModelPlaza] i18n messages injected')
  } catch (e) {
    console.warn('[ModelPlaza] i18n inject failed:', e)
  }
}
