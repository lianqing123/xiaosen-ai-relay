# sub2api 批量导入 OpenAI 账户分析

## 1. 目标

本文件用于回答下面几个问题：

1. 当前 `sub2api` 源码里，OpenAI 账户添加流程到底是怎样的。
2. 现有源码里是否已经存在“批量导入 / 批量创建 OpenAI 账户”的入口。
3. 如果现有入口不足，后续应该如何设计一个适合批量导入的方案。
4. 无头浏览器是否可行，如果可行，应该放在什么层实现。

结论先写在前面：

- 当前源码里，OpenAI 的“生成授权链接 -> 浏览器登录 -> 复制回调 URL/Code -> 粘贴回来”流程确实存在，而且是默认的手动 OAuth 流程。
- 但这不是唯一入口。源码里实际上已经有一条更适合批量导入的现成路径：`手动输入 Refresh Token`，并且支持“一行一个 RT”的批量创建。
- 另外还存在两个通用批量入口：
  - `POST /api/v1/admin/accounts/batch`
  - `POST /api/v1/admin/accounts/data`
- 所以从“是否完全没有批量导入能力”的角度看，答案是否定的。源码里已经有批量能力，只是入口分散、前端暴露不完整、能力边界不一致。
- 如果后续要做真正稳定的“OpenAI 批量导入”，优先级应该是：
  1. 优先强化现有 `RT 批量创建` 和 `JSON 数据导入`
  2. 其次补一个专门的 `OpenAI 批量导入` 界面
  3. 最后才考虑无头浏览器批量授权

## 2. 关键源码定位

### 2.1 前端入口

- `frontend/src/views/admin/AccountsView.vue`
  - 账户页有“创建账户”和“导入数据”两个入口。
- `frontend/src/components/account/CreateAccountModal.vue`
  - 添加账户主弹窗。
  - OpenAI 的手动 OAuth、RT 批量导入逻辑都在这里。
- `frontend/src/components/account/OAuthAuthorizationFlow.vue`
  - OAuth 第 2 步 UI。
  - 支持：
    - 手动授权
    - 手动输入 RT
    - 手动输入 Mobile RT
  - 还能自动从完整回调 URL 中提取 `code` 和 `state`。
- `frontend/src/components/admin/account/ImportDataModal.vue`
  - 通用 JSON 数据导入弹窗。
- `frontend/src/api/admin/accounts.ts`
  - 封装了 `/admin/accounts`、`/admin/accounts/batch`、`/admin/accounts/data`、`/admin/openai/*` 等接口。
- `frontend/src/composables/useOpenAIOAuth.ts`
  - OpenAI OAuth 的 URL 生成、Code 兑换、RT 校验逻辑。

### 2.2 后端入口

- `backend/internal/server/routes/admin.go`
  - 注册了所有 admin 账户相关路由。
- `backend/internal/handler/admin/openai_oauth_handler.go`
  - OpenAI OAuth 路由处理器。
- `backend/internal/handler/admin/account_handler.go`
  - 单个创建、批量创建、批量刷新等账户通用逻辑。
- `backend/internal/handler/admin/account_data.go`
  - 通用导入导出逻辑。
- `backend/internal/service/openai_oauth_service.go`
  - OpenAI OAuth 状态、Code 兑换、RT 刷新、隐私设置等核心服务。
- `backend/internal/pkg/openai/oauth.go`
  - OpenAI OAuth 常量，包括默认 `client_id`、默认回调地址、状态存储 TTL。

## 3. 当前 OpenAI 添加账户的真实流程

### 3.1 手动 OAuth 流程确实存在

前端在 `CreateAccountModal.vue` 中，OpenAI 选择 OAuth 方式后会进入第 2 步的 `OAuthAuthorizationFlow`。

这个流程是：

1. 调用 `POST /api/v1/admin/openai/generate-auth-url`
2. 后端生成 `state`、`code_verifier`、`session_id`
3. 后端把这些内容保存在内存 session store 中
4. 前端拿到 `auth_url`
5. 用户手动用浏览器打开该地址完成 OpenAI 授权
6. 用户把最终回调 URL 或其中的 `code` 粘贴回页面
7. 前端调用 `POST /api/v1/admin/openai/exchange-code`
8. 拿到 token 后，再调用 `POST /api/v1/admin/accounts` 创建账户

