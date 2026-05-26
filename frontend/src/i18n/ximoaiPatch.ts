type MessageTree = Record<string, any>

const zhPatch: MessageTree = {
  nav: {
    modelPlaza: '模型广场'
  },
  modelPlaza: {
    title: '模型广场',
    description: '按你可用的渠道定价和分组倍率展示模型价格',
    searchPlaceholder: '搜索模型、平台、渠道或分组...',
    allPlatforms: '全部平台',
    groupCount: '{count} 个分组',
    billingMode: '计费模式',
    billingModeToken: '按 Token',
    billingModePerRequest: '按次',
    billingModeImage: '按图片',
    groupLabel: '分组',
    rateValue: 'x{rate}',
    inputPrice: '输入',
    outputPrice: '输出',
    cacheWritePrice: '缓存写入',
    cacheReadPrice: '缓存读取',
    imageOutputPrice: '图片输出',
    perRequestPrice: '每次请求',
    intervals: '阶梯定价',
    perMillionUnit: '/ 1M token',
    perRequestUnit: '/ 次',
    perImageUnit: '/ 张',
    intervalUnlimited: '无限制',
    noPricing: '未配置定价',
    noModels: '暂无模型数据',
    copyModelName: '复制模型名称',
    copied: '已复制模型名称',
    retry: '重试',
    refresh: '刷新',
    loadError: '加载模型广场失败'
  },
  admin: {
    accounts: {
      fillRelatedModels: '同步最新支持模型',
      bulkActions: {
        batchTest: '批量测试'
      },
      batchTest: {
        title: '批量测试账号连接',
        selectedCount: '已选择 {count} 个账号，所有账号将使用同一个模型测试。',
        start: '开始批量测试',
        waiting: '等待测试',
        notStarted: '未开始',
        success: '成功',
        failed: '失败',
        summary: '测试结果',
        successCount: '成功 {count}',
        failedCount: '失败 {count}',
        accountId: '账号 #{id}',
        loadingAccount: '正在加载账号信息'
      },
      sendingAudioRequest: '发送语音测试请求...',
      sendingVideoRequest: '发送视频测试请求...',
      testFormat: '测试格式',
      testFormatAuto: '自动识别',
      testFormatText: '文本',
      testFormatImage: '图片',
      testFormatAudio: '语音',
      testFormatVideo: '视频',
      audioInputLabel: '语音输入文本',
      audioInputPlaceholder: '例如：hi',
      audioTestMode: '模式：语音测试',
      audioPreview: '语音结果',
      audioReceived: '已收到第 {count} 段测试音频',
      videoPromptLabel: '视频提示词',
      videoPromptPlaceholder: '例如：A tiny test video of a sunrise over mountains.',
      videoSecondsLabel: '视频秒数',
      videoSizeLabel: '视频尺寸',
      videoTestMode: '模式：视频测试',
      videoPreview: '视频结果',
      videoReceived: '已收到第 {count} 段测试视频',
      openVideoResult: '打开视频'
    }
  }
}

const enPatch: MessageTree = {
  nav: {
    modelPlaza: 'Model Plaza'
  },
  modelPlaza: {
    title: 'Model Plaza',
    description: 'Model prices calculated from your accessible channel pricing and group multipliers',
    searchPlaceholder: 'Search models, platforms, channels or groups...',
    allPlatforms: 'All Platforms',
    groupCount: '{count} groups',
    billingMode: 'Billing Mode',
    billingModeToken: 'Per Token',
    billingModePerRequest: 'Per Request',
    billingModeImage: 'Per Image',
    groupLabel: 'Group',
    rateValue: 'x{rate}',
    inputPrice: 'Input',
    outputPrice: 'Output',
    cacheWritePrice: 'Cache Write',
    cacheReadPrice: 'Cache Read',
    imageOutputPrice: 'Image Output',
    perRequestPrice: 'Per Request',
    intervals: 'Tiered Pricing',
    perMillionUnit: '/ 1M tokens',
    perRequestUnit: '/ request',
    perImageUnit: '/ image',
    intervalUnlimited: 'unlimited',
    noPricing: 'Pricing not configured',
    noModels: 'No model data',
    copyModelName: 'Copy model name',
    copied: 'Model name copied',
    retry: 'Retry',
    refresh: 'Refresh',
    loadError: 'Failed to load model plaza'
  },
  admin: {
    accounts: {
      fillRelatedModels: 'Sync latest supported models',
      bulkActions: {
        batchTest: 'Batch test'
      },
      batchTest: {
        title: 'Batch Test Account Connections',
        selectedCount: '{count} accounts selected. All selected accounts will use the same model.',
        start: 'Start Batch Test',
        waiting: 'Waiting',
        notStarted: 'Not started',
        success: 'Success',
        failed: 'Failed',
        summary: 'Test result',
        successCount: 'Success {count}',
        failedCount: 'Failed {count}',
        accountId: 'Account #{id}',
        loadingAccount: 'Loading account details'
      },
      sendingAudioRequest: 'Sending audio speech test request...',
      sendingVideoRequest: 'Sending video generation test request...',
      testFormat: 'Test format',
      testFormatAuto: 'Auto detect',
      testFormatText: 'Text',
      testFormatImage: 'Image',
      testFormatAudio: 'Audio',
      testFormatVideo: 'Video',
      audioInputLabel: 'Audio input text',
      audioInputPlaceholder: 'Example: hi',
      audioTestMode: 'Mode: Audio speech test',
      audioPreview: 'Generated audio:',
      audioReceived: 'Received test audio #{count}',
      videoPromptLabel: 'Video prompt',
      videoPromptPlaceholder: 'Example: A tiny test video of a sunrise over mountains.',
      videoSecondsLabel: 'Video seconds',
      videoSizeLabel: 'Video size',
      videoTestMode: 'Mode: Video generation test',
      videoPreview: 'Generated video:',
      videoReceived: 'Received test video #{count}',
      openVideoResult: 'Open video'
    }
  }
}

function mergeMessages(target: MessageTree, patch: MessageTree): MessageTree {
  for (const [key, value] of Object.entries(patch)) {
    if (value && typeof value === 'object' && !Array.isArray(value)) {
      const existing = target[key]
      target[key] = mergeMessages(
        existing && typeof existing === 'object' && !Array.isArray(existing) ? existing : {},
        value
      )
    } else {
      target[key] = value
    }
  }
  return target
}

export function applyXimoAII18nPatch(locale: 'zh' | 'en', messages: MessageTree): MessageTree {
  return mergeMessages(messages, locale === 'zh' ? zhPatch : enPatch)
}
