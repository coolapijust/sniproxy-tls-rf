# Sniproxy-TLS-RF

SniShaper `tls-rf`  模式的简单Sniproxy服务端实现，可用于为SniShaper或lumine提供自建上游。
这只是一个最简单的实现示例，你可以发挥脑洞，为各个模式涉及不同的解决方案。

## 特性

- **分片重组支持**：原生处理 `tls-rf` 模式产生的碎片化 TLS ClientHello，自动重组。
- **简单防滥用**：
  - **版本指纹校验**：强制要求 TLS Record 层的 Minor Version 为 `0x04`。
  - **多记录校验**：强制要求 ClientHello 由多个记录组成。
  - **Silent Drop**：对于不符合特征的普通探测包（标准 TLS/HTTP），服务端直接静默关闭连接，不返回任何响应。
  目前这套机制可防止公网非SniShaper/lumine客户端的滥用或主动探测。
- **透明转发**：解析出 SNI 后，原样回放分片包至目标服务器。

## 快速安装 (Linux VPS)

使用以下一键脚本进行安装：

```bash
curl -L https://raw.githubusercontent.com/coolapijust/sniproxy-tls-rf/main/install.sh | sudo bash
```

## 编译运行


```bash
cd sniproxy-tls-rf
go build -o sniproxy-tls-rf
./sniproxy-tls-rf -l :443 -v
```

## 客户端配置 (Snishaper)

在 Snishaper 的规则设置中进行如下配置：

1.  **Mode**: `tls-rf`
2.  **Upstream**: `你的服务器IP`

## 开源协议

MIT
