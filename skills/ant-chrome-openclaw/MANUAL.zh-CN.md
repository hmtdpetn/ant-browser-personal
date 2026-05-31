# ant-chrome-openclaw 使用说明

## 1. 安装 Skill 方式

把 `skills/ant-chrome-openclaw` 整个目录复制到 OpenClaw 的 `skills` 目录。

也可以直接用脚本安装。

下面这些相对路径命令，默认都要在项目根目录执行。

### Windows

如果 OpenClaw 已经装在当前机器上，直接执行：

```powershell
pwsh -File skills/ant-chrome-openclaw/scripts/install_ant_chrome_openclaw.ps1 `
  -SetDefaultProfile
```

如果没有探测到 OpenClaw 路径，再显式指定：

```powershell
pwsh -File skills/ant-chrome-openclaw/scripts/install_ant_chrome_openclaw.ps1 `
  -TargetSkillsDir "C:\path\to\openclaw\skills" `
  -ConfigFile "C:\path\to\openclaw\openclaw.json" `
  -SetDefaultProfile
```

### Linux

如果 OpenClaw 已经装在当前机器上，直接执行：

```bash
bash skills/ant-chrome-openclaw/scripts/install_ant_chrome_openclaw.sh \
  --set-default-profile
```

如果没有探测到 OpenClaw 路径，再显式指定：

```bash
bash skills/ant-chrome-openclaw/scripts/install_ant_chrome_openclaw.sh \
  --target-skills-dir /path/to/openclaw/skills \
  --config-file /path/to/openclaw/openclaw.json \
  --set-default-profile
```

如果 Ant Browser 开了 API Key，安装时补上 `-ApiKey` 或 `--api-key`。

安装后重启 OpenClaw。

## 2. 提问方式

每次提问开头都写：

```text
使用 ant-chrome-openclaw skill。
```

然后直接写你的目标。

这类任务会启动、停止、切换实例，或者执行已有自动化脚本，属于有副作用的操作。不要指望它“自动猜到”你要不要切换目标实例，最好明确说清楚：

- 是只查询，还是允许启动 / 停止 / 切换
- 用哪个实例
- 如果实例不唯一，是不是允许它继续筛选，还是先停下来问你

预置脚本场景不用再手写 JSON，也不用把 `health`、`active`、`debugReady`、`runtimeWarning` 这些内部检查重复写进提示词。
直接口头说明：

- 用哪个脚本
- 用哪个实例
- 参数是否用默认值，或者只改哪几个参数

### 启动并接管

```text
使用 ant-chrome-openclaw skill。
启动实例 BUYER_001。
确认 debugReady=true 后接管浏览器，并打开 https://example.com
```

### 执行预置脚本

```text
使用 ant-chrome-openclaw skill。
直接执行预置脚本 news-query-txt。
```

如果脚本已经在 Ant Browser 里绑定了目标实例和默认参数，通常这两句就够了。

只有下面这些情况，才继续补充：

- 脚本还没绑定目标实例
- 你要覆盖默认参数
- 你想强制指定另一个实例

例如：

```text
使用 ant-chrome-openclaw skill。
直接执行预置脚本 news-query-txt。
目标实例用 Code G6AN4Q 的“默认实例”。
如果没启动就先启动。
参数用脚本默认值。
```

### 覆盖默认参数

```text
使用 ant-chrome-openclaw skill。
直接执行预置脚本 news-query-txt。
目标实例继续用脚本默认绑定。
把 keyword 改成 OpenAI Agents，limit 改成 5，其他参数保持默认。
```

### 只接管当前实例

```text
使用 ant-chrome-openclaw skill。
先检查当前 active 实例。
如果已经是 BUYER_001，就直接接管，不要切换到别的实例。
```

### 停止实例

```text
使用 ant-chrome-openclaw skill。
停止 launchCode=BUYER_001 对应的实例。
```

### 只查，不要切换

```text
使用 ant-chrome-openclaw skill。
先检查当前 active 实例和 BUYER_001 的运行状态。
如果当前 active 不是 BUYER_001，不要自动切换，先告诉我。
```

## 3. 注意事项

- 先在 Ant Browser 前端里把实例配置好，再让 OpenClaw 接管
- 提问时尽量写清楚实例名或 `launchCode`，优先用精确标识
- 只有 `debugReady=true` 时才适合接管
- 如果有多个匹配实例，不要让它自动选，先让它告诉你
- `browser stop` 只是断开接管，不是关闭实例
- 如果脚本已经在 Ant Browser 里绑定好了目标实例和默认参数，提示词尽量短，不要重复描述内部 API 流程