这一点和你的描述一致，只是源码里比纯手工多了一点自动化：

- 前端支持直接粘贴完整回调 URL，不必手动拆 `code`
- 组件会自动从 URL 中提取 `code` 和 `state`

但整体上，它仍然不适合“批量导入几十个甚至上百个 OpenAI 账户”。

### 3.2 默认回调地址和状态存储方式

后端 OpenAI OAuth 默认回调地址是：

- `http://localhost:1455/auth/callback`

状态存储方式是：

- 内存 `sessionStore`
- TTL 为 30 分钟

这带来两个很重要的约束：

1. `generate-auth-url` 和 `exchange-code` 必须命中同一个后端进程，否则 session 取不到。
2. 整个授权流程必须在 30 分钟内完成，否则 session 过期。

这对未来做无头浏览器批量授权也同样成立。

## 4. 源码里已经存在的“批量导入 / 批量创建”能力

## 4.1 已存在能力 A：OpenAI RT 批量创建

这是当前最重要的发现。

在 `CreateAccountModal.vue` 中：

- OpenAI 会显示 `show-refresh-token-option`
- 还会显示 `show-mobile-refresh-token-option`
- `OAuthAuthorizationFlow.vue` 的 RT 输入框支持多行
- 文案里明确写了“每行一个，批量创建账号”

实际逻辑在：

- `handleOpenAIBatchRT`
- `handleOpenAIValidateRT`
- `handleOpenAIValidateMobileRT`

流程是：

1. 在添加账户弹窗里选 OpenAI
2. 进入 OAuth 第 2 步
3. 授权方式切换为：
   - `手动输入 RT`
   - 或 `手动输入 Mobile RT`
4. 一行粘贴一个 RT
5. 前端逐个调用 `POST /api/v1/admin/openai/refresh-token`
6. 用返回的 token 信息组装 credentials
7. 再逐个调用 `POST /api/v1/admin/accounts` 创建 OpenAI OAuth 账户

这说明：

- “OpenAI 批量导入”能力并不是没有，而是已经存在于添加账户弹窗中。
- 只是这条路径依赖于你已经拿到了 RT，而不是依赖回调 URL。

### 4.1.1 RT 批量创建的优势

- 不需要手动生成授权链接
- 不需要手动打开浏览器
- 不需要手动复制回调 URL
- 支持多行 RT，一次批量创建
- 同时支持普通 RT 和 Mobile RT

### 4.1.2 RT 批量创建的限制

- 前端是“逐条校验 + 逐条创建”，不是单个 bulk API。
- 失败重试、断点续传、任务队列都没有。
- 同一批导入只能共享当前表单的全局配置：
  - `proxy_id`
  - `group_ids`
  - `concurrency`
  - `priority`
  - `rate_multiplier`
  - `expires_at`
- 不支持每个 RT 单独设置不同代理、分组或备注。
- 目前这条路径没有利用后端现成的 `/admin/accounts/batch`。
- 这条路径虽然已经能批量创建，但更像“前端循环单创建”，不是产品化的“批量导入模块”。

### 4.1.3 额外说明

普通 RT 默认使用的 client_id 来自后端 OpenAI OAuth 常量：

- `app_EMoamEEZ73f0CkXaXp7hrann`

Mobile RT 在前端里单独写死了一个 client_id：

- `app_LlGpXReQgckcGGUo2JrYvtJK`

这说明作者已经显式考虑过不同 RT 来源的兼容性。

## 4.2 已存在能力 B：通用 JSON 数据导入 `/admin/accounts/data`

账户页上已经有一个“导入数据”按钮，对应：

- `ImportDataModal.vue`
- `POST /api/v1/admin/accounts/data`

该导入只接收 JSON。

数据结构大致是：

```json
{
  "data": {
    "type": "sub2api-data",
    "version": 1,
    "exported_at": "2026-04-23T00:00:00Z",
    "proxies": [],
    "accounts": [
      {
        "name": "openai-01",
        "platform": "openai",
        "type": "oauth",
        "credentials": {
          "refresh_token": "xxx",
          "access_token": "xxx",
          "id_token": "xxx",
          "client_id": "app_EMoamEEZ73f0CkXaXp7hrann",
          "email": "user@example.com"
        },
        "extra": {
          "openai_oauth_responses_websockets_v2_mode": "off"
        },
        "concurrency": 10,
        "priority": 1,
        "rate_multiplier": 1
      }
    ]
  },
  "skip_default_group_bind": true
}
```

