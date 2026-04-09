#!/bin/bash

# Sniproxy-TLS-RF One-Click Installer
# Features: Architecture detection, Systemd integration, Auto-restart

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}=================================================${NC}"
echo -e "${GREEN}      Snishaper Dedicated SNI Proxy Installer     ${NC}"
echo -e "${BLUE}=================================================${NC}"

# 1. 权限检查
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}错误: 请使用 root 权限运行此脚本。${NC}"
    exit 1
fi

# 2. 架构检测
ARCH=$(uname -m)
case $ARCH in
    x86_64) BIN_ARCH="amd64" ;;
    aarch64) BIN_ARCH="arm64" ;;
    *) echo -e "${RED}不支持的架构: $ARCH${NC}"; exit 1 ;;
esac

# 3. 设置下载地址
REPO="coolapijust/sniproxy-tls-rf"
VERSION="v0.1"
GITHUB_URL="https://github.com/$REPO/releases/download/$VERSION/sniproxy-tls-rf-linux-$BIN_ARCH"

# 4. 设置下载及安装路径
INSTALL_DIR="/usr/local/bin"
service_file="/etc/systemd/system/sniproxy-tls-rf.service"

# 5. 交互式配置：端口
echo -e "${BLUE}>>> 端口配置${NC}"
# 在 curl | bash 模式下，需要强制从 /dev/tty 读取输入
if [ -t 0 ]; then
    read -p "请输入服务监听端口 [默认 443]: " LISTEN_PORT
else
    read -p "请输入服务监听端口 [默认 443]: " LISTEN_PORT < /dev/tty
fi

if [ -z "$LISTEN_PORT" ]; then
    LISTEN_PORT=443
fi

# 确保端口是数字
if ! [[ "$LISTEN_PORT" =~ ^[0-9]+$ ]]; then
    echo -e "${RED}错误: 端口号必须是纯数字。${NC}"
    exit 1
fi

# 6. 端口冲突检查
echo -e "${BLUE}>>> 正在检查端口 $LISTEN_PORT 是否被占用...${NC}"
OCCUPIED_PID=$(lsof -t -i :$LISTEN_PORT)

if [ ! -z "$OCCUPIED_PID" ]; then
    OCCUPIED_NAME=$(ps -p $OCCUPIED_PID -o comm=)
    echo -e "${RED}警告: 端口 $LISTEN_PORT 已被进程 [ $OCCUPIED_NAME (PID: $OCCUPIED_PID) ] 占用。${NC}"
    if [ -t 0 ]; then
        read -p "是否强制结束该进程并继续? [y/N]: " CONFIRM_KILL
    else
        read -p "是否强制结束该进程并继续? [y/N]: " CONFIRM_KILL < /dev/tty
    fi
    if [[ "$CONFIRM_KILL" =~ ^[Yy]$ ]]; then
        kill -9 $OCCUPIED_PID
        echo -e "${GREEN}进程已清理。${NC}"
    else
        echo -e "${RED}已取消。请手动清理端口后再运行。${NC}"
        exit 1
    fi
fi

# 7. 下载二进制文件
echo -e "${BLUE}[1/4] 下载二进制文件...${NC}"
curl -L "$GITHUB_URL" -o "$INSTALL_DIR/sniproxy-tls-rf"
if [ $? -ne 0 ]; then
    echo -e "${RED}下载失败，请检查网络连接。${NC}"
    exit 1
fi
chmod +x "$INSTALL_DIR/sniproxy-tls-rf"

# 8. 创建 Systemd 服务
echo -e "${BLUE}[2/4] 配置 Systemd 服务 (端口: $LISTEN_PORT)...${NC}"
cat <<EOF > "$service_file"
[Unit]
Description=Snishaper Dedicated SNI Proxy (tls-rf)
After=network.target

[Service]
Type=simple
User=root
ExecStart=$INSTALL_DIR/sniproxy-tls-rf -l :$LISTEN_PORT -v
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# 9. 启动服务
echo -e "${BLUE}[3/4] 启动服务...${NC}"
systemctl daemon-reload
systemctl enable sniproxy-tls-rf
systemctl restart sniproxy-tls-rf

# 10. 完成
echo -e "${BLUE}[4/4] 验证安装状况...${NC}"
if systemctl is-active --quiet sniproxy-tls-rf; then
    echo -e "${GREEN}=================================================${NC}"
    echo -e "${GREEN}安装成功!${NC}"
    echo -e "服务端口: ${BLUE}$LISTEN_PORT${NC}"
    echo -e "日志查询: ${BLUE}journalctl -u sniproxy-tls-rf -f${NC}"
    echo -e "卸载命令: ${BLUE}$0 uninstall${NC}"
    echo -e "-------------------------------------------------"
    echo -e "${GREEN}提示: 若无法连接，请确保防火墙已放行端口:${NC}"
    echo -e "UFW:   ${BLUE}ufw allow $LISTEN_PORT/tcp${NC}"
    echo -e "IPTables: ${BLUE}iptables -I INPUT -p tcp --dport $LISTEN_PORT -j ACCEPT${NC}"
    echo -e "${GREEN}=================================================${NC}"
else
    echo -e "${RED}服务启动失败，请检查 [ journalctl -u sniproxy-tls-rf -n 50 ] 了解原因。${NC}"
fi
