type MessageTree = Record<string, any>

const zhPatch: MessageTree = {
  nav: {
    modelPlaza: '模型广场',
    membership: '会员中心',
    membershipManagement: '会员管理'
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
  membership: {
    levels: {
      bronze: '黄铜会员',
      silver: '白银会员',
      gold: '黄金会员',
      platinum: '铂金会员',
      diamond: '钻石会员'
    },
    title: '会员中心',
    description: '查看当前会员等级、可用分组和系统托管 API Key。',
    loading: '正在加载会员信息...',
    currentLevel: '当前会员等级',
    unassigned: '未绑定会员等级',
    defaultLevel: '默认等级',
    discountRate: '折扣倍率',
    expiresAt: '到期时间',
    longTerm: '长期有效',
    benefits: '会员权益',
    benefitsDescription: '全部会员等级的折扣倍率和专属分组。',
    current: '当前',
    exclusiveGroups: '专属分组',
    exclusiveGroupCount: '{count} 个',
    noExclusiveGroups: '暂无专属分组',
    noBenefits: '暂无可展示的会员权益。',
    availableGroups: '可用分组',
    publicGroup: '公开分组',
    groupRate: '分组倍率',
    effectiveRate: '实际结算倍率',
    noGroups: '当前会员等级没有配置可用分组。',
    managedKeys: '系统托管 Key',
    managedBadge: '会员托管',
    managedEnableBlocked: '会员托管 Key 不能由用户自行启用',
    managedDeleteBlocked: '会员托管 Key 不能由用户删除',
    enabled: '启用',
    disabled: '停用',
    groupFallback: '分组 #{id}',
    noManagedKeys: '暂无系统托管 Key。',
    loadFailed: '会员信息加载失败',
    expiresAtInline: '到期：{time}',
    disabledReasons: {
      membership_expired: '会员到期',
      membership_group_removed: '会员等级移除分组',
      membership_level_disabled: '会员等级停用',
      repair_disabled: '自动修复停用'
    }
  },
  adminMemberships: {
    title: '会员等级管理',
    description: '配置会员等级、折扣倍率、可用分组，并可按用户 ID 分配会员等级。',
    default: '默认',
    unconfigured: '未配置',
    noLevelsTitle: '暂无会员等级',
    noLevelsDescription: '系统会通过迁移创建黄铜、白银、黄金、铂金、钻石五个固定会员等级。',
    assignTitle: '分配用户会员等级',
    assignDescription: '按用户 ID 分配或调整会员等级',
    userId: '用户 ID',
    level: '会员等级',
    selectLevel: '选择会员等级',
    expiresAt: '到期时间',
    emptyExpiresAtHint: '留空表示长期有效',
    assignButton: '分配会员',
    assignmentsTitle: '已分配会员',
    assignmentsDescription: '显示当前生效的用户会员等级，便于核对分配结果。',
    userFallback: '用户 #{id}',
    levelFallback: '会员等级 #{id}',
    discount: '折扣',
    longTerm: '长期有效',
    editLevelTitle: '编辑会员等级',
    noAssignmentsTitle: '暂无已分配会员',
    noAssignmentsDescription: '为用户分配会员等级后，会在这里显示当前生效记录。',
    configureLevel: '配置会员等级',
    userRateFormula: '用户专属倍率 = 分组倍率 × 会员折扣',
    availableGroups: '可用分组',
    selectedCount: '已选 {count} 个',
    exclusive: '专属',
    public: '公开',
    noSelectableGroups: '暂无可选分组。',
    cancel: '取消',
    save: '保存',
    editUserLevel: '编辑用户会员等级',
    columns: {
      name: '会员等级',
      discountRate: '折扣倍率',
      groups: '可用分组',
      sortOrder: '排序',
      actions: '操作',
      user: '用户',
      level: '会员等级',
      source: '来源',
      startsAt: '开始时间',
      expiresAt: '到期时间'
    },
    sources: {
      system: '系统',
      admin: '管理员',
      purchase: '购买'
    },
    errors: {
      loadFailed: '会员等级加载失败',
      missingAssignInput: '请填写用户 ID 并选择会员等级',
      selectLevel: '请选择会员等级',
      saveFailed: '会员等级保存失败',
      syncFailed: '会员等级同步失败',
      assignFailed: '分配会员失败',
      updateFailed: '会员等级更新失败'
    },
    success: {
      updated: '会员等级已更新',
      synced: '会员等级已同步',
      assigned: '用户会员等级已分配',
      userUpdated: '用户会员等级已更新'
    }
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
    modelPlaza: 'Model Plaza',
    membership: 'Membership',
    membershipManagement: 'Membership Management'
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
  membership: {
    levels: {
      bronze: 'Brass Member',
      silver: 'Silver Member',
      gold: 'Gold Member',
      platinum: 'Platinum Member',
      diamond: 'Diamond Member'
    },
    title: 'Membership',
    description: 'View your current membership level, available groups, and system-managed API keys.',
    loading: 'Loading membership information...',
    currentLevel: 'Current membership level',
    unassigned: 'No membership level assigned',
    defaultLevel: 'Default level',
    discountRate: 'Discount rate',
    expiresAt: 'Expires at',
    longTerm: 'Long-term valid',
    benefits: 'Membership Benefits',
    benefitsDescription: 'Discount rates and exclusive groups for all membership levels.',
    current: 'Current',
    exclusiveGroups: 'Exclusive Groups',
    exclusiveGroupCount: '{count}',
    noExclusiveGroups: 'No exclusive groups',
    noBenefits: 'No membership benefits to display.',
    availableGroups: 'Available Groups',
    publicGroup: 'Public group',
    groupRate: 'Group rate',
    effectiveRate: 'Effective billing rate',
    noGroups: 'No available groups configured for the current membership level.',
    managedKeys: 'System-Managed Keys',
    managedBadge: 'Membership Managed',
    managedEnableBlocked: 'Membership-managed keys cannot be enabled by users',
    managedDeleteBlocked: 'Membership-managed keys cannot be deleted by users',
    enabled: 'Enabled',
    disabled: 'Disabled',
    groupFallback: 'Group #{id}',
    noManagedKeys: 'No system-managed keys.',
    loadFailed: 'Failed to load membership information',
    expiresAtInline: 'Expires: {time}',
    disabledReasons: {
      membership_expired: 'Membership expired',
      membership_group_removed: 'Group removed from membership level',
      membership_level_disabled: 'Membership level disabled',
      repair_disabled: 'Disabled by automatic repair'
    }
  },
  adminMemberships: {
    title: 'Membership Level Management',
    description: 'Configure membership levels, discount rates, and available groups; assign levels by user ID.',
    default: 'Default',
    unconfigured: 'Not configured',
    noLevelsTitle: 'No membership levels',
    noLevelsDescription: 'Migrations create five fixed levels: Brass, Silver, Gold, Platinum, and Diamond.',
    assignTitle: 'Assign User Membership',
    assignDescription: 'Assign or update a membership level by user ID',
    userId: 'User ID',
    level: 'Membership Level',
    selectLevel: 'Select membership level',
    expiresAt: 'Expires at',
    emptyExpiresAtHint: 'Leave empty for long-term validity',
    assignButton: 'Assign Membership',
    assignmentsTitle: 'Assigned Members',
    assignmentsDescription: 'Shows active user membership assignments for verification.',
    userFallback: 'User #{id}',
    levelFallback: 'Membership level #{id}',
    discount: 'Discount',
    longTerm: 'Long-term valid',
    editLevelTitle: 'Edit membership level',
    noAssignmentsTitle: 'No assigned members',
    noAssignmentsDescription: 'Active assignments will appear here after assigning membership levels to users.',
    configureLevel: 'Configure Membership Level',
    userRateFormula: 'User-specific rate = group rate × membership discount',
    availableGroups: 'Available Groups',
    selectedCount: '{count} selected',
    exclusive: 'Exclusive',
    public: 'Public',
    noSelectableGroups: 'No selectable groups.',
    cancel: 'Cancel',
    save: 'Save',
    editUserLevel: 'Edit User Membership',
    columns: {
      name: 'Membership Level',
      discountRate: 'Discount Rate',
      groups: 'Available Groups',
      sortOrder: 'Sort',
      actions: 'Actions',
      user: 'User',
      level: 'Membership Level',
      source: 'Source',
      startsAt: 'Starts At',
      expiresAt: 'Expires At'
    },
    sources: {
      system: 'System',
      admin: 'Admin',
      purchase: 'Purchase'
    },
    errors: {
      loadFailed: 'Failed to load membership levels',
      missingAssignInput: 'Enter a user ID and select a membership level',
      selectLevel: 'Select a membership level',
      saveFailed: 'Failed to save membership level',
      syncFailed: 'Failed to sync membership level',
      assignFailed: 'Failed to assign membership',
      updateFailed: 'Failed to update membership level'
    },
    success: {
      updated: 'Membership level updated',
      synced: 'Membership level synced',
      assigned: 'User membership assigned',
      userUpdated: 'User membership updated'
    }
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
