# Changelog

所有版本的变更记录。格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

---

## [v1.0.1] - 2026-05-17

完整 15 步 AWS Builder ID 自动注册（OIDC → 设备授权 → 邮箱验证 → 密码设置 → SSO → Kiro Token 交换）
注册完成后自动验证账号存活状态
支持批量注册，可配置数量、并发数、任务间隔
邮箱支持

Outlook 邮箱池：导入账号，自动 IMAP 获取验证码
MoeMail 临时邮箱：多域名配置，支持随机/全部/指定域名模式
反检测

随机化 Chrome 版本（120–144）及设备指纹
TLS 指纹模拟，WebGL / Canvas 伪造
其他

全局代理支持（HTTP / HTTPS / SOCKS5）
注册结果 JSON 输出，可配置输出目录
实时日志、概览仪表盘
自动更新（SHA256 校验 + 无感替换重启）
深色 / 浅色主题