### 4.2.1 JSON 数据导入的优势

- 这是当前源码里最接近“真正批量导入”的现成入口。
- 可以一次导入多账号。
- 可以连代理一起导入。
- 导入时支持复用已有代理，不会重复创建。
- 对 OpenAI OAuth 账户，如果 credentials 中带 `id_token`，后端会自动补齐缺失的：
  - `email`
  - `plan_type`
  - `chatgpt_account_id`
  - `chatgpt_user_id`
  - `organization_id`

### 4.2.2 JSON 数据导入的限制

这个入口更偏“数据迁移”，不是“OpenAI 批量采集”。

它的限制很明显：

- 必须先有完整 credentials，不能只给账号密码。
- 不能直接喂“回调 URL 列表”。
- 当前 `DataAccount` 结构不包含：
  - `group_ids`
  - `load_factor`
  - `schedulable`
  - `status`
- 前端导入时显式传了 `skip_default_group_bind: true`
  - 导入完成后默认不会自动绑定平台默认分组

所以：

- 如果你的来源是“另一套 sub2api / 一套已有 JSON 凭证”，这个入口很好用。
- 如果你的来源是“浏览器里登录好的 OpenAI 网页账户”，这个入口本身不解决 RT/Code 的采集问题。

## 4.3 已存在能力 C：后端批量创建 `/admin/accounts/batch`

后端已经注册了：

- `POST /api/v1/admin/accounts/batch`

对应处理器：

- `AccountHandler.BatchCreate`

这说明从 API 层面，项目已经支持一次提交多条 `CreateAccountRequest`。

### 4.3.1 这个接口的价值

- 适合后续做真正的“批量导入弹窗”
- 适合脚本或 CLI 导入
- 比当前前端“for 循环单创建”更像正式批量接口

### 4.3.2 这个接口当前的问题

- Accounts 页面当前并没有调用它。
- 也就是说，后端有 bulk create，前端没有把它真正用起来。
- `BatchCreate` 虽然接收 `CreateAccountRequest`，但实际实现里没有把 `load_factor` 传进 `CreateAccountInput`。
  - 单条创建接口支持 `load_factor`
  - 批量创建接口当前实际会丢掉 `load_factor`

这意味着：

- 它已经可以用，但还不是一个完全对齐单条创建能力的成熟批量入口。

## 4.4 已存在能力 D：`/admin/openai/create-from-oauth`

后端还单独提供了：

- `POST /api/v1/admin/openai/create-from-oauth`

它可以在一次请求里：

1. 用 `session_id + code + state` 兑换 token
2. 直接创建 OpenAI OAuth 账户

这个接口现在前端没有使用，但它对“无头浏览器自动授权”很有价值。

### 4.4.1 这个接口适合什么场景

适合自动化脚本先拿到：

- `session_id`
- `code`
- `state`

然后一次请求直接建号。

### 4.4.2 这个接口当前的限制

它的请求参数比较少，只支持：

- `name`
- `proxy_id`
- `concurrency`
- `priority`
- `group_ids`

它不支持更完整的创建参数，例如：

- `notes`
- `rate_multiplier`
- `load_factor`
- `expires_at`
- `auto_pause_on_expired`
- `extra`

所以如果后续做自动化，且需要和现有“创建账户弹窗”完全对齐，仍然更推荐：

1. 自动化拿到 `code/state`
2. 调 `exchange-code`
3. 再调 `/admin/accounts` 或 `/admin/accounts/batch`

而不是直接依赖 `create-from-oauth`。

## 5. 当前源码里还没有的能力

尽管已有 RT 批量导入和 JSON 导入，但以下能力仍然缺失：

### 5.1 没有“批量浏览器授权”能力

当前源码没有找到成熟的浏览器自动化实现：

- 没有 Playwright 流程
- 没有 Puppeteer 流程
- 没有 chromedp 流程
- 没有专门的 worker 去自动登录 OpenAI 网页并收集 callback

所以：

- 源码里现在没有“批量打开多个 OpenAI 授权链接并自动收集回调”的产品能力。

### 5.2 没有专门的 OpenAI 批量导入页面

当前批量能力分散在三个地方：

