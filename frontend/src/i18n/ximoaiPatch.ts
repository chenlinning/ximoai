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
  common: {
    apply: '应用',
    clear: '清除',
    creating: '创建中...',
    login: '登录',
    required: '必填',
    sending: '发送中...',
    tryAgain: '请重试'
  },
  admin: {
    accounts: {
      fillRelatedModels: '同步最新支持模型',
      fromModel: '源模型',
      toModel: '目标模型',
      bulkActions: {
        batchTest: '批量测试'
      },
      oauth: {
        openai: {
          mobileRefreshTokenAuth: '手动输入 Mobile RT',
          accessTokenAuth: '手动输入 AT'
        }
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
    },
    channels: {
      noGroupsSelected: '{platform} 平台未选择分组，请至少选择一个分组或禁用该平台',
      emptyModelsInPricing: '{platform} 平台下有定价条目未添加模型，请添加模型或删除该条目'
    },
    groups: {
      failedToSave: '保存分组失败'
    },
    users: {
      passwordCopied: '密码已复制'
    },
    ops: {
      result: '结果',
      timeRange: {
        custom: '自定义'
      },
      customTimeRange: {
        startTime: '开始时间',
        endTime: '结束时间'
      },
      runtime: {
        metricThresholds: '指标阈值配置',
        metricThresholdsHint: '配置各项指标的告警阈值，超出阈值时将以红色显示',
        slaMinPercent: 'SLA 最低百分比',
        slaMinPercentHint: 'SLA 低于此值时显示为红色（默认：99.5%）',
        ttftP99MaxMs: 'TTFT P99 最大值（毫秒）',
        ttftP99MaxMsHint: 'TTFT P99 高于此值时显示为红色（默认：500ms）',
        requestErrorRateMaxPercent: '请求错误率最大值（%）',
        requestErrorRateMaxPercentHint: '请求错误率高于此值时显示为红色（默认：5%）',
        upstreamErrorRateMaxPercent: '上游错误率最大值（%）',
        upstreamErrorRateMaxPercentHint: '上游错误率高于此值时显示为红色（默认：5%）'
      }
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
  common: {
    apply: 'Apply',
    clear: 'Clear',
    creating: 'Creating...',
    login: 'Login',
    required: 'required',
    sending: 'Sending...',
    tryAgain: 'Please try again'
  },
  admin: {
    accounts: {
      fillRelatedModels: 'Sync latest supported models',
      fromModel: 'From model',
      toModel: 'To model',
      bulkActions: {
        batchTest: 'Batch test'
      },
      oauth: {
        openai: {
          mobileRefreshTokenAuth: 'Enter Mobile RT manually',
          accessTokenAuth: 'Enter AT manually'
        }
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
    },
    channels: {
      noGroupsSelected: '{platform} has no groups selected. Select at least one group or disable this platform.',
      emptyModelsInPricing: '{platform} has a pricing entry without models. Add models or remove the entry.'
    },
    groups: {
      failedToSave: 'Failed to save group'
    },
    users: {
      passwordCopied: 'Password copied'
    },
    ops: {
      result: 'Result',
      timeRange: {
        custom: 'Custom'
      },
      customTimeRange: {
        startTime: 'Start time',
        endTime: 'End time'
      },
      runtime: {
        metricThresholds: 'Metric Thresholds',
        metricThresholdsHint: 'Configure alert thresholds for metrics. Values exceeding thresholds are shown in red.',
        slaMinPercent: 'SLA Minimum Percentage',
        slaMinPercentHint: 'SLA below this value is shown in red (default: 99.5%).',
        ttftP99MaxMs: 'TTFT P99 Maximum (ms)',
        ttftP99MaxMsHint: 'TTFT P99 above this value is shown in red (default: 500ms).',
        requestErrorRateMaxPercent: 'Request Error Rate Maximum (%)',
        requestErrorRateMaxPercentHint: 'Request error rate above this value is shown in red (default: 5%).',
        upstreamErrorRateMaxPercent: 'Upstream Error Rate Maximum (%)',
        upstreamErrorRateMaxPercentHint: 'Upstream error rate above this value is shown in red (default: 5%).'
      }
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
