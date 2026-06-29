# 做了个 Codex 体检工具：看看你的 Codex 到底是不是又变笨了

本帖使用社区开源推广，符合推广要求。我申明并遵循社区要求的以下内容：

我的帖子已经打上 开源推广 标签：是

我的开源项目完整开源，无未开源部分：是

我的开源项目已链接认可 LINUX DO 社区：是

我帖子内的项目介绍，AI 生成、润色内容部分已截图发出：是

以上选择我承诺是永久有效的，接受社区和佬友监督：是

项目地址：

<https://github.com/1222hxy/LD-gpt-check>

产品前端：

<https://xxx.com>

---

最近大家应该都有类似感觉：

同一个 Codex，有时候很聪明，有时候又像没睡醒。

同一道题，昨天能做对，今天突然开始胡扯。

开了 `xhigh`，token 也烧了，结果还不一定稳。

所以我做了一个很小的工具：**LD-gpt-check**。

它的目标很简单：**别再只靠体感说 Codex 变没变笨，先跑一组固定测试看看。**

## 这是什么

LD-gpt-check 是一个开源 Go CLI。

它可以调用你本机已经登录好的 Codex CLI，也可以直接走兼容 API，运行固定 benchmark 题，然后自动统计：

- 答案对不对
- 正确率是多少
- 输入 / 输出 tokens
- reasoning tokens
- 每轮耗时
- TPS

目前默认题是 `candy_21`，也就是糖果抽取题，标准答案是 `21`。

这个题看着不长，但很适合观察模型有没有认真推理。答错时通常也很明显，不需要玄学评分。

## 为什么想让大家一起跑

单个人跑一次只能看个热闹。

但如果 Linux.do 上大家都跑一下，就能看到更多有意思的东西：

- 不同模型的稳定性差异
- `medium / high / xhigh` 到底有没有提升
- 同一个模型在不同时间是否波动
- reasoning tokens 高不高，和答对有没有关系
- Codex/API 后端、中转站和系统环境会不会影响结果

说白了，就是把“我感觉它今天不太行”变成一组能贴出来的数据。

## 安装

前提：

- 已安装 Go 1.22+
- 已安装并登录 Codex CLI，或准备好兼容 API 的 Base URL 和临时 Key

安装：

```bash
go install github.com/1222hxy/LD-gpt-check/cmd/ld-gpt-check@latest
```

跑一次：

```bash
ld-gpt-check run -r xhigh -n 5
```

如果你想指定模型：

```bash
ld-gpt-check run -m gpt-5.5 -r xhigh -n 5
```

不传 `-m` 的话，会使用你本机 Codex 配置里的默认模型。

没有本机 Codex 时可以走 API 模式：

```bash
LD_GPT_CHECK_MODEL_API_KEY="你的临时 API Key" \
ld-gpt-check run --backend api \
  --api-format openai-chat \
  --model-api-base-url "https://api.krill-ai.com/codex/v1" \
  -m gpt-5.4 -n 5
```

建议新建一个临时 Key，跑完马上销毁。

支持的 reasoning effort：

```text
low / medium / high / xhigh
```

想保存原始 JSON：

```bash
ld-gpt-check run -r xhigh -n 5 --json
```

## 推荐大家怎么测

最推荐先跑这个：

```bash
ld-gpt-check run -r xhigh -n 5
```

如果你愿意多测几组，可以这样：

```bash
ld-gpt-check run -r medium -n 5
ld-gpt-check run -r high -n 5
ld-gpt-check run -r xhigh -n 5
```

然后把结果贴到楼里，格式可以用这个：

```text
系统：
Codex 版本：
模型：
reasoning effort：
测试次数：
正确率：
平均 reasoning tokens：
平均耗时：
平均 TPS：
有没有异常：
```

如果你懒得整理，也可以直接贴终端输出。

## 可选上传：接了 Linux.do 登录

项目里也做了 Linux.do OAuth 登录，可以选择把测试 summary 上传到默认后端，后面方便做社区趋势页面。

也可以直接打开产品前端：

<https://xxx.com>

登录后的账号页在同域 `/account`，可以查看账号状态和最近上传记录。

如果只想贡献测试数据、不想在社区展示 Linux.do 身份，可以上传时加 `--anonymous`；页面会用 `匿名` 占位，但模型、准确率、耗时等测试数据仍然正常展示和参与统计。

登录：

```bash
ld-gpt-check login
```

跑完并上传：

```bash
ld-gpt-check run -r xhigh -n 5 --upload
```

默认后端：

```text
https://codexgo.yhklab.com
```

上传不是必须的。本地跑完全可用。

## 隐私说明

这个工具不是拿来收集你本地数据的。

目前上传的是测试 summary，主要包括：

- 模型名
- reasoning effort
- 正确率
- tokens
- reasoning tokens
- 耗时和 TPS
- 题目摘要 / prompt hash
- 答案预览

不会上传：

- 你的 Codex 本地数据库
- 你的完整聊天历史
- 你的本地项目文件
- OAuth secret

测试过程本身发生在本机。CLI 调用的是 `codex exec --json`，并且会使用临时只读工作区、关闭 memories。如果模型尝试调用外部工具，本轮会直接中止。

## 后续想做

现在还是 `0.1.0`，功能很小，但已经能跑。

后面计划继续补：

- 更多固定题库
- 社区结果趋势页
- 不同模型 / effort 对照
- 更完整的自部署文档
- 更清楚的隐私和数据说明

如果你觉得这个方向有用，欢迎帮忙跑一次。

如果你遇到安装、登录、上传、判分、Windows / macOS / Linux 兼容问题，也欢迎直接提 issue 或在楼里反馈。

项目地址：

<https://github.com/1222hxy/LD-gpt-check>

一句话总结：

**别光说 Codex 变笨了，跑一下，让数据说话。**
