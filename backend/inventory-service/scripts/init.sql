-- 创建库存数据库
CREATE DATABASE IF NOT EXISTS inventory_db;

-- 切换到库存数据库
\c inventory_db;

-- 创建扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 设置时区
SET timezone = 'Asia/Shanghai';
