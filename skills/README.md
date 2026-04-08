# Skills

`skills/` 现在的目标不再只是“整理外部 Skill 链接”，而是直接把公开第三方 `SKILL.md` 收进仓库，做成一个可浏览、可筛选、可安装的大库。

我们要解决的是一个非常具体的问题：

- 外面已经有人写了很多可复用 Skill
- 用户不想自己到处找、手工复制、再猜怎么装
- 仓库需要一个统一入口，把 Skill 收进来、补元数据、再交给 manager 做懒人式安装

## 这是什么

这个目录现在承担 3 个职责：

1. 作为第三方 Skill 的仓库内集合点
2. 为每个已入库 Skill 维护最小元数据
3. 为后续的 Skills manager 提供统一的数据来源

换句话说，`skills/` 不再只是目录页，也不只是“有趣项目观察名单”。

## 当前已入库

第一批已经统一落到：

- `skills/third_party/github/...`
- `skills/catalog.json`

当前这一批是“先收进来”的种子库，来源包括：

- `HughYau/qiushi-skill`
- `mliu98/awesome-human-distillation` 里挑出的高活跃仓库
- `VoltAgent/awesome-openclaw-skills` 指向的 `openclaw/skills` 条目

重要边界：

- 目前是 `SKILL.md` 原始入库优先
- 每个 Skill 都带最小 `metadata.json`
- 当前默认状态是 `skill-md-only` + `unverified`

也就是说，已经收进仓库，不等于已经验证可以一键安装运行。

## 当前确认目标

当前主线已经明确：

- 直接抓取公开第三方仓库里的 `SKILL.md`
- 把可用 Skill 放进我们的仓库
- 给每个 Skill 补最小元数据
- 做一个 manager，支持用户傻瓜式挑选和安装

这条主线优先级高于复杂 taxonomy、评级系统和“先观察不落地”的思路。

## 目录要解决什么

第一版先解决 4 件事：

| 能力 | 要解决的问题 |
| --- | --- |
| 收 | 怎么把外部 `SKILL.md` 抓进来，并保留来源信息 |
| 看 | 怎么让用户快速浏览、筛选、理解库里有什么 |
| 选 | 怎么根据元数据做最低限度的搜索和过滤 |
| 装 | 怎么把选中的 Skill 安装到目标环境，而不是手工复制 |

## 最小入库单元

每个已入库 Skill 至少应包含：

- 原始或轻改造后的 `SKILL.md`
- 一个最小元数据文件

元数据不追求复杂，但至少要支撑：

- Skill 标识
- 来源仓库 / 来源路径
- 一句话说明
- 标签或分类
- 安装时展示所需的信息
- 当前导入状态与验证状态

元数据的具体字段可以小，但不能小到 manager 无法用。

## Skills Manager

这个库最终要由一个 manager 来消费，而不是让用户自己翻目录。

第一版 manager 至少要支持：

- `list` / `browse`
- `search` / `filter`
- `show metadata`
- `install`

在 manager 落地之前，`skills/catalog.json` 就是最小统一索引。

目标体验是：

- 用户先挑 Skill
- manager 展示最小信息
- 用户直接安装

而不是：

- 用户先理解仓库结构
- 自己找目录
- 自己复制文件
- 再手动调目标路径

## 收录原则

- 优先收公开可获取的 `SKILL.md`
- 收录时必须保留来源信息，避免变成来源不明的二次拼贴
- 元数据先求够用，不先做过度设计
- 第一批允许先只收 `SKILL.md`，但要明确标记为 `unverified`
- 先把“能收、能看、能装”跑通，再讨论质量评级和大而全分类
- 不要求第一版解决所有上游同步问题

## 下一步

- 定义仓库内的 Skill 入库目录结构
- 冻结最小元数据格式
- 实现一个最小可用的 Skills manager
- 跑通至少一条“从外部仓库抓 Skill -> 入库 -> 安装”的完整链路

设计边界和方向说明，见 [DESIGN](./DESIGN.md)。
