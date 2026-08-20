type MessageTree = Record<string, any>

const zhPatch: MessageTree = {
  desktopAuthorization: {
    title: '\u684c\u9762\u7aef\u6388\u6743',
    authorizing: '\u6b63\u5728\u9a8c\u8bc1\u767b\u5f55\u5e76\u5efa\u7acb\u8bbe\u5907\u4f1a\u8bdd...',
    returning: '\u6388\u6743\u6210\u529f\uff0c\u6b63\u5728\u8fd4\u56de\u684c\u9762\u5e94\u7528...',
    failed: '\u65e0\u6cd5\u5b8c\u6210\u684c\u9762\u7aef\u6388\u6743\u3002',
    retry: '\u91cd\u8bd5',
  },
  nav: {
    ximoaiModelPlaza: 'XimoAI 模型广场',
  },
  modelPlaza: {
    billingModeVideo: '按视频',
    perVideoUnit: '/ 个视频',
    videoPrice: '视频价格',
  },
  ximoappUpdate: {
    hiddenInDownloadCenter: '\u4ece\u4e0b\u8f7d\u4e2d\u5fc3\u9690\u85cf',
  },
  admin: {
    settings: {
      features: {
        ximoaiModelPlazaEntry: {
          title: '定制模型广场入口',
          description: '控制定制模型广场是否显示在侧边栏。',
          enabled: '显示侧边栏入口',
          enabledHint: '关闭后只隐藏入口，不影响页面直达或接口。',
        },
      },
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
  desktopAuthorization: {
    title: 'Desktop Authorization',
    authorizing: 'Verifying your login and establishing the device session...',
    returning: 'Authorization complete. Returning to the desktop app...',
    failed: 'Desktop authorization could not be completed.',
    retry: 'Retry',
  },
  nav: {
    ximoaiModelPlaza: 'XimoAI Model Plaza',
  },
  modelPlaza: {
    billingModeVideo: 'Per Video',
    perVideoUnit: '/ video',
    videoPrice: 'Video Price',
  },
  ximoappUpdate: {
    hiddenInDownloadCenter: 'Hide from download center',
  },
  admin: {
    settings: {
      features: {
        ximoaiModelPlazaEntry: {
          title: 'Custom Model Plaza Entry',
          description: 'Control whether the custom Model Plaza appears in the sidebar.',
          enabled: 'Show sidebar entry',
          enabledHint: 'Disabling this only hides the entry; direct page and API access remain available.',
        },
      },
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
