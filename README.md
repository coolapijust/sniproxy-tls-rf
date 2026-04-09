# Sniproxy-TLS-RF

专属 `tls-rf` (TLS Reverse Fragment) 模式的 SNI 代理服务端实现。

## 特性

- **专属分片支持**：原生处理 `tls-rf` 模式产生的碎片化 TLS ClientHello，支持自动重组。
- **高强度防滥用**：
  - **版本指纹校验**：强制要求 TLS Record 层的 Minor Version 为 `0x04`。
  - **多记录校验**：强制要求 ClientHello 由多个记录组成。
  - **Silent Drop**：对于不符合特征的普通探测包（标准 TLS/HTTP），服务端直接静默关闭连接，不返回任何响应。
- **透明转发**：解析出 SNI 后，原样回放分片包至目标服务器，确保端到端规避效果。

## 快速安装 (Linux VPS)

使用以下一键脚本进行安装：

```bash
curl -L https://raw.githubusercontent.com/coolapijust/sniproxy-tls-rf/main/install.sh | sudo bash
```
*(注：请在上传至 GitHub 后替换为实际地址)*

## 编译运行

如果你想手动编译：

```bash
cd sniproxy-tls-rf
go build -o sniproxy-tls-rf
./sniproxy-tls-rf -l :443 -v
```

## 客户端配置 (Snishaper)

在 Snishaper 的规则设置中进行如下配置：

1.  **Mode**: `tls-rf`
2.  **Upstream**: `你的服务器IP:443`
3.  **TLS-RF 配置**: 保持默认 (Records=4, ModMinorVer=true)

## 开源协议

MIT
