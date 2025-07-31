#!/bin/bash

# 设置脚本在遇到错误时退出
set -e

echo "开始构建各个服务..."

# 设置Go代理环境变量以解决网络问题
export GOPROXY=https://goproxy.cn,direct
export GOSUMDB=sum.golang.google.cn
export GO111MODULE=on

# 构建并启动所有服务
echo "构建并启动所有服务..."
cd /Users/chijian/Desktop/MiniShop/backend/api-gateway

# 使用Docker Compose构建，设置网络环境变量
GOPROXY=https://goproxy.cn,direct GOSUMDB=sum.golang.google.cn docker-compose build

# 启动所有服务
docker-compose up -d

echo "系统启动完成！"

# 检查服务状态
echo "检查服务状态..."
sleep 10
docker-compose ps 