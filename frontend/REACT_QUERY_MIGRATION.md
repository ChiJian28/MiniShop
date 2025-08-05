# React Query (TanStack Query) 迁移文档

## 概述
已成功将前端的数据获取方式从直接使用 axios 迁移到 TanStack Query (React Query)，提供了更好的缓存、错误处理、加载状态管理和用户体验。

## 主要更改

### 1. 安装依赖
```bash
npm install @tanstack/react-query @tanstack/react-query-devtools
```

### 2. 创建的新文件

#### `src/lib/queryClient.ts`
- 配置 React Query 客户端
- 设置缓存策略、重试逻辑和错误处理

#### `src/hooks/useAuth.ts`
- 认证相关的 React Query hooks
- `useLogin()` - 登录 mutation
- `useLogout()` - 登出 mutation

#### `src/hooks/useSeckill.ts`
- 秒杀相关的 React Query hooks
- `useProductStatus()` - 获取商品状态 query
- `useSeckillPurchase()` - 秒杀购买 mutation
- `usePrefetchProduct()` - 预加载商品数据

### 3. 更新的组件

#### `src/App.tsx`
- 添加了 `QueryClientProvider` 包装整个应用
- 集成了 React Query 开发工具

#### `src/components/LoginForm.tsx`
- 使用 `useLogin()` hook 替换直接的 API 调用
- 利用 mutation 的 `isPending` 状态管理加载状态

#### `src/pages/SeckillPage.tsx`
- 使用 `useProductStatus()` hook 获取商品数据
- 自动处理加载状态、错误状态和数据刷新

#### `src/components/ProductCard.tsx`
- 使用 `useSeckillPurchase()` hook 处理秒杀请求
- 利用 mutation 状态管理按钮状态

## React Query 的优势

### 1. 自动缓存管理
- **智能缓存**: 数据自动缓存，避免重复请求
- **缓存失效**: 通过 `invalidateQueries` 精确控制缓存更新
- **后台更新**: 数据过期时自动在后台刷新

### 2. 加载状态管理
- **统一状态**: `isLoading`, `isPending`, `isError` 等状态自动管理
- **细粒度控制**: 区分初次加载和后台刷新状态
- **错误边界**: 统一的错误处理机制

### 3. 用户体验优化
- **乐观更新**: 可以在请求完成前更新 UI
- **重试机制**: 失败请求自动重试，可配置重试策略
- **实时同步**: 窗口聚焦时自动刷新数据

### 4. 开发体验
- **DevTools**: 强大的开发工具，可视化查询状态
- **TypeScript**: 完整的 TypeScript 支持
- **React 集成**: 与 React 生命周期完美集成

## 配置详解

### Query Client 配置
```typescript
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000,     // 5分钟内数据保持新鲜
      gcTime: 10 * 60 * 1000,       // 10分钟后清理缓存
      retry: (failureCount, error) => {
        // 认证错误不重试，其他错误最多重试2次
        if (error?.response?.status === 401) return false;
        return failureCount < 2;
      },
      refetchOnWindowFocus: true,    // 窗口聚焦时刷新
    },
  },
});
```

### 查询键策略
```typescript
export const seckillKeys = {
  all: ['seckill'] as const,
  products: () => [...seckillKeys.all, 'products'] as const,
  product: (id: number) => [...seckillKeys.products(), id] as const,
  productStatus: (id: number) => [...seckillKeys.product(id), 'status'] as const,
};
```

## 使用示例

### 查询数据
```typescript
const { data: product, isLoading, error, refetch } = useProductStatus(productId);
```

### 变更数据
```typescript
const loginMutation = useLogin();

const handleLogin = async () => {
  try {
    const result = await loginMutation.mutateAsync({
      username,
      password,
    });
    // 处理成功结果
  } catch (error) {
    // 处理错误
  }
};
```

### 缓存管理
```typescript
// 刷新特定查询
queryClient.invalidateQueries({
  queryKey: seckillKeys.productStatus(productId)
});

// 清除所有缓存
queryClient.clear();
```

## 特殊功能

### 1. 自动刷新
商品状态每5秒自动刷新，确保用户看到最新的库存和状态信息：
```typescript
refetchInterval: 5000,
refetchOnWindowFocus: true,
```

### 2. 预加载
可以预加载商品数据，提升用户体验：
```typescript
const prefetchProduct = usePrefetchProduct();
prefetchProduct(productId);
```

### 3. 乐观更新
秒杀成功后立即更新购物车，无需等待服务器响应：
```typescript
onSuccess: (data, productId) => {
  // 立即添加到购物车
  addToCart(productInfo);
  // 然后刷新商品状态
  queryClient.invalidateQueries({
    queryKey: seckillKeys.productStatus(productId)
  });
}
```

## 开发工具

### React Query DevTools
在开发环境中可以使用 DevTools 查看：
- 所有查询的状态
- 缓存数据
- 查询执行时间线
- 错误信息

访问应用时，点击右下角的 React Query 图标即可打开 DevTools。

## 性能对比

### 迁移前
- 每次组件挂载都会发起新的 API 请求
- 手动管理加载状态和错误状态
- 无缓存机制，重复请求浪费资源
- 错误处理分散在各个组件中

### 迁移后
- ✅ 智能缓存，避免重复请求
- ✅ 自动管理加载和错误状态
- ✅ 后台自动刷新，数据始终最新
- ✅ 统一的错误处理和重试机制
- ✅ 更好的用户体验（加载状态、错误恢复）

## 后续优化建议

1. **添加更多查询**: 为用户信息、订单历史等添加 React Query hooks
2. **离线支持**: 使用 React Query 的离线功能
3. **无限滚动**: 利用 `useInfiniteQuery` 实现商品列表的无限滚动
4. **实时更新**: 结合 WebSocket 实现实时数据更新
5. **错误边界**: 添加全局错误边界处理查询错误

## 总结

React Query 迁移显著提升了应用的性能和用户体验：
- **开发效率**: 减少了大量样板代码
- **用户体验**: 更快的响应速度和更好的加载状态
- **可维护性**: 统一的数据获取和状态管理模式
- **扩展性**: 易于添加新的查询和变更操作 