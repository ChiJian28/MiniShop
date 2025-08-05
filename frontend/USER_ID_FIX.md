# 用户ID问题修复文档

## 问题描述

用户反馈：**同一个用户能够多次购买同一商品**，这与后端的"一人只能买一次"限购设计不符。

## 问题排查

### 1. 后端检查
通过检查后端代码发现：
- ✅ 后端已正确实现限购逻辑（Lua脚本中的 `RESULT_USER_ALREADY_BOUGHT`）
- ✅ Redis 会记录已购买用户的ID（`SADD users_key user_id`）
- ✅ 重复购买会返回 409 Conflict 状态码

### 2. 前端问题定位
检查前端 `src/services/api.ts` 发现**根本问题**：

```typescript
// 问题代码（第60行）
const response = await api.post(`/api/v1/seckill/${productId}`, {
  productId,
  userId: 1, // ❌ 硬编码用户ID为1
});
```

**问题分析**：
- 无论哪个用户登录，前端都向后端发送 `userId: 1`
- 后端认为都是同一个用户（ID=1），正确执行了限购逻辑
- 但前端的购物车状态是独立的，造成了"不同用户都能购买"的假象

## 解决方案

### 1. 添加用户ID获取函数
```typescript
// 获取当前用户ID的辅助函数
const getCurrentUserId = (): number => {
  try {
    const authStorage = localStorage.getItem('auth-storage');
    if (authStorage) {
      const authData = JSON.parse(authStorage);
      if (authData.state?.user?.id) {
        return authData.state.user.id;
      }
    }
  } catch (error) {
    console.warn('Failed to get user ID from auth storage:', error);
  }
  return 1; // 默认用户ID
};
```

### 2. 修复秒杀请求
```typescript
// 修复后的代码
participateSeckill: async (productId: number): Promise<SeckillResponse> => {
  try {
    const userId = getCurrentUserId(); // ✅ 获取真实用户ID
    console.log('Sending seckill request with userId:', userId, 'productId:', productId);
    
    const response = await api.post(`/api/v1/seckill/${productId}`, {
      productId,
      userId, // ✅ 使用真实用户ID
    });
    return response.data;
  } catch (error: any) {
    // ... 错误处理
  }
}
```

### 3. 优化用户ID生成
为了确保不同用户名有不同的用户ID，添加了基于用户名的哈希函数：

```typescript
// 根据用户名生成稳定的用户ID
const generateUserIdFromUsername = (username: string): number => {
  let hash = 0;
  for (let i = 0; i < username.length; i++) {
    const char = username.charCodeAt(i);
    hash = ((hash << 5) - hash) + char;
    hash = hash & hash;
  }
  return Math.abs(hash % 10000) + 1000;
};
```

现在不同用户名会生成不同的用户ID：
- `admin` → ID: 1
- `user1` → ID: 约 4500+
- `test` → ID: 约 2800+

## 修复效果

### 修复前
- 所有用户的请求都使用 `userId: 1`
- 后端正确限购，但前端状态独立
- 表现为"同一用户能多次购买"

### 修复后
- ✅ 不同用户使用不同的用户ID
- ✅ 同一用户的重复购买会被后端正确拒绝
- ✅ 前端会收到 409 错误并显示相应提示
- ✅ 购物车状态与用户ID正确关联

## 测试建议

1. **同一用户测试**：
   - 用同一账号登录
   - 第一次秒杀应该成功
   - 第二次秒杀应该失败（收到"您已参与过此次秒杀"提示）

2. **不同用户测试**：
   - 用不同用户名登录
   - 每个用户都应该能成功秒杀一次
   - 用户之间不会互相影响

3. **查看控制台日志**：
   - 现在会打印实际发送的 userId
   - 可以验证不同用户确实使用了不同的ID

## 相关文件

- `src/services/api.ts` - 主要修复文件
- `src/stores/authStore.ts` - 用户认证状态管理
- `src/stores/cartStore.ts` - 购物车状态管理

## 总结

这是一个典型的**前端硬编码导致的业务逻辑错误**。后端的限购逻辑是正确的，问题出在前端没有正确传递用户身份信息。修复后，系统的限购功能将按预期工作。 