# 2026-11-18 上游合并与定制保留记录

## 合并范围

- 合并前本地：`3e82442350a82055edae771cf3bd0b58ab8b6dad`，`main`。
- 原上游基线：`15871f904056f548dd874a1d069ffc6cdf777b78`，版本 `0.2.22`。
- 本次上游：`2af3f1e15c98c893a89255a037f551046883b8c0`，版本 `0.2.23`。
- 完整合入 32 个提交，含合并提交；净变化涉及 90 个文件。
- 合并前比较上游变更文件与本地定制文件，无直接文件交集。
- 上游的新增功能、修复、优化、测试及迁移整体合并，没有挑选部分提交。

## 上游变化及适配

| 上游变化 | 合并处理 |
| --- | --- |
| GPT-6.2 Sol/Astra 的模型清单、别名、参数处理、Codex 描述及价格 | 保留上游实现；独立模型广场补充两个型号的参考元数据 |
| GPT-6.2 Ultra Fast 套餐权限和渠道倍率 | 原样保留上游账户权限判断、管理表单、价格解析与计费 |
| Anthropic 缓存创建总量、5m/1h TTL 与缓存读取语义 | 原样保留协议转换、流式用量合并及最终计费修复 |
| 缓存未分类用量及 TTL 覆盖计费 | 原样保留上游修复与对应测试 |
| OpenAI 流式空闲超时及首输出前故障切换预算 | 原样保留；已有输出后不重放请求 |
| 原生 Chat Completions flush 时机优化 | 原样保留上游实现 |
| DeepSeek/Ollama reasoning 请求体单遍改写 | 原样保留上游实现 |
| Go 工具链及 follow-redirects 安全补丁提交 | 合并完整历史；已在当前树具备的依赖修复不重复改写 |

没有发现本次上游新增实现可以等价替换现有主题、独立模型广场或页面体积优化。
网关、账号调度、计费、协议转换、新迁移及渠道管理的本次变更文件与目标上游保持一致。

## 保留的定制与优化

- 主题配色、按钮与图表颜色、登录动态背景。
- 原生模型广场继续保持上游入口和行为；定制模型广场保持独立入口和设置。
- 定制模型广场的品牌、类型、调用模式筛选、管理员覆盖及公开字段白名单。
- 继续使用渠道实际模型名识别、用户请求别名展示。公开模型条目不增加上游名称、URL、字段或账号信息。
- 普通用户页面不展示推理/思考控件；管理员编辑和接口元数据继续有效。
- 第三方资源分包、支付 SDK 延迟加载、表格导出库按需加载。
- Base64 Logo 在 HTML 中仅保留一份，前端继续设置自定义 favicon。
- 缺失 `/assets/` 资源及 `/favicon.ico` 返回 404；含点号的 SPA 路由继续工作。
- 带哈希资源的长期缓存、HTML 的 no-store 及现有流式响应行为。
- GitHub SHA 镜像工作流保留，版本号跟随上游 `0.2.23`。

GPT-6.2 的新参考记录来自本次上游附带的描述与
`backend/internal/service/openai_gpt62_test.go`：
HTTP 档位为 `low / medium / high / xhigh / max`，默认 `low`，
不标记为可关闭思考。Codex 描述中的客户端自动委派模式 `ultra`
没有被推断为额外 HTTP 推理档位。
管理员自定义值继续优先，旧型号记录未批量改写。

## 生产部署优化保留

本次仅检查生产配置，没有执行部署或重启服务。
当时在线镜像仍为 `sha-3e82442350a82055edae771cf3bd0b58ab8b6dad`，
Sub2API 健康检查正常，内存上限 `3221225472` 字节。

后续部署应保留：

- 仅更新 Sub2API 容器，生产服务器不构建镜像。
- PostgreSQL、Redis、Magpie、Dola、s-ui 保持各自运行配置。
- Nginx gzip 级别 5、最小 1024 字节，流式 SSE 不加入压缩类型。
- `/assets/` 响应缓冲与后端提供的哈希缓存头，不重复追加缓存头。
- HTML 和 API 不配置长期缓存。
- `/var/log/nginx/ximoai-timing.log` 中的总耗时、上游耗时及请求 ID。
- 3 GiB 内存上限、既有环境变量、挂载、端口和网络配置。

检查时的配置 SHA-256：

| 文件 | SHA-256 |
| --- | --- |
| `/etc/nginx/nginx.conf` | `43348259c2fe4ba3b3cc99680c25042352cf7158785014aad5d280cf86c1f4cc` |
| `/etc/nginx/conf.d/ximoai-http.conf` | `e39b24d4449fdbc61e1b16b0db2e1f5ac9fbbd845707f64189ca7bc14e0c6f98` |
| `/etc/nginx/conf.d/00-ximoai-timing.conf` | `1f45b2b7656dc81949c4732c79411fe891e5b95b2222fb3725cf4551ff33aa7d` |
| `/etc/nginx/snippets/ximoai-static-cache.conf` | `d999b7f5cdf1f3ff4d8778d7547b9b93d5af969ba54b996f6a0ff8f7aeade466` |

## 验证

- 前端生产构建、TypeScript、i18n 检查及全量 ESLint 通过。
- 44 个前端测试文件、675 项测试通过，覆盖原生新增配置和保留定制。
- Go 嵌入前端与模型注册表测试通过。
- 上游变更的服务层测试文件中选取的 374 个测试函数，加定制模型广场测试，通过。
- handler、协议转换、server 等包在首轮全量测试中通过；
  repository 清除本机失效代理后整包重跑通过。
- 后端带嵌入前端的生产构建通过。
- 真实 Chrome 检查桌面亮色和手机宽度暗色视口：
  登录动画持续更新、自定义 favicon 生效、无横向溢出或页面 JS 异常，
  非支付页面不请求 Airwallex，非导出页面不加载表格库。
- PostgreSQL 16 独立临时实例执行上游 `252` 迁移：
  旧价格保留、NULL 默认值保留、重复执行幂等、已有倍率保留，
  零和负倍率拒绝且不会破坏原价格。实例已停止。

### 尚未通过的验证与边界

- Windows 首轮全量后端单元测试没有全部通过。
- 配置包读到了本机 `D:\app\data\config.yaml`。在独立工作目录运行原测试二进制后通过。
- 本机失效代理影响了验证码测试，清除测试进程代理后 repository 测试通过。
- 两项时间相关测试在合并前 `3e82442350` 的独立源码快照上同样复现：
  `TestOllamaProbeCallback_StaleLongDoesNotOverrideNewShort`、
  `TestCommandCodeGatewayPassesThroughCatalogProtocols`。
  对应实现和测试均未被本次合并改动，没有放宽断言或修改业务代码。
- 本机 Docker Desktop 因 `dockerInference` socket 错误无法启动，
  完整 Linux 容器集成套件未运行。单独的 PostgreSQL 迁移验证不替代该套件。
- 浏览器手机检查为窄视口模拟，未进行真实手机或付费模型调用测试。

本地详细日志位于 `.cache/upstream-merge-20261118/`。
