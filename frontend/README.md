# MiniShop 秒杀前端

这是 MiniShop 秒杀系统的前端应用，使用 React + TypeScript + Tailwind CSS 构建的现代化秒杀界面。

## 功能特性

### 🎯 核心功能
- **秒杀界面**: 完整的秒杀商品展示和购买流程
- **倒计时**: 精确到秒的倒计时显示，支持秒杀开始和结束时间
- **实时状态**: 动态显示商品库存、秒杀状态、用户操作结果
- **响应式设计**: 适配桌面端和移动端，完美的用户体验
- **错误处理**: 完善的错误边界和加载状态处理

### 🎨 设计特色
- **淘宝风格**: 采用橙色 (#ff4000) 主题色，模仿淘宝秒杀页面
- **现代UI**: 扁平化设计，圆角卡片，渐变背景
- **动画效果**: 丰富的交互动画，hover 效果，加载动画
- **视觉反馈**: 清晰的成功/失败状态提示

### 📱 技术特性
- **TypeScript**: 完整的类型安全
- **组件化**: 模块化组件设计，易于维护和扩展
- **性能优化**: 懒加载，错误边界，性能监控
- **PWA 支持**: 支持离线使用和安装到桌面

## 技术栈

- **React 18**: 前端框架
- **TypeScript**: 类型安全的 JavaScript
- **Tailwind CSS**: 原子化 CSS 框架
- **Axios**: HTTP 客户端
- **Web Vitals**: 性能监控

## 项目结构

```
frontend/
├── public/                     # 静态资源
│   ├── index.html             # HTML 模板
│   ├── manifest.json          # PWA 配置
│   └── favicon.ico            # 网站图标
├── src/
│   ├── components/            # 可复用组件
│   │   ├── Header.tsx         # 头部导航
│   │   ├── ProductCard.tsx    # 商品卡片
│   │   ├── CountdownTimer.tsx # 倒计时组件
│   │   ├── LoadingSpinner.tsx # 加载动画
│   │   └── ErrorBoundary.tsx  # 错误边界
│   ├── pages/
│   │   └── SeckillPage.tsx    # 秒杀页面
│   ├── services/
│   │   └── api.ts             # API 服务
│   ├── types/
│   │   └── index.ts           # TypeScript 类型定义
│   ├── App.tsx                # 主应用组件
│   ├── index.tsx              # 应用入口
│   └── index.css              # 全局样式
├── scripts/
│   └── start.sh               # 启动脚本
├── package.json               # 项目配置
├── tailwind.config.js         # Tailwind 配置
├── tsconfig.json              # TypeScript 配置
└── README.md                  # 项目文档
```

## 快速开始

### 环境要求
- Node.js >= 16.0.0
- npm >= 8.0.0

### 安装和运行

1. **使用启动脚本（推荐）**
```bash
cd frontend
./scripts/start.sh
```

2. **手动启动**
```bash
cd frontend

# 安装依赖
npm install

# 启动开发服务器
npm start
```

3. **访问应用**
- 本地访问: http://localhost:3000
- 网络访问: http://[your-ip]:3000

### 🔐 登录功能

应用现在包含完整的登录系统：

- **路由管理**: 使用 React Router DOM 进行页面导航
- **认证状态**: Context API 管理全局登录状态
- **路由保护**: 未登录用户自动重定向到登录页
- **持久化**: Token 存储在 localStorage，刷新页面保持登录状态

**测试账号**:
- 演示账号: `admin` / `password`
- 任意账号: 任何非空用户名密码都可以登录（演示模式）

**路由说明**:
- `/` - 重定向到 `/seckill`
- `/login` - 登录页面（已登录用户会重定向到 `/seckill`）
- `/seckill` - 秒杀页面（未登录用户会重定向到 `/login`）

**登录测试**:
```bash
# 使用专门的登录测试脚本
./scripts/test-login.sh

# 或直接打开静态测试页面
open test-login.html
```

### 其他命令

```bash
# 构建生产版本
npm run build

# 运行测试
npm test

# 分析构建包大小
npm run build && npx serve -s build
```

## 组件说明

### 1. Header 组件
**文件**: `src/components/Header.tsx`

导航头部组件，包含：
- MiniShop Logo
- 导航菜单
- 购物车图标（带数量提示）
- 用户头像

```tsx
<Header cartCount={cartCount} />
```

### 2. ProductCard 组件
**文件**: `src/components/ProductCard.tsx`

核心的商品卡片组件，功能包括：
- 商品图片展示
- 价格对比（原价/秒杀价）
- 库存信息
- 倒计时显示
- 秒杀按钮
- 状态反馈

```tsx
<ProductCard 
  product={product} 
  onSeckillComplete={handleSeckillComplete}
/>
```

### 3. CountdownTimer 组件
**文件**: `src/components/CountdownTimer.tsx`

倒计时组件，支持：
- 天、时、分、秒显示
- 自定义前缀文字
- 倒计时结束回调
- 动画效果

```tsx
<CountdownTimer
  targetTime={product.startTime}
  onComplete={handleCountdownComplete}
  prefix="距离秒杀开始还有："
/>
```

### 4. LoadingSpinner 组件
**文件**: `src/components/LoadingSpinner.tsx`

加载动画组件：
- 多种尺寸选择
- 多种颜色主题
- 可自定义加载文字

```tsx
<LoadingSpinner 
  size="lg" 
  color="orange" 
  message="正在加载商品信息..." 
/>
```

### 5. ErrorBoundary 组件
**文件**: `src/components/ErrorBoundary.tsx`

错误边界组件：
- 捕获 React 组件错误
- 友好的错误页面
- 开发模式下显示错误详情
- 提供刷新和返回操作

## API 集成

### API 服务
**文件**: `src/services/api.ts`

封装了与后端 API 的交互：

```typescript
// 获取商品状态
const product = await seckillApi.getProductStatus(productId);

// 参与秒杀
const result = await seckillApi.participateSeckill(productId);

// 用户登录
const loginResult = await seckillApi.login(username, password);
```

### 模拟数据
当后端服务不可用时，前端会自动使用模拟数据：
- 随机生成商品信息
- 模拟不同的秒杀结果
- 模拟网络延迟

### 请求拦截器
- 自动添加 JWT Token
- 统一错误处理
- 请求超时处理

## 样式设计

### Tailwind CSS 配置
**文件**: `tailwind.config.js`

自定义了秒杀主题：
```javascript
colors: {
  'seckill-orange': '#ff4000',
  'seckill-orange-dark': '#e63600',
  'seckill-orange-light': '#ff6633',
}
```

### 自定义样式
**文件**: `src/index.css`

定义了常用的组件样式：
- `.seckill-button`: 秒杀按钮样式
- `.product-card`: 商品卡片样式
- `.countdown-digit`: 倒计时数字样式
- `.loading-spinner`: 加载动画样式

### 响应式设计
- 移动端优先设计
- 断点适配：sm, md, lg, xl
- 灵活的网格布局

## 状态管理

### 本地状态
使用 React Hooks 管理组件状态：
- `useState`: 组件状态
- `useEffect`: 副作用处理
- 自定义 Hook: 可复用逻辑

### 状态类型
```typescript
interface Product {
  id: number;
  productName: string;
  imageUrl: string;
  originalPrice: number;
  seckillPrice: number;
  stock: number;
  startTime: string;
  endTime: string;
  status: 'waiting' | 'active' | 'ended';
}

interface SeckillStatus {
  success: boolean;
  message: string;
  type: 'success' | 'error' | 'soldout' | 'waiting' | 'ended';
}
```

## 用户体验

### 加载状态
- 页面加载时显示骨架屏
- 按钮点击时显示加载动画
- 网络请求时显示进度提示

### 错误处理
- 网络错误友好提示
- 重试机制
- 错误边界保护

### 性能优化
- 图片懒加载
- 组件懒加载
- 防抖和节流
- 内存泄漏防护

## 部署指南

### 开发环境
```bash
npm start
```
- 热重载
- 开发工具
- 错误提示

### 生产构建
```bash
npm run build
```
生成优化后的静态文件到 `build/` 目录

### 部署选项

1. **静态托管**
   - Netlify
   - Vercel
   - GitHub Pages

2. **CDN 部署**
   - AWS CloudFront
   - 阿里云 CDN

3. **Docker 部署**
```dockerfile
FROM nginx:alpine
COPY build/ /usr/share/nginx/html/
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

## 浏览器支持

- Chrome >= 88
- Firefox >= 85
- Safari >= 14
- Edge >= 88

## 开发指南

### 添加新组件
1. 在 `src/components/` 创建组件文件
2. 使用 TypeScript 定义 Props 接口
3. 添加 Tailwind CSS 样式
4. 导出组件供其他地方使用

### 添加新页面
1. 在 `src/pages/` 创建页面组件
2. 配置路由（如果使用 React Router）
3. 添加页面特定的状态管理

### 样式规范
- 使用 Tailwind CSS 原子类
- 自定义样式放在 `@layer components`
- 保持响应式设计
- 遵循设计系统颜色规范

### 代码规范
- 使用 TypeScript 严格模式
- 组件 Props 必须定义接口
- 使用 ESLint 和 Prettier
- 编写有意义的注释

## 测试

### 单元测试
```bash
npm test
```

### 端到端测试
可以集成 Cypress 或 Playwright：
```bash
npm run e2e
```

### 性能测试
使用 Web Vitals 监控：
- LCP (Largest Contentful Paint)
- FID (First Input Delay)
- CLS (Cumulative Layout Shift)

## 故障排除

### 常见问题

1. **API 请求失败**
   - 检查后端服务是否启动
   - 确认 API 地址配置正确
   - 查看浏览器控制台错误

2. **样式不生效**
   - 确认 Tailwind CSS 正确安装
   - 检查 PostCSS 配置
   - 清除浏览器缓存

3. **倒计时不准确**
   - 检查系统时间
   - 确认时区设置
   - 验证服务器时间同步

### 调试技巧
- 使用 React Developer Tools
- 启用 Redux DevTools（如果使用）
- 查看 Network 面板
- 使用 console.log 调试

## 更新日志

- **v1.0.0**: 初始版本
  - 基础秒杀界面
  - 倒计时功能
  - API 集成

- **v1.1.0**: 功能增强
  - 添加加载状态
  - 改进错误处理
  - 优化移动端体验

- **v1.2.0**: 性能优化
  - 组件懒加载
  - 图片优化
  - 缓存策略

## 贡献指南

1. Fork 项目
2. 创建功能分支
3. 提交代码
4. 创建 Pull Request

## 许可证

MIT License

---

MiniShop 秒杀前端 - 现代化的秒杀购物体验！🛒✨ 