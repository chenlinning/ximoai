/**
 * XimoAi 扩展模块入口
 *
 * 所有自定义扩展在此注册
 * 每个扩展必须完全独立，不修改源项目任何文件
 */

import { mountModelPlaza } from './model-plaza'

export function initExtensions(): void {
  mountModelPlaza()
}
