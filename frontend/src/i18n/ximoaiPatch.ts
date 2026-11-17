type MessageTree = Record<string, any>

const zhPatch: MessageTree = {
  "nav": {
    "ximoaiModelPlaza": "XimoAI 模型广场"
  },
  "ximoaiModelPlaza": {
    "title": "模型广场",
    "description": "按你可用的渠道定价和分组倍率展示模型价格",
    "searchPlaceholder": "搜索模型、品牌、平台、渠道或分组...",
    "allBrands": "全部品牌",
    "groupCount": "{count} 个分组",
    "billingMode": "计费模式",
    "billingModeToken": "按 Token",
    "billingModePerRequest": "按次",
    "billingModeImage": "按图片",
    "billingModeVideo": "按视频",
    "groupLabel": "分组",
    "rateValue": "x{rate}",
    "inputPrice": "输入",
    "outputPrice": "输出",
    "cacheWritePrice": "缓存写入",
    "cacheReadPrice": "缓存读取",
    "imageOutputPrice": "图片输出",
    "perRequestPrice": "每次请求",
    "intervals": "阶梯定价",
    "perMillionUnit": "/ 1M token",
    "perRequestUnit": "/ 次",
    "perImageUnit": "/ 张",
    "perVideoUnit": "/ 个视频",
    "videoPrice": "视频价格",
    "intervalUnlimited": "无限制",
    "noPricing": "未配置定价",
    "noModels": "暂无模型数据",
    "copyModelName": "复制模型名称",
    "copied": "已复制模型名称",
    "retry": "重试",
    "refresh": "刷新",
    "loadError": "加载模型广场失败",
    "brand": "品牌",
    "modelType": "模型类型",
    "allTypes": "全部类型",
    "invocationMode": "调用模式",
    "allModes": "全部模式",
    "types": {
      "conversation": "对话",
      "embedding": "向量嵌入",
      "image": "图像",
      "video": "视频",
      "tts": "语音合成",
      "asr": "语音识别"
    },
    "modes": {
      "sync": "同步",
      "stream": "流式",
      "async": "异步",
      "bidirectional": "双向流",
      "batch": "批处理"
    },
    "metadataEditor": {
      "reasoningLevels": "推理档位",
      "thinkingSupport": "思考支持",
      "thinkingEnabled": "启用思考",
      "title": "编辑 {model} 元数据",
      "edit": "编辑模型元数据",
      "brand": "显示品牌",
      "useAutomatic": "使用自动值",
      "automaticValue": "自动值",
      "resetAll": "全部恢复自动值",
      "cancel": "取消",
      "save": "保存",
      "saved": "模型元数据已保存",
      "saveError": "模型元数据保存失败",
      "resetDone": "已恢复全部自动值",
      "resetError": "恢复自动值失败"
    }
  },
  "admin": {
    "settings": {
      "features": {
        "ximoaiModelPlazaEntry": {
          "title": "定制模型广场入口",
          "description": "控制定制模型广场是否显示在侧边栏。",
          "enabled": "显示侧边栏入口",
          "enabledHint": "关闭后只隐藏入口，不影响页面直达或接口。"
        }
      }
    }
  }
}

const enPatch: MessageTree = {
  "nav": {
    "ximoaiModelPlaza": "XimoAI Model Plaza"
  },
  "ximoaiModelPlaza": {
    "title": "Model Plaza",
    "description": "Model prices calculated from your accessible channel pricing and group multipliers",
    "searchPlaceholder": "Search models, brands, platforms, channels or groups...",
    "allBrands": "All Brands",
    "groupCount": "{count} groups",
    "billingMode": "Billing Mode",
    "billingModeToken": "Per Token",
    "billingModePerRequest": "Per Request",
    "billingModeImage": "Per Image",
    "billingModeVideo": "Per Video",
    "groupLabel": "Group",
    "rateValue": "x{rate}",
    "inputPrice": "Input",
    "outputPrice": "Output",
    "cacheWritePrice": "Cache Write",
    "cacheReadPrice": "Cache Read",
    "imageOutputPrice": "Image Output",
    "perRequestPrice": "Per Request",
    "intervals": "Tiered Pricing",
    "perMillionUnit": "/ 1M tokens",
    "perRequestUnit": "/ request",
    "perImageUnit": "/ image",
    "perVideoUnit": "/ video",
    "videoPrice": "Video Price",
    "intervalUnlimited": "unlimited",
    "noPricing": "Pricing not configured",
    "noModels": "No model data",
    "copyModelName": "Copy model name",
    "copied": "Model name copied",
    "retry": "Retry",
    "refresh": "Refresh",
    "loadError": "Failed to load model plaza",
    "brand": "Brand",
    "modelType": "Model type",
    "allTypes": "All types",
    "invocationMode": "Invocation mode",
    "allModes": "All modes",
    "types": {
      "conversation": "Conversation",
      "embedding": "Embedding",
      "image": "Image",
      "video": "Video",
      "tts": "Text to Speech",
      "asr": "Speech Recognition"
    },
    "modes": {
      "sync": "Synchronous",
      "stream": "Streaming",
      "async": "Asynchronous",
      "bidirectional": "Bidirectional",
      "batch": "Batch"
    },
    "metadataEditor": {
      "title": "Edit {model} metadata",
      "edit": "Edit model metadata",
      "brand": "Display brand",
      "useAutomatic": "Use automatic value",
      "automaticValue": "Automatic value",
      "resetAll": "Restore all automatic values",
      "cancel": "Cancel",
      "save": "Save",
      "saved": "Model metadata saved",
      "saveError": "Failed to save model metadata",
      "resetDone": "All automatic values restored",
      "resetError": "Failed to restore automatic values",
      "reasoningLevels": "Reasoning levels",
      "thinkingSupport": "Thinking support",
      "thinkingEnabled": "Thinking enabled"
    }
  },
  "admin": {
    "settings": {
      "features": {
        "ximoaiModelPlazaEntry": {
          "title": "Custom Model Plaza Entry",
          "description": "Control whether the custom Model Plaza appears in the sidebar.",
          "enabled": "Show sidebar entry",
          "enabledHint": "Disabling this only hides the entry; direct page and API access remain available."
        }
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
