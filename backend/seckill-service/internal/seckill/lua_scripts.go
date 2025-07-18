package seckill

// 秒杀 Lua 脚本 - 原子性库存扣减 + 用户去重
const SeckillLuaScript = `
-- 秒杀 Lua 脚本
-- KEYS[1]: 库存key (seckill:stock:productId)
-- KEYS[2]: 用户购买记录key (seckill:users:productId)
-- KEYS[3]: 活动信息key (seckill:activity:productId)
-- ARGV[1]: 用户ID
-- ARGV[2]: 购买数量
-- ARGV[3]: 当前时间戳

local stock_key = KEYS[1]
local users_key = KEYS[2]
local activity_key = KEYS[3]
local user_id = ARGV[1]
local quantity = tonumber(ARGV[2])
local current_time = tonumber(ARGV[3])

-- 返回码定义
local RESULT_SUCCESS = 1        -- 成功
local RESULT_STOCK_NOT_FOUND = -1   -- 库存不存在
local RESULT_INSUFFICIENT_STOCK = -2   -- 库存不足
local RESULT_USER_ALREADY_BOUGHT = -3   -- 用户已购买
local RESULT_ACTIVITY_NOT_FOUND = -4   -- 活动不存在
local RESULT_ACTIVITY_NOT_STARTED = -5   -- 活动未开始
local RESULT_ACTIVITY_ENDED = -6   -- 活动已结束
local RESULT_INVALID_QUANTITY = -7   -- 无效数量

-- 验证购买数量
if quantity <= 0 then
    return RESULT_INVALID_QUANTITY
end

-- 检查活动是否存在
local activity_info = redis.call('GET', activity_key)
if not activity_info then
    return RESULT_ACTIVITY_NOT_FOUND
end

-- 解析活动信息 (简化版，实际可能需要 JSON 解析)
-- 这里假设活动信息包含开始时间和结束时间
local activity_data = cjson.decode(activity_info)
local start_time = tonumber(activity_data.start_time)
local end_time = tonumber(activity_data.end_time)

-- 检查活动时间
if current_time < start_time then
    return RESULT_ACTIVITY_NOT_STARTED
end

if current_time > end_time then
    return RESULT_ACTIVITY_ENDED
end

-- 检查用户是否已经购买
local user_bought = redis.call('SISMEMBER', users_key, user_id)
if user_bought == 1 then
    return RESULT_USER_ALREADY_BOUGHT
end

-- 检查库存是否存在
local current_stock = redis.call('GET', stock_key)
if not current_stock then
    return RESULT_STOCK_NOT_FOUND
end

current_stock = tonumber(current_stock)

-- 检查库存是否足够
if current_stock < quantity then
    return RESULT_INSUFFICIENT_STOCK
end

-- 扣减库存
local new_stock = current_stock - quantity
redis.call('SET', stock_key, new_stock)

-- 添加用户购买记录
redis.call('SADD', users_key, user_id)

-- 设置用户购买详情
local purchase_key = 'seckill:purchase:' .. activity_data.product_id .. ':' .. user_id
local purchase_info = {
    user_id = user_id,
    product_id = activity_data.product_id,
    quantity = quantity,
    purchase_time = current_time,
    status = 'pending'
}
redis.call('SET', purchase_key, cjson.encode(purchase_info))
redis.call('EXPIRE', purchase_key, 3600)  -- 1小时过期

-- 返回成功和剩余库存
return {RESULT_SUCCESS, new_stock}
`

// 简化版秒杀脚本（不依赖 cjson）
const SeckillSimpleLuaScript = `
-- 简化版秒杀 Lua 脚本
-- KEYS[1]: 库存key (seckill:stock:productId)
-- KEYS[2]: 用户购买记录key (seckill:users:productId)
-- ARGV[1]: 用户ID
-- ARGV[2]: 购买数量

local stock_key = KEYS[1]
local users_key = KEYS[2]
local user_id = ARGV[1]
local quantity = tonumber(ARGV[2])

-- 返回码定义
local RESULT_SUCCESS = 1
local RESULT_STOCK_NOT_FOUND = -1
local RESULT_INSUFFICIENT_STOCK = -2
local RESULT_USER_ALREADY_BOUGHT = -3
local RESULT_INVALID_QUANTITY = -7

-- 验证购买数量
if quantity <= 0 then
    return RESULT_INVALID_QUANTITY
end

-- 检查用户是否已经购买
local user_bought = redis.call('SISMEMBER', users_key, user_id)
if user_bought == 1 then
    return RESULT_USER_ALREADY_BOUGHT
end

-- 检查库存是否存在
local current_stock = redis.call('GET', stock_key)
if not current_stock then
    return RESULT_STOCK_NOT_FOUND
end

current_stock = tonumber(current_stock)

-- 检查库存是否足够
if current_stock < quantity then
    return RESULT_INSUFFICIENT_STOCK
end

-- 扣减库存
local new_stock = current_stock - quantity
redis.call('SET', stock_key, new_stock)

-- 添加用户购买记录
redis.call('SADD', users_key, user_id)

-- 返回成功和剩余库存
return {RESULT_SUCCESS, new_stock}
`

// 库存回滚脚本
const StockRollbackLuaScript = `
-- 库存回滚 Lua 脚本
-- KEYS[1]: 库存key (seckill:stock:productId)
-- KEYS[2]: 用户购买记录key (seckill:users:productId)
-- ARGV[1]: 用户ID
-- ARGV[2]: 回滚数量

local stock_key = KEYS[1]
local users_key = KEYS[2]
local user_id = ARGV[1]
local quantity = tonumber(ARGV[2])

-- 检查用户是否已购买
local user_bought = redis.call('SISMEMBER', users_key, user_id)
if user_bought == 0 then
    return 0  -- 用户未购买，无需回滚
end

-- 回滚库存
redis.call('INCRBY', stock_key, quantity)

-- 移除用户购买记录
redis.call('SREM', users_key, user_id)

-- 获取回滚后的库存
local new_stock = redis.call('GET', stock_key)
return tonumber(new_stock)
`

// 批量检查用户购买状态脚本
const BatchCheckUserScript = `
-- 批量检查用户购买状态
-- KEYS[1]: 用户购买记录key (seckill:users:productId)
-- ARGV[1..n]: 用户ID列表

local users_key = KEYS[1]
local result = {}

for i = 1, #ARGV do
    local user_id = ARGV[i]
    local bought = redis.call('SISMEMBER', users_key, user_id)
    result[i] = bought
end

return result
`

// 获取秒杀统计信息脚本
const SeckillStatsScript = `
-- 获取秒杀统计信息
-- KEYS[1]: 库存key (seckill:stock:productId)
-- KEYS[2]: 用户购买记录key (seckill:users:productId)
-- KEYS[3]: 活动信息key (seckill:activity:productId)

local stock_key = KEYS[1]
local users_key = KEYS[2]
local activity_key = KEYS[3]

-- 获取当前库存
local current_stock = redis.call('GET', stock_key)
if not current_stock then
    current_stock = 0
else
    current_stock = tonumber(current_stock)
end

-- 获取购买用户数量
local user_count = redis.call('SCARD', users_key)

-- 获取活动信息
local activity_info = redis.call('GET', activity_key)

-- 返回统计信息
return {
    current_stock,
    user_count,
    activity_info or ""
}
`
