type MessageTree = Record<string, any>

const zhPatch: MessageTree = {
  nav: {
    platforms: '平台管理',
  },
  modelPlaza: {
    billingModeVideo: '按视频',
    perVideoUnit: '/ 个视频',
    videoPrice: '视频价格',
  },
  admin: {
    platforms: {
      title: '平台管理',
      description: '管理自定义上游平台，并选择对应的上游协议类型',
      create: '添加平台',
      edit: '编辑平台',
      slug: '平台标识',
      slugHint: '小写字母、数字、横线或下划线，例如 openrouter',
      displayName: '显示名称',
      baseUrl: 'Base URL',
      color: '颜色',
      enabled: '启用',
      builtin: '内置',
      custom: '自定义',
      protocol: '协议',
      capabilities: '能力',
      noPlatforms: '暂无平台',
      createSuccess: '平台已创建',
      updateSuccess: '平台已更新',
      deleteSuccess: '平台已删除',
      failedToLoad: '加载平台失败',
      failedToSave: '保存平台失败',
      failedToDelete: '删除平台失败',
      deleteConfirm: '确定删除平台“{name}”吗？如果已有分组或账号使用该平台，后端会拒绝删除。',
      readOnlyBuiltin: '核心内置平台会锁定协议和 Base URL；平台标识、显示名称、颜色和启用状态可以编辑。',
      openAICompatibleOnly: '自定义平台可选择 OpenAI-compatible、Anthropic 或 Gemini 协议；所有自定义平台都仅支持 API Key，OAuth、Codex、WebSocket 等官方能力仍只保留给官方平台。',
      apiKeyOnly: '自定义平台仅支持 API Key。',
      protocolOpenAICompatible: 'OpenAI-compatible',
      protocolAnthropic: 'Anthropic',
      protocolGemini: 'Gemini',
      actions: '操作',
    },
    channels: {
      billingMode: {
        video: '视频（按次）',
      },
      form: {
        defaultVideoPrice: '默认视频价格（未命中层级时使用）',
        videoTiers: '视频计费层级（按次）',
        durationOrQuality: '规格',
      },
    },
    usage: {
      billingModeVideo: '按次（视频）',
    },
  },
  usage: {
    videoUnit: '个视频',
    videoCount: '视频数量',
    videoUnitPrice: '单视频价格',
    videoTotalPrice: '视频费用',
  },
  availableChannels: {
    pricing: {
      billingModeVideo: '按视频',
    },
  },
}

const enPatch: MessageTree = {
  nav: {
    platforms: 'Platforms',
  },
  modelPlaza: {
    billingModeVideo: 'Per Video',
    perVideoUnit: '/ video',
    videoPrice: 'Video Price',
  },
  admin: {
    platforms: {
      title: 'Platform Management',
      description: 'Manage custom upstream platforms and choose their upstream protocol',
      create: 'Add Platform',
      edit: 'Edit Platform',
      slug: 'Slug',
      slugHint: 'Lowercase letters, numbers, hyphen, or underscore, such as openrouter',
      displayName: 'Display Name',
      baseUrl: 'Base URL',
      color: 'Color',
      enabled: 'Enabled',
      builtin: 'Built-in',
      custom: 'Custom',
      protocol: 'Protocol',
      capabilities: 'Capabilities',
      noPlatforms: 'No platforms',
      createSuccess: 'Platform created',
      updateSuccess: 'Platform updated',
      deleteSuccess: 'Platform deleted',
      failedToLoad: 'Failed to load platforms',
      failedToSave: 'Failed to save platform',
      failedToDelete: 'Failed to delete platform',
      deleteConfirm: 'Delete platform "{name}"? The backend will reject deletion if groups or accounts still use it.',
      readOnlyBuiltin: 'Core built-in platforms keep their protocol and Base URL locked. Slug, display name, color, and enabled state can be edited.',
      openAICompatibleOnly: 'Custom platforms can use OpenAI-compatible, Anthropic, or Gemini protocols. All custom platforms support API Key only; OAuth, Codex, and WebSocket features remain reserved for official platforms.',
      apiKeyOnly: 'Custom platforms support API Key only.',
      protocolOpenAICompatible: 'OpenAI-compatible',
      protocolAnthropic: 'Anthropic',
      protocolGemini: 'Gemini',
      actions: 'Actions',
    },
    channels: {
      billingMode: {
        video: 'Video (per request)',
      },
      form: {
        defaultVideoPrice: 'Default video price (used when no tier matches)',
        videoTiers: 'Video billing tiers (per request)',
        durationOrQuality: 'Spec',
      },
    },
    usage: {
      billingModeVideo: 'Video',
    },
  },
  usage: {
    videoUnit: 'videos',
    videoCount: 'Video Count',
    videoUnitPrice: 'Price per Video',
    videoTotalPrice: 'Video Cost',
  },
  availableChannels: {
    pricing: {
      billingModeVideo: 'Per Video',
    },
  },
}

function mergeMessages(target: MessageTree, patch: MessageTree): MessageTree {
  for (const [key, value] of Object.entries(patch)) {
    if (value && typeof value === 'object' && !Array.isArray(value)) {
      const existing = target[key]
      target[key] = mergeMessages(
        existing && typeof existing === 'object' && !Array.isArray(existing) ? existing : {},
        value,
      )
    } else {
      target[key] = value
    }
  }
  return target
}

export function applyPlatformI18nPatch(locale: 'zh' | 'en', messages: MessageTree): MessageTree {
  return mergeMessages(messages, locale === 'zh' ? zhPatch : enPatch)
}
