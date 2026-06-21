export interface XimoAINavItem {
  path: string
  labelKey: string
  icon: unknown
  hideInSimpleMode?: boolean
}

export function ximoAIUserNavItems(icons: { modelPlaza: unknown; downloadCenter: unknown }): XimoAINavItem[] {
  return [
    {
      path: '/model-plaza',
      labelKey: 'nav.modelPlaza',
      icon: icons.modelPlaza,
    },
    {
      path: '/download-center',
      labelKey: 'downloadCenter.nav',
      icon: icons.downloadCenter,
    },
  ]
}

export function ximoAIAdminNavItems(icons: { platform: unknown; application: unknown }): XimoAINavItem[] {
  return [
    {
      path: '/admin/platforms',
      labelKey: 'nav.platforms',
      icon: icons.platform,
    },
    {
      path: '/admin/ximoapp-update',
      labelKey: 'ximoappUpdate.nav',
      icon: icons.application,
      hideInSimpleMode: true,
    },
  ]
}
