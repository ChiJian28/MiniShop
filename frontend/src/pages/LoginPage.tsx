import React from 'react';
import LoginForm from '../components/LoginForm';

const LoginPage: React.FC = () => {
  const handleLoginSuccess = (token: string) => {
    console.log('登录成功，token:', token);
    // 可以在这里添加其他登录成功后的逻辑
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-orange-50 to-red-50 flex items-center justify-center px-4">
      {/* 背景装饰 */}
      <div className="absolute inset-0 overflow-hidden">
        <div className="absolute -top-40 -right-40 w-80 h-80 bg-seckill-orange opacity-10 rounded-full"></div>
        <div className="absolute -bottom-40 -left-40 w-80 h-80 bg-red-400 opacity-10 rounded-full"></div>
      </div>

      {/* 主要内容 */}
      <div className="relative z-10 w-full max-w-md">
        {/* 顶部装饰 */}
        <div className="text-center mb-8">
          <div className="inline-flex items-center justify-center w-16 h-16 bg-seckill-orange rounded-full mb-4">
            <svg className="w-8 h-8 text-white" fill="currentColor" viewBox="0 0 20 20">
              <path d="M3 4a1 1 0 011-1h12a1 1 0 011 1v2a1 1 0 01-1 1H4a1 1 0 01-1-1V4zM3 10a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H4a1 1 0 01-1-1v-6zM14 9a1 1 0 00-1 1v6a1 1 0 001 1h2a1 1 0 001-1v-6a1 1 0 00-1-1h-2z" />
            </svg>
          </div>
          <h2 className="text-2xl font-bold text-gray-900 mb-2">
            欢迎回来
          </h2>
          <p className="text-gray-600">
            登录您的账户，开始秒杀之旅
          </p>
        </div>

        {/* 登录表单 */}
        <LoginForm onLoginSuccess={handleLoginSuccess} />

        {/* 底部链接 */}
        <div className="mt-8 text-center">
          <div className="text-sm text-gray-500 space-y-2">
            <p>
              还没有账户？
              <button className="text-seckill-orange hover:text-seckill-orange-dark ml-1 font-medium">
                立即注册
              </button>
            </p>
            <p>
              <button className="text-gray-400 hover:text-gray-600">
                忘记密码？
              </button>
            </p>
          </div>
        </div>

        {/* 功能特色 */}
        <div className="mt-8 bg-white/50 backdrop-blur-sm rounded-lg p-4">
          <h3 className="text-sm font-medium text-gray-700 mb-3 text-center">
            🔥 秒杀系统特色
          </h3>
          <div className="grid grid-cols-2 gap-3 text-xs text-gray-600">
            <div className="flex items-center">
              <span className="w-2 h-2 bg-green-400 rounded-full mr-2"></span>
              高并发处理
            </div>
            <div className="flex items-center">
              <span className="w-2 h-2 bg-blue-400 rounded-full mr-2"></span>
              实时库存
            </div>
            <div className="flex items-center">
              <span className="w-2 h-2 bg-yellow-400 rounded-full mr-2"></span>
              防超卖机制
            </div>
            <div className="flex items-center">
              <span className="w-2 h-2 bg-red-400 rounded-full mr-2"></span>
              限流保护
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default LoginPage; 