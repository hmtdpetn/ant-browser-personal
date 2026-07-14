# Ant Browser Personal 2.0.1

## 新增 Launch HTTP API

- 实例树分组：`/api/groups` 支持列表、新建、更新、删除和批量移动实例。
- 代理/订阅树分组：`/api/proxy-groups` 支持列表、新建、更新、删除和批量移动代理。
- 删除分组会沿用桌面端语义：子分组和直属成员安全提升到父分组；批量移动时传空 `groupId` 可取消分组。
- 新接口自动继承 Launch Server 的 `127.0.0.1` 限制与可选 `X-Ant-Api-Key` 鉴权，不需要也不允许第三方直接操作 SQLite。

## 文档与验证

- 新增中文 [API.md](API.md)，补齐健康检查、实例、启动、运行状态、自动化、CDP 和两类分组 API 的说明与请求示例。
- 已执行分组 HTTP API 自动化测试、完整 Go 测试、TypeScript 检查、Vite 生产构建、Wails Windows/amd64 构建和 `git diff --check`。

## 升级

从 2.0.0 升级到 2.0.1 没有新增数据库迁移。按既有升级方式保留 `data/` 和个人配置，只更新程序与运行时文件即可。首次从 1.3.x 升级时，仍会自动执行 2.0.0 的 v13/v14 数据库迁移；详情见 `UPGRADE-2.0.0-PERSONAL.zh-CN.md`。
