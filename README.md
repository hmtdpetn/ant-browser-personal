# AntBrowser 个人自用版

> 这是基于 [Ant Browser](https://github.com/black-ant/Ant-Browser) 的个人自用构建，**不是官方版本，不用于商用，不用于分发**。
>
> 仓库为私有，仅供个人备份和重新构建使用。

---

## 与官方版本的主要区别

| 改动 | 说明 |
|---|---|
| AnyTLS Clash YAML 支持 | 代理池可导入 `type: anytls` 的 Clash YAML 节点，直接交给 sing-box 处理 |
| User-Agent fallback | Clash 订阅 URL 下载轮询多个 UA（clash-verge / FlClash），避免因 UA 被拦截导入失败 |
| UI 清理 | 移除官方作者介绍、博客、掘金、公众号、GitHub Star 引导、项目宣传等展示内容 |
| 实例上限调整 | 默认实例上限从官方限制改为 `9999`，适合个人长期使用 |
| 首页实例上限显示 | 控制台"系统信息"卡片新增"实例上限"行，直接展示当前 `maxProfileLimit` 值 |
| 窗口尺寸适配 | 初始窗口缩小，适配虚拟机小分辨率，避免窗口超出屏幕导致关闭按钮不可见 |

---

## 当前默认配置

```yaml
app:
  window:
    width: 1280
    height: 720
    min_width: 960
    min_height: 560
  max_profile_limit: 9999
```

---

## 构建环境要求

Go 1.21+、Node.js 18+、Wails v2、Windows 10/11 64 位。

---

## Windows 便携版（个人推荐发布方式）

> **不要直接用 `wails build` 生成的单个 exe 当作完整运行包。**
> `wails build` 只输出主程序 exe，不包含代理运行时（`xray.exe`、`sing-box.exe`），
> 代理实例和 IP 健康检测会因找不到这两个文件而失败。

### 生成便携 zip

```powershell
bat\package-portable.bat
```

输出路径：

```
dist\ant-browser-personal-windows-amd64-portable-{version}.zip
```

### zip 内结构

```
ant-browser-personal\
  ant-chrome.exe          主程序
  config.yaml             初始配置模板
  bin\
    xray.exe              代理运行时（必需）
    sing-box.exe          代理运行时（必需）
  data\                   运行时数据目录（首次启动自动初始化）
```

### 使用方式

1. 解压 zip 到任意目录
2. 运行 `ant-browser-personal\ant-chrome.exe`
3. `bin\xray.exe` 和 `bin\sing-box.exe` 是代理实例和 IP 健康检测所需文件，**不要删除**
4. `data\` 是运行时数据目录，首次启动后自动生成数据库和实例数据，**不应提交到 git**

### 跳过构建步骤（已有构建产物时）

```powershell
bat\package-portable.bat -SkipBuild
```

### GitHub Release

- 不要把 `dist\*.zip` 提交进主分支
- 正式发布时把 zip 上传为 GitHub Release 附件
- 旧 Release 若有单独 exe 附件，建议删除或标记废弃，改用 portable zip

---

## 运行注意事项

- 首次运行会自动生成 `data/app.db`，无需手动创建
- **不要提交**以下内容到 Git：
  - 真实代理配置、订阅 URL、token、`api_key`
  - `node_modules/`、`frontend/dist/`、`build/bin/`
  - `data/`（数据库、实例数据、日志、缓存）
  - `dist/`（portable zip 输出目录）
- `bin/xray.exe` 和 `bin/sing-box.exe` 已随仓库提交，无需单独下载

---

## 浏览器内核

代理运行时已内置，只需单独准备指纹 Chrome 内核：

1. 打开应用 → `内核管理` → 添加内核路径
2. 或手动下载后指向包含 `chrome.exe` 的目录

推荐内核来源：[fingerprint-chromium](https://github.com/adryfish/fingerprint-chromium/releases)

---

## License

遵循上游 [Ant Browser](https://github.com/black-ant/Ant-Browser) 的许可协议。
