# Skills DESIGN

> `skills/` 不再只是“Skill 收藏库”，而是 Red Agent Network 的能力目录、发现层和未来信誉系统的前置设计。

## 1. 设计目标

根据仓库根目录 [README.md](../README.md)、[docs/00-overview.md](../docs/00-overview.md)、[docs/06-peers.md](../docs/06-peers.md)、[docs/08-roadmap.md](../docs/08-roadmap.md) 的语义，`skills/` 在 RedAnet 里的职责应该是：

1. 让 Agent 的能力可被发现。
2. 让能力描述可被搜索、比较和筛选。
3. 让能力包未来可以映射到任务接单、信誉积累和验收标准。
4. 让能力收录具备审核标准，而不是靠主观“感觉这个 prompt 挺强”。

因此，这里的 Skill 不是 prompt 碎片，而是一个面向网络协作的能力包：

- 对 `发现`：Skill 必须能转化为 `discover` / `skills=` 这类搜索语义。
- 对 `通信`：Skill 必须有清晰的触发边界、输入输出和对外说明。
- 对 `协作`：Skill 必须指向一个可重复执行的工作流，而不是一次性答案。
- 对 `信任`：Skill 必须可审、可验证、可评级，才能进入未来的 reputation / task acceptance 体系。

## 2. Vision 对齐

### 2.1 与主 README 的对齐

仓库主 README 定义的是一个“去中心化 Agent 协作网络”，核心关键词是：

- `发现`
- `通信`
- `协作`
- `信任`

`skills/` 的设计必须服务这个网络愿景，而不是只服务本地单机工作流。一个被收录的 Skill，最终应该能回答下面这些网络问题：

- 这个 Agent 会什么，别人如何找到它？
- 这个能力适合什么任务，不适合什么任务？
- 这个能力的输出是否可验收？
- 这个能力是否值得信任，维护成本如何？

### 2.2 与 `discover` / `skills=` 的对齐

`docs/06-peers.md` 已经给出未来搜索形态：

```http
GET /api/discover?q=translation&skills=nlp&limit=10
```

这意味着 skills taxonomy 不能只做面向人类阅读的内容分类，还必须兼容未来网络索引。每个 Skill 都应当至少具备：

- 一个明确的主分类，便于网络过滤。
- 一组次级 tags，便于热点和需求排序。
- 一套审计状态，便于信任体系引用。

换句话说，`skills/` 目录本质上是在为未来的：

- 能力目录
- 搜索排序
- 任务匹配
- 信誉累计

准备语义层。

## 3. 设计原则

1. `Mission first, market aware`
主分类优先服务 RedAnet mission，对热度和话题度保持敏感，但不被热点牵着走。

2. `Workflow over prompt`
收录的是可复用工作流，不是单次提示词。

3. `Audit before admission`
先过准入门槛，再谈收藏价值。

4. `Verifiable outputs`
没有验证路径的 Skill，不进入高等级目录。

5. `Portable by default`
优先收录可以跨 agent / repo / workflow 迁移的 Skill。

6. `Progressive disclosure`
Skill 应该是可分层加载的能力单元，而不是一次性把大量无关上下文塞进 agent。

7. `Prefer evidence over vibes`
高热度只能加分，不能替代证据、边界和可验性。

## 4. 研究基线与参考标准

本设计优先采用官方来源和具生产证明的资料，作为后续分类和收录的参考标准。

### 4.1 官方参考

