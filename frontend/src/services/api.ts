import axios from 'axios';
import { Product, SeckillResponse } from '../types';

const API_BASE_URL = process.env.REACT_APP_API_URL || '';

// 创建 axios 实例
const api = axios.create({
  baseURL: API_BASE_URL,
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// 请求拦截器 - 添加认证 token
api.interceptors.request.use(
  (config) => {
    // 这里可以从 localStorage 或其他地方获取真实的 token
    const token = localStorage.getItem('token') || 'fake_token_here';
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// 响应拦截器
api.interceptors.response.use(
  (response) => {
    return response;
  },
  (error) => {
    console.error('API Error:', error);
    return Promise.reject(error);
  }
);

// API 方法
export const seckillApi = {
  // 获取产品状态
  getProductStatus: async (productId: number): Promise<Product> => {
    try {
      const response = await api.get(`/api/v1/seckill/${productId}/status`);
      return response.data.data;
    } catch (error) {
      // 如果后端服务不可用，返回模拟数据
      console.warn('API not available, using mock data');
      return getMockProductData(productId);
    }
  },

  // 参与秒杀
  participateSeckill: async (productId: number): Promise<SeckillResponse> => {
    try {
      const response = await api.post(`/api/v1/seckill/${productId}`, {
        productId,
        userId: 1, // 模拟用户ID
      });
      return response.data;
    } catch (error: any) {
      // 模拟不同的响应结果
      const mockResponses = [
        { code: 0, message: '秒杀成功！', data: { success: true, orderId: 'ORDER_' + Date.now() } },
        { code: 1001, message: '库存不足，秒杀失败', data: { success: false, reason: 'sold_out' } },
        { code: 1002, message: '您已参与过此次秒杀', data: { success: false, reason: 'already_purchased' } },
        { code: 1003, message: '秒杀太火爆了，请稍后再试', data: { success: false, reason: 'too_busy' } },
      ];
      
      const randomResponse = mockResponses[Math.floor(Math.random() * mockResponses.length)];
      
      // 模拟网络延迟
      await new Promise(resolve => setTimeout(resolve, 1000 + Math.random() * 2000));
      
      if (randomResponse.code === 0) {
        return randomResponse;
      } else {
        throw { response: { data: randomResponse } };
      }
    }
  },

  // 用户登录（模拟）
  login: async (username: string, password: string) => {
    try {
      const response = await api.post('/api/v1/auth/login', {
        username,
        password,
      });
      return response.data;
    } catch (error) {
      // 模拟登录逻辑
      console.warn('API not available, using mock login');
      
      // 模拟验证逻辑
      console.log('Mock login attempt:', { username, password });
      
      if (username === 'admin' && (password === 'password' || password === 'pwd')) {
        // 支持两种演示密码
        const result = {
          code: 0,
          data: {
            token: 'mock_jwt_token_' + Date.now(),
            user: { id: 1, username },
          },
          message: '登录成功',
        };
        console.log('Login successful:', result);
        return result;
      } else if (username && password && password.length >= 3) {
        // 其他用户名密码组合也允许登录（演示用），密码至少3位
        const result = {
          code: 0,
          data: {
            token: 'mock_jwt_token_' + Date.now(),
            user: { id: Date.now(), username },
          },
          message: '登录成功',
        };
        console.log('Login successful (generic):', result);
        return result;
      } else {
        console.log('Login failed:', { username, password });
        throw {
          response: {
            status: 401,
            data: {
              code: 401,
              message: '用户名或密码错误',
            }
          }
        };
      }
    }
  },
};

// 模拟产品数据
function getMockProductData(productId: number): Product {
  const now = new Date();
  
  // 可以通过环境变量控制秒杀状态，方便测试
  const showCountdown = process.env.REACT_APP_SHOW_COUNTDOWN === 'true';
  
  let startTime: Date;
  let endTime: Date;
  
  if (showCountdown) {
    // 显示倒计时场景：5秒后开始
    startTime = new Date(now.getTime() + 5000);
    endTime = new Date(startTime.getTime() + 3600000);
  } else {
    // 立即可用场景：已经开始
    startTime = new Date(now.getTime() - 1000);
    endTime = new Date(now.getTime() + 3600000);
  }

  return {
    id: productId,
    productName: 'iPhone 15 Pro Max 256GB 深空黑色',
    imageUrl: 'https://images.unsplash.com/photo-1592750475338-74b7b21085ab?w=400&h=400&fit=crop',
    originalPrice: 9999,
    seckillPrice: 6999,
    stock: Math.floor(Math.random() * 100) + 10,
    startTime: startTime.toISOString(),
    endTime: endTime.toISOString(),
    status: now < startTime ? 'waiting' : now < endTime ? 'active' : 'ended',
  };
}

export default api; 