- 添加账户弹窗中的 RT 批量创建
- 通用 JSON 数据导入
- 后端 `/admin/accounts/batch`

但没有一个专门的“OpenAI 批量导入”界面把这些能力统一起来。

### 5.3 没有来源分层

源码目前没有把 OpenAI 导入来源拆成明确的几类：

- 来源 A：已有 Refresh Token
- 来源 B：已有 sub2api/JSON 导出
- 来源 C：已有浏览器登录态 / Cookie / profile
- 来源 D：只有账号密码，需要浏览器登录

如果后续真的要做“批量导入”，我建议必须先把来源分层，否则产品会混乱。

## 6. 针对你的需求，当前最可行的三条路径

## 6.1 路径一：如果你已经有 RT，直接走现有 RT 批量创建

这是当前源码下最值得优先使用的路径。

### 操作步骤

1. 进入账户页
2. 点击“创建账户”
3. 平台选 `OpenAI`
4. 账户类型走 `OAuth`
5. 填好统一的批量参数：
   - 名称前缀
   - 代理
   - 分组
   - 并发
   - 优先级
   - 计费倍率
6. 进入第 2 步后，把授权方式切换为：
   - `手动输入 RT`
   - 或 `手动输入 Mobile RT`
7. 每行粘贴一个 RT
8. 点击校验并创建

### 适用条件

- 你已经能拿到 Refresh Token
- 不需要每个账号单独配置不同代理/分组
- 不要求导入过程有任务管理、断点续传、失败重跑

### 评价

- 这是“现有源码可直接用”的最佳方案
- 不需要无头浏览器
- 也不需要改代码

## 6.2 路径二：如果你已有完整凭证，走 JSON 数据导入

如果你的来源是另一套 sub2api，或者你手里已经有标准化 credentials，这条路更稳。

### 操作步骤

1. 准备符合 `DataPayload` 结构的 JSON
2. 账户记录里写：
   - `platform: openai`
   - `type: oauth`
   - `credentials: { ... }`
3. 如果有代理，先写到 `proxies`
4. 账户通过 `proxy_key` 引用代理
5. 在账户页点击“导入数据”
6. 选择 JSON 文件导入

### 最佳实践

- 最好先从现有系统导出一个 `/admin/accounts/data` JSON 做模板
- 不建议从零手写字段名和字段格式

### 评价

- 这是“已有凭证批量迁移”的最佳方案
- 也不需要无头浏览器

## 6.3 路径三：如果你只有网页登录态，才考虑无头浏览器

如果你没有 RT，也没有现成 JSON，只能通过 OpenAI 网页授权拿到 callback，那么无头浏览器才有意义。

但这里必须明确：

- 这不是当前源码里的现成功能
- 这是后续要新增的一套自动化方案

## 7. 无头浏览器方案是否可行

结论：可行，但不应该是第一优先级。

## 7.1 为什么可行

现有后端已经具备自动化所需的关键接口：

1. `POST /api/v1/admin/openai/generate-auth-url`
   - 生成授权链接、返回 `session_id`
2. `POST /api/v1/admin/openai/exchange-code`
   - 用 `session_id + code + state` 换 token
3. `POST /api/v1/admin/openai/create-from-oauth`
   - 直接把 `session_id + code + state` 变成账户

所以无头浏览器并不需要自己去实现 OAuth 协议，它只需要做一件事：

- 自动打开授权页并拿到最终 callback 中的 `code` 和 `state`

## 7.2 推荐的无头浏览器实现方式

推荐使用 Playwright 或 chromedp，但建议作为“独立 worker / 独立脚本”实现，而不是塞进前端页面里。

推荐流程：

1. 调 `generate-auth-url`
2. 启动浏览器上下文
3. 打开 `auth_url`
4. 完成登录、授权
5. 监听最终跳转到 `http://localhost:1455/auth/callback?...`
6. 取出 `code` 和 `state`
7. 调：
   - `create-from-oauth`
   - 或 `exchange-code` + `/admin/accounts`
8. 记录结果

## 7.3 为什么不建议先做无头浏览器

因为它的失败点太多：

- OpenAI 登录态校验
- 人机验证 / 风控
- 二次验证
- 多账号并发封禁风险
- 浏览器 profile 隔离
- 本地回调监听和上下文绑定
- 多实例后端下的 `session_id` 一致性问题