1. [Agent Skills 概览](https://agentskills.io/home)
价值：确认 Skills 是开放格式，不绑定单一客户端。

2. [What are skills? - Agent Skills](https://agentskills.io/what-are-skills)
价值：明确 Skill 是一个以 `SKILL.md` 为核心、可带 scripts / references / assets 的能力包，并强调 progressive disclosure。

3. [Best practices for skill creators - Agent Skills](https://agentskills.io/skill-creation/best-practices)
价值：给出“从真实任务抽取”“控制上下文开销”“提供默认值而非菜单”“计划-验证-执行”等高价值设计模式。

4. [Optimizing skill descriptions - Agent Skills](https://agentskills.io/skill-creation/optimizing-descriptions)
价值：说明 `description` 是触发精度的主开关，适合作为触发边界审计依据。

5. [Anthropic 官方 skills 仓库](https://github.com/anthropics/skills)
价值：提供高质量公开样例，并明确 skills 范围覆盖 creative/design、technical、enterprise/communication、document 等多类场景。

6. [Claude Skills 帮助文档](https://support.claude.com/en/articles/12512180-using-skills-in-claude)
价值：证明生产级内建 skill 已覆盖 Excel、Word、PowerPoint、PDF，并且官方目录已出现 Notion、Figma、Atlassian 等 partner skills。

7. [OpenAI Codex Skills 文档](https://developers.openai.com/codex/skills)
价值：确认 Skills 是 Codex 的可复用工作流格式，可由 instructions、resources、scripts 组成，并区分 authoring 与 plugin distribution。

8. [OpenAI Codex Use Cases](https://developers.openai.com/codex/use-cases)
价值：提供 Codex 官方公开 use cases 入口，证明 engineering、frontend、integrations 等方向具持续投入。

9. [Using skills to accelerate OSS maintenance](https://developers.openai.com/blog/skills-agents-sdk)
价值：证明 OpenAI 已把 skills 用在真实 OSS maintenance workflow 中。

10. [Building frontend UIs with Codex and Figma](https://developers.openai.com/blog/building-frontend-uis-with-codex-and-figma)
价值：证明前端 / Figma / MCP roundtrip 是官方主推的高影响力 workflow。

11. [GitHub Copilot skills 文档](https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/create-skills)
价值：说明 Skills 是 instructions + scripts + resources 的专门定制层，区别于全局 instructions。

12. [GitHub Copilot 生产力证据](https://github.blog/news-insights/product-news/github-copilot-is-generally-available-for-businesses/)
价值：提供“开发者编码可快至 55%”这一强实证信号，证明 engineering automation 是高收益方向。

### 4.2 参考标准

后续判断一个 Skill 值不值得进目录，至少参考以下四类证据：

1. `Official production evidence`
来自官方产品文档、官方仓库、官方帮助中心、官方案例，且明确在真实产品中使用。

2. `Official workflow evidence`
来自官方 blog / use cases / cookbook / docs，说明该工作流被一线产品团队公开推广。

3. `Cross-platform convergence`
同类 Skill 在 Anthropic、OpenAI、GitHub Copilot、Agent Skills 标准中都出现，说明其不是平台偶然特性，而是高通用需求。

4. `Auditability evidence`
该 Skill 的触发条件、输出模板、验证方式、风险边界是否能写清楚，决定它能否进入高信任等级。

## 5. 分类架构

本目录采用“双轴并重”的设计：

- 主轴：服务 RedAnet mission。
- 副轴：反映市场需求、讨论热度和可交易性。

为避免目录失控，采用 `L0 + L1-L4` 的结构：

- `L0` 是控制平面，不参与主分类竞争。
- `L1-L4` 是目录四层。

### 5.1 L0 Meta Skills

Meta Skills 不描述业务领域，而描述“如何组织和放大其他 Skills”。

- `plan`
- `research`
- `route`
- `execute`
- `verify`
- `audit`
- `package`

这些能力是 Skills 体系的控制平面，负责：

- 定义什么时候该调用别的 Skill
- 定义如何做证据搜集
- 定义如何自检和验收
- 定义如何把 Skill 封装成可分发资产

说明：

- 用户提到的 “Mata Skills” 在本设计中按 `Meta Skills` 理解。
- `lead / teach / post` 不是一级分类，但可以作为 Meta 和 Market 的交叉灵感：
  - `lead` 更接近 route / audit / package
  - `teach` 更接近 research / education
  - `post` 更接近 content-gtm / publishing

### 5.2 L1 Mission Pillars

每个 Skill 必须只有一个主 `pillar`。

#### `discover`

适合帮助 Agent 被发现、被匹配、被理解的能力。

典型能力：

- profile generation
- capability registry
- service description
- search metadata generation
- taxonomy mapping
- benchmark / portfolio summarization

#### `communicate`

适合帮助 Agent 产生高质量对外沟通产物的能力。

典型能力：

- enterprise communication
- brand-safe writing
- stakeholder updates
- proposal / memo / docs
- teaching / explanation
- publishing workflows

#### `collaborate`

适合直接参与多步工作流、交付结果、与其他 Agent 协作的能力。

典型能力：

- engineering automation
- research workflows
- design execution
- data analysis
- document processing
- connector workflows

#### `trust`

适合提升验收、可追溯、安全、合规和审计能力的 Skill。

典型能力：

- code review
- verification
- policy checks
- risk assessment
- output validation
- evidence collection

### 5.3 L2 Market Tags

`market_tags[]` 反映需求和话题度，可多选，但不能抢主分类。

建议标准 tags：

- `engineering`
- `frontend-design`
- `data-analysis`
- `documents`
- `integrations`
- `content-gtm`
- `operations`
- `education`
- `research`
- `leadership`
- `publishing`

说明：

- `lead / teach / post` 可以映射为 `leadership`、`education`、`publishing` 三类次级 tag。
- 如果某个 Skill 同时适合“主线能力”和“流量热点”，用 tags 体现，不要改主 `pillar`。

### 5.4 L3 Maturity

每个 Skill 只能落一个成熟度状态：

- `certified`
- `curated`
- `experimental`
- `deprecated`

定义如下：

#### `certified`

满足以下任一条件：

- 官方内建或官方维护。
- 官方明确说明在生产中使用。
- 有强实证收益，并且触发边界、验证路径、风险边界都清晰。

#### `curated`

满足以下条件：

- 多来源可信证据支持。
- 结构清晰，可复用。
- 输出可以被人工或脚本验证。
- 维护成本可控。

#### `experimental`

满足以下条件：

- 市场趋势强或讨论度高。
- 设计思路有价值。
- 但证据、验证、稳定性或维护方式尚不足以进入 curated。

#### `deprecated`

满足以下任一条件：

- 被更高质量 Skill 替代。
- 依赖失效或成本过高。
- 不再契合 repo mission。

### 5.5 L4 Packaging

每个 Skill 还需要声明包装方式：

- `instruction-only`
- `script-backed`
- `asset-backed`
- `connector-backed`

含义：

- `instruction-only`：主要靠方法论与流程。
- `script-backed`：依赖附带脚本或自动化命令。
- `asset-backed`：依赖模板、样式、参考文件。
- `connector-backed`：依赖外部系统、MCP、API、SaaS 连接器。

## 6. 当前优先收录的 7 类 Skill Family

下表定义“当前已被官方和市场共同验证，且具有高收益 / 高话题度”的一级收录重点。

| Family | 常见主 `pillar` | 典型任务 | 对 RedAnet 的价值 | 官方证据 |
| --- | --- | --- | --- | --- |
| `Engineering Automation` | `collaborate` / `trust` | codebase understanding、PR review、tests、refactor、API upgrade | 直接提升接单效率、验收速度和交付可信度 | OpenAI Codex skills / OSS maintenance、GitHub Copilot productivity research |
| `Front-end / Design Execution` | `collaborate` | screenshot-to-ui、Figma-to-code、visual QA、roundtrip design iteration | 适合高单价任务、跨设计与工程协作 | OpenAI Figma + Codex blog、Anthropic design examples |
| `Data Analysis & Reporting` | `collaborate` | 数据清洗、统计、图表、管理报告 | 高商业价值，易形成交付模板 | Agent Skills best practices、Anthropic enterprise/data examples |
| `Document Intelligence` | `collaborate` / `communicate` | PDF / DOCX / PPTX / XLSX 处理、抽取、生成、编辑 | 文档是企业 Agent 最常见输入输出形态 | Claude built-in document skills、Anthropic production document skills |
| `Integration & Workflow Automation` | `collaborate` / `discover` | GitHub、Slack、MCP、任务触发、handoff | 连接外部系统，增强 agent network 的协作密度 | GitHub skills docs、OpenAI Codex integrations/use cases、Claude partner skills |
| `Enterprise Communication` | `communicate` | brand-safe writing、memo、模板化沟通、内部文档 | 提升对外可信度和组织适配度 | Anthropic repo enterprise/communication、Claude shared/provisioned skills |
| `Research & Exploration` | `discover` / `trust` | web research、codebase research、evidence synthesis、competitive analysis | 为高质量协作、报价、验收和审计提供情报基础 | OpenAI use cases、Agent Skills standard、Codex / Copilot specialized workflows |

### 6.1 为什么这 7 类优先

这 7 类同时满足以下条件：

1. 官方产品和官方文档中反复出现。
2. 任务形态清楚，容易写触发边界。
3. 输出容易审计和模板化。
4. 能映射到 RedAnet 未来的搜索、交易和信誉系统。
5. 高收益且高讨论度，不是冷门“一次性 demo”。

## 7. Admission Policy

`skills/` 不是“看到有趣链接就收”，而是一个有门槛的目录。

### 7.1 基本政策

- 高热度只能加分，不能替代 mission fit。
- 高收益只能加分，不能替代 trigger precision。
- 高话题度只能加分，不能替代 output verifiability。
- 官方背书只能加分，不能替代 trust / safety 审计。

### 7.2 不收录的典型情况

- 只是“写得像 prompt”的一次性答案。
- 触发边界模糊，什么都想做。
- 输出不可检查，只能靠主观感觉。
- 强依赖私有环境，却没有说明依赖成本。
- 与 RedAnet 主线无关，只是蹭热点。

## 8. 公共元数据约定

未来每个正式收录的 Skill 都要填写以下字段：

| 字段 | 说明 | RedAnet 未来语义映射 |
| --- | --- | --- |
| `name` | 唯一标识 | capability id |
| `one_line_value` | 一句话价值主张 | 搜索摘要 / 能力卡片 |
| `pillar` | 主分类，仅一个 | discover filter |
| `market_tags[]` | 次级标签，可多选 | demand ranking / trending filters |
| `evidence_tier` | 证据等级 | reputation seed |
| `proof_links[]` | 证据链接 | 审计追溯 |
| `trigger_when` | 应触发场景 | invocation precision |
| `do_not_trigger_when` | 不应触发场景 | false-positive control |
| `inputs` | 输入约束 | task schema compatibility |
| `outputs` | 输出产物 | deliverable expectations |
| `verification` | 如何验收 | task acceptance |
| `risk_level` | 风险等级 | trust / safety gating |
| `maintenance_cost` | 维护成本 | long-term sustainability |
| `audit_status` | 当前审计状态 | listing state |

`audit_status` 建议枚举值：

- `certified`
- `curated`
- `experimental`
- `deprecated`
- `rejected`

说明：

- `maturity` 只描述已入库 Skill 的生命周期状态。
- `rejected` 是审计结论，不属于已入库成熟度。

### 8.1 建议字段格式

```yaml
name: repo-code-review
one_line_value: Review code changes with a bug-first, acceptance-oriented workflow.
pillar: trust
market_tags:
  - engineering
  - research
evidence_tier: A
proof_links:
  - https://developers.openai.com/codex/use-cases
  - https://github.blog/news-insights/product-news/github-copilot-is-generally-available-for-businesses/
trigger_when:
  - User asks for code review, bug triage, or regression risk analysis.
do_not_trigger_when:
  - User only wants style polish or a rewrite with no review intent.
inputs:
  - diff
  - file references
outputs:
  - prioritized findings
  - residual risks
verification:
  - findings cite file paths and concrete failure modes
risk_level: medium
maintenance_cost: low
audit_status: certified
```

## 9. 证据等级标准

`evidence_tier` 采用四级制：

### `A`

- 官方内建
- 官方维护
- 官方明确生产使用

适合进入 `certified`。

### `B`

- 官方样例库
- 官方 use cases
- 官方 blog / cookbook / docs 中明确推荐

通常可进入 `curated`，若验证充分也可进 `certified`。

### `C`

- 多平台重复出现
- 社区成熟实践强
- 但缺少官方生产级背书

通常进入 `curated` 或 `experimental`。

### `D`

- 热点强
- 可想象空间大
- 但缺乏稳定证据

默认只能进 `experimental`。

## 10. Audit Standard

收录流程分两步：

1. 先过 Gate。
2. 再做评分和定级。

### 10.1 Gate 必过项

以下 7 项任何一项不通过，直接拒绝：

1. `格式合规`
必须具备清晰名称、用途、边界、输入输出。

2. `单一连贯任务`
Skill 必须解决一个连贯工作流，不能把多个松散能力硬塞在一起。

3. `触发边界清晰`
必须说明何时触发、何时不触发。

4. `输出可检查`
最终产物必须能被人或脚本检查。

5. `验证路径明确`
至少给出一种可执行的验收方式。

6. `风险可说明`
至少说明主要风险、依赖或误用方式。

7. `与 repo mission 有对应关系`
必须能映射到 `discover / communicate / collaborate / trust` 之一。

### 10.2 评分维度

每项 0-5 分，总分 40。

| 维度 | 问题 |
| --- | --- |
| `mission_fit` | 是否真正服务 RedAnet 主线？ |
| `demand_monetizability` | 是否高频、高收益、可交易？ |
| `evidence_strength` | 是否有强证据支撑？ |
| `trigger_precision` | 是否容易在正确时机被触发？ |
| `output_verifiability` | 输出是否容易验收？ |
| `trust_safety` | 是否有清晰风险边界？ |
| `portability_reuse` | 是否易迁移、易复用？ |
| `maintenance_burden` | 是否维护成本合理？ |

说明：

- `maintenance_burden` 分数越高表示维护负担越低、可持续性越好。
- 若某 Skill 高收益但高维护，则不应因热度而高分通过。

### 10.3 定级规则

#### `certified`

必须同时满足：

- Gate 全通过
- 总分 `>= 31`
- `evidence_strength >= 4`
- `trigger_precision >= 4`
- `output_verifiability >= 4`
- `trust_safety >= 4`

#### `curated`

必须同时满足：

- Gate 全通过
- 总分 `>= 24`
- `evidence_strength >= 3`
- `trigger_precision >= 3`
- `output_verifiability >= 3`

#### `experimental`

必须同时满足：

- Gate 全通过
- 总分 `>= 16`
- 但未达到 curated 或 certified 标准

#### `rejected`

满足以下任一条件：

- Gate 任意一项失败
- 总分 `< 16`
- 或与 repo mission 无关

### 10.4 审计流程

1. 收集候选 Skill 与来源。
2. 提取公共元数据。
3. 做 Gate 检查。
4. 打 8 项分数。
5. 生成 `audit_status` 与简短结论。
6. 若进入 `curated` 或以上，补充维护建议和复审周期。

## 11. 收录与命名建议

建议未来目录结构保持简单，不提前过度设计：

```text
skills/
├── README.md
├── DESIGN.md
├── certified/
├── curated/
├── experimental/
└── deprecated/
```

每个 Skill 自身再根据需要采用标准结构：

```text
my-skill/
├── SKILL.md
├── scripts/
├── references/
├── assets/
└── audit.yaml
```

说明：

- 成熟度目录优先于市场目录，避免一个 Skill 在多个分类里重复出现。
- `pillar` 和 `market_tags[]` 通过元数据表达，不通过目录层级重复编码。

## 12. 三个示例审计条目

下面的示例不是正式收录结果，而是为了演示这套 rubric 如何区分等级。

### 12.1 示例 A: `repo-code-review`

| 字段 | 值 |
| --- | --- |
| `pillar` | `trust` |
| `market_tags[]` | `engineering`, `research` |
| `evidence_tier` | `A` |
| `audit_status` | `certified` |
| 说明 | 官方 use cases、官方 skills、Copilot 生产力研究共同支持；输出天然可审计，风险边界清晰。 |

建议评分：

- `mission_fit`: 5
- `demand_monetizability`: 5
- `evidence_strength`: 5
- `trigger_precision`: 4
- `output_verifiability`: 5
- `trust_safety`: 5
- `portability_reuse`: 5
- `maintenance_burden`: 4

### 12.2 示例 B: `xlsx-quarterly-report`

| 字段 | 值 |
| --- | --- |
| `pillar` | `collaborate` |
| `market_tags[]` | `data-analysis`, `documents`, `operations` |
| `evidence_tier` | `A` |
| `audit_status` | `curated` |
| 说明 | Claude 内建 XLSX / DOCX / PPTX 能力和官方 examples 证明其高价值；但通常依赖企业模板和数据口径，触发边界与维护成本略高于 code-review。 |

建议评分：

- `mission_fit`: 4
- `demand_monetizability`: 5
- `evidence_strength`: 4
- `trigger_precision`: 3
- `output_verifiability`: 4
- `trust_safety`: 3
- `portability_reuse`: 3
- `maintenance_burden`: 3

### 12.3 示例 C: `launch-post-to-slack-notion`

| 字段 | 值 |
| --- | --- |
| `pillar` | `communicate` |
| `market_tags[]` | `integrations`, `content-gtm`, `publishing`, `leadership` |
| `evidence_tier` | `C` |
| `audit_status` | `experimental` |
| 说明 | 市场需求和话题度强，partner skills 与 connector workflows 也有趋势支持；但不同组织的 brand、审批链、权限模型差异很大，稳定性和验证路径依赖上下文。 |

建议评分：

- `mission_fit`: 4
- `demand_monetizability`: 4
- `evidence_strength`: 2
- `trigger_precision`: 3
- `output_verifiability`: 2
- `trust_safety`: 2
- `portability_reuse`: 3
- `maintenance_burden`: 2

## 13. 维护策略

### 13.1 复审节奏

- `certified`：季度复审
- `curated`：半年复审
- `experimental`：按热点和依赖变化滚动复审
- `deprecated`：仅保留迁移说明

### 13.2 触发复审的条件

- 官方文档或 API 发生重大变化
- 关键依赖失效
- 维护成本持续上升
- 更高质量 Skill 出现
- 与 repo mission 的关系变弱

## 14. 当前结论

对 RedAnet 而言，`skills/` 的正确建设路径不是“先疯狂收藏，再慢慢整理”，而是：

1. 先建立 taxonomy。
2. 先建立 audit rubric。
3. 先定义公共元数据。
4. 再按 `certified -> curated -> experimental` 的顺序持续收录。

只有这样，`skills/` 才能从个人提示词收藏夹，升级成一个能服务：

- 网络发现
- 任务匹配
- 多 Agent 协作
- 信任和信誉积累

的能力基础设施。
