/**
 * XimoAi 扩展模块入口
 *
 * 所有自定义扩展在此注册
 * 每个扩展必须完全独立，不修改源项目任何文件
 *
 * 注意：此模块通过 main.ts 的 dynamic import 加载，
 * 模块顶层代码会在 import 时自动执行，因此直接调用初始化函数即可。
 */

import { mountModelPlaza } from './model-plaza'

// 模块加载时自动初始化所有扩展
mountModelPlaza()
