export interface XimoAINavItem {
  path: string
  labelKey: string
  icon: unknown
  hideInSimpleMode?: boolean
}

export function ximoAIUserNavItems(icons: { modelPlaza: unknown }): XimoAINavItem[] {
  return [
    {
      path: '/model-plaza',
      labelKey: 'nav.modelPlaza',
      icon: icons.modelPlaza,
    },
  ]
}

export function ximoAIAdminNavItems(icons: { platform: unknown }): XimoAINavItem[] {
  return [
    {
      path: '/admin/platforms',
      labelKey: 'nav.platforms',
      icon: icons.platform,
    },
  ]
}
