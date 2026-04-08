# 飞书 Bot 输入格式

这版目标只有一个：让你在飞书里像聊天一样，把一条小红书种子“凑齐”，而不是一次写一整段模板。

## 你实际怎么用

### 活动开始

先发一条：

```text
event shanghai-demo-day
```

bot 记住这次活动默认值，直到你改掉。

### 采一条种子最快方式

1. 发小红书主页 URL
2. 发一张封面图
3. 发 `done`

就够了。

### 如果你知道更多，再慢慢补

```text
name 林未
team 虾网探针组
scope 做线下参与者发现与资料沉淀
note 愿意后续配合数据采集
```

这些都不是必填。

## bot 支持的最小命令

- `event <slug>`：设置当前会话默认活动
- 直接发 URL：设置当前草稿的 `xhs_profile_url`
- 直接发图片：设置当前草稿的 `cover_photo`
- `name <text>`：设置 `display_name`
- `team <text>`：设置 `team_name`
- `scope <text>`：设置 `team_scope`
- `note <text>`：设置 `note`
- `show`：查看当前草稿
- `done`：提交当前草稿
- `reset`：放弃当前草稿

## 写串了怎么办

不用删，也不用重新开始。

直接再发一次同字段，bot 采用最后一次为准：

```text
team A组
team B组
```

最终保存 `B组`。

## 漏了怎么办

- 漏了 `name/team/scope/note`：可以直接 `done`
- 漏了 URL：`done` 失败，bot 提示“还缺小红书 URL”
- 漏了图片：`done` 失败，bot 提示“还缺封面图”

## 不建议的输入方式

- 一大段自由文本
- 多张图混着发
- 没有 `event` 就开采很多条

这三种都容易让后续整理变复杂。

## 推荐的一次完整对话

```text
event shanghai-demo-day
https://www.xiaohongshu.com/user/profile/sample_linwei
name 林未
team 虾网探针组
scope 做线下参与者发现与资料沉淀
```

然后发一张图，再发：

```text
done
```
