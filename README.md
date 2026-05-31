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

## 构建方式

```powershell
# 进入仓库目录
cd <仓库路径>

# 构建（会自动编译前端和 Go）
wails build

# 构建产物路径
build\bin\ant-chrome.exe
```

**环境要求**：Go 1.21+、Node.js 18+、Wails v2、Windows 10/11 64 位。

---

## 运行注意事项

- 首次运行会自动生成 `config.yaml` 和 `data/app.db`，无需手动创建
- **不要提交**以下内容到 Git：
  - 真实代理配置、订阅 URL、token、`api_key`
  - `node_modules/`、`frontend/dist/`、`build/bin/`
  - `data/`（数据库、实例数据、日志、缓存）
- 运行时 `bin/` 目录下的 `xray.exe` / `sing-box.exe` 已随仓库提供，无需单独下载

---

## 浏览器内核

代理运行时已内置，只需单独准备指纹 Chrome 内核：

1. 打开应用 → `内核管理` → 添加内核路径
2. 或手动下载后指向包含 `chrome.exe` 的目录

推荐内核来源：[fingerprint-chromium](https://github.com/adryfish/fingerprint-chromium/releases)

---

## License

遵循上游 [Ant Browser](https://github.com/black-ant/Ant-Browser) 的许可协议。
