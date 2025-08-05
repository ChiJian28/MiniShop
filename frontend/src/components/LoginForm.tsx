import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { seckillApi } from '../services/api';
import { useAuthStore } from '../stores/authStore';
import LoadingSpinner from './LoadingSpinner';

interface LoginFormProps {
  onLoginSuccess?: (token: string) => void;
}

const LoginForm: React.FC<LoginFormProps> = ({ onLoginSuccess }) => {
  const [formData, setFormData] = useState({
    username: '',
    password: '',
  });
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');
  const navigate = useNavigate();
  const login = useAuthStore((state) => state.login);

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setFormData(prev => ({
      ...prev,
      [name]: value,
    }));
    // 清除错误信息
    if (error) setError('');
  };

  const validateForm = (): boolean => {
    if (!formData.username.trim()) {
      setError('请输入用户名');
      return false;
    }
    if (!formData.password.trim()) {
      setError('请输入密码');
      return false;
    }
    if (formData.username.length < 2) {
      setError('用户名至少2个字符');
      return false;
    }
    if (formData.password.length < 3) {
      setError('密码至少3个字符');
      return false;
    }
    return true;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!validateForm()) return;

    setIsLoading(true);
    setError('');

    try {
      console.log('Attempting login with:', { username: formData.username, password: formData.password });
      const response = await seckillApi.login(formData.username, formData.password);
      console.log('Login response:', response);
      
      if (response.code === 0 && response.data?.token) {
        // 使用 AuthContext 的 login 方法更新全局状态
        login(response.data.token, response.data.user);
        console.log('Auth context updated with token:', response.data.token);
        console.log('Auth context updated with user:', response.data.user);

        // 回调通知父组件
        onLoginSuccess?.(response.data.token);

        // 导航到秒杀页面
        console.log('Navigating to /seckill...');
        navigate('/seckill');
      } else {
        console.log('Login failed - invalid response:', response);
        setError(response.message || '登录失败，请重试');
      }
    } catch (error: any) {
      console.error('Login error:', error);
      
      // 处理不同类型的错误
      if (error.response?.status === 401) {
        setError('用户名或密码错误');
      } else if (error.response?.status === 429) {
        setError('登录请求过于频繁，请稍后再试');
      } else if (error.code === 'NETWORK_ERROR') {
        setError('网络连接失败，请检查网络');
      } else {
        setError(error.response?.data?.message || '登录失败，请重试');
      }
    } finally {
      setIsLoading(false);
    }
  };

  const handleDemoLogin = () => {
    setFormData({
      username: 'admin',
      password: 'password',
    });
    setError('');
  };

  return (
    <div className="w-full max-w-md mx-auto">
      <form onSubmit={handleSubmit} className="bg-white shadow-xl rounded-2xl px-8 pt-8 pb-6">
        {/* Logo 区域 */}
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold text-seckill-orange mb-2">
            MiniShop
          </h1>
          <p className="text-gray-600">登录参与秒杀</p>
        </div>

        {/* 错误提示 */}
        {error && (
          <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg">
            <div className="flex items-center">
              <svg className="w-5 h-5 text-red-500 mr-2" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
              </svg>
              <span className="text-red-700 text-sm">{error}</span>
            </div>
          </div>
        )}

        {/* 用户名输入框 */}
        <div className="mb-4">
          <label className="block text-gray-700 text-sm font-medium mb-2" htmlFor="username">
            用户名
          </label>
          <input
            id="username"
            name="username"
            type="text"
            value={formData.username}
            onChange={handleInputChange}
            className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-seckill-orange focus:border-transparent transition-all duration-200"
            placeholder="请输入用户名"
            disabled={isLoading}
            autoComplete="username"
          />
        </div>

        {/* 密码输入框 */}
        <div className="mb-6">
          <label className="block text-gray-700 text-sm font-medium mb-2" htmlFor="password">
            密码
          </label>
          <input
            id="password"
            name="password"
            type="password"
            value={formData.password}
            onChange={handleInputChange}
            className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-seckill-orange focus:border-transparent transition-all duration-200"
            placeholder="请输入密码"
            disabled={isLoading}
            autoComplete="current-password"
          />
        </div>

        {/* 登录按钮 */}
        <button
          type="submit"
          disabled={isLoading}
          className="seckill-button w-full text-lg font-bold mb-4"
        >
          {isLoading ? (
            <div className="flex items-center justify-center">
              <LoadingSpinner size="sm" color="white" />
              <span className="ml-2">登录中...</span>
            </div>
          ) : (
            '登录'
          )}
        </button>

        {/* 演示账号 */}
        <div className="text-center">
          <button
            type="button"
            onClick={handleDemoLogin}
            disabled={isLoading}
            className="text-sm text-gray-500 hover:text-seckill-orange transition-colors underline"
          >
            使用演示账号 (admin/password)
          </button>
        </div>

        {/* 提示信息 */}
        <div className="mt-6 text-center text-xs text-gray-500 space-y-1">
          <p>✓ 支持演示账号快速体验</p>
          <p>✓ 登录后可参与秒杀活动</p>
          <p>✓ 数据仅用于演示，请放心使用</p>
        </div>
      </form>
    </div>
  );
};

export default LoginForm; 