从工程成本看，它明显比“RT 批量导入”贵很多。

## 7.4 无头浏览器更适合什么场景

- 没有 RT
- 没有现成导出 JSON
- 但已经有人在浏览器里登录好了多个 OpenAI 账号
- 或者必须通过交互式网页授权获取 token

## 8. 如果后续要做新功能，推荐的实现优先级

## 8.1 第一阶段：做一个正式的 OpenAI 批量导入界面

这是我最推荐的方案。

### 目标

把现有零散能力整合成一个单独入口，而不是让用户自己在“添加账户”弹窗里猜。

### 建议支持的导入来源

1. `RT 批量导入`
2. `Mobile RT 批量导入`
3. `JSON 凭证导入`

### 不建议第一阶段就做的内容

- 浏览器自动登录
- 账号密码直登
- 自动处理验证码

### 第一阶段的产品形态

建议新增一个独立弹窗或独立页面：

- 入口：`OpenAI 批量导入`
- 模式切换：
  - `Refresh Token`
  - `Mobile Refresh Token`
  - `JSON`

### 第一阶段的后端建议

- 复用现有 `/admin/openai/refresh-token`
- 复用现有 `/admin/accounts/batch`
- 不要继续让前端逐条 `create`
- 改成：
  1. 前端批量校验 RT
  2. 前端一次性提交 `/admin/accounts/batch`

## 8.2 第二阶段：补齐 batch 接口和 data 导入缺口

建议补齐下面这些差异：

### `/admin/accounts/batch`

- 把 `load_factor` 真正传入后端创建逻辑
- 和单创建接口能力保持一致

### `/admin/accounts/data`

- 是否要支持 `group_ids`
- 是否要支持 `load_factor`
- 是否要支持 `schedulable`
- 是否要支持保留状态字段

否则它只能算“迁移接口”，很难算“批量导入接口”。

## 8.3 第三阶段：做无头浏览器 worker

只有在前两阶段仍然无法满足业务时，再进入这一步。

### 推荐形态

一个独立 worker，而不是在前端里直接驱动浏览器。

### 推荐能力

- 每个账号独立 browser context
- 支持 profile 持久化
- 支持每账号代理
- 支持人工接管
- 支持截图、日志、失败回放
- 支持并发数限制

### 推荐调用链

1. worker 调 `/admin/openai/generate-auth-url`
2. browser context 打开授权页
3. 捕获 callback URL
4. worker 调 `/admin/openai/exchange-code`
5. worker 调 `/admin/accounts/batch` 或 `/admin/accounts`

## 9. 当前源码下的建议结论

结合你的需求，我给出的判断是：

### 9.1 现在就能用的方案

优先试现有的 `RT 批量创建`。

原因：

- 源码已经实现
- 不需要改业务代码
- 不需要无头浏览器
- 最贴近“批量导入 OpenAI 账户”

### 9.2 如果你的数据源不是 RT

如果你的来源是现成 credentials 或历史系统导出：

- 用 `账户数据导入 /admin/accounts/data`

### 9.3 只有在这两条都不满足时

才值得继续设计“无头浏览器批量授权”。

## 10. 我认为后续最合理的产品化路线

### 推荐路线

1. 保留当前手动 OAuth 流程
   - 作为单账号兜底入口
2. 明确暴露 RT 批量导入
   - 作为主入口
3. 明确暴露 JSON 导入
   - 作为迁移入口
4. 后续再评估无头浏览器 worker
   - 作为最后手段

### 不推荐路线

直接围绕“手动 callback URL 粘贴”去做批量化。

原因很简单：

- 这条链路天生就不适合人工批量操作
- 也不适合直接在前端里堆复杂交互
- 现有源码已经提供了更适合批量化的 RT 方案

## 11. 附：这次分析得到的关键结论摘要

### 已存在

- OpenAI 手动 OAuth 授权流程
- OpenAI RT 批量创建
- OpenAI Mobile RT 批量创建
- 通用 JSON 数据导入
- 后端批量创建接口
- 后端 `create-from-oauth` 自动化接口

### 不存在

- 成熟的无头浏览器批量授权流程
- 专门的 OpenAI 批量导入页面
- 统一的来源分层设计

### 推荐顺序

1. RT 批量导入
2. JSON 数据导入
3. 无头浏览器自动化
