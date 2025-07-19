-- 创建订单数据库
CREATE DATABASE IF NOT EXISTS order_db;

-- 切换到订单数据库
\c order_db;

-- 创建扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 设置时区
SET timezone = 'Asia/Shanghai';
