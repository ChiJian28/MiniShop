import React from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider } from './contexts/AuthContext';
import LoginPage from './pages/LoginPage';
import SeckillPage from './pages/SeckillPage';
import ProtectedRoute from './components/ProtectedRoute';
import ErrorBoundary from './components/ErrorBoundary';
import './index.css';

function App() {
  return (
    <ErrorBoundary>
      <AuthProvider>
        <Router>
          <div className="App">
            <Routes>
              {/* 默认路由重定向到秒杀页面 */}
              <Route path="/" element={<Navigate to="/seckill" replace />} />
              
              {/* 登录页面 - 已登录用户会被重定向到秒杀页面 */}
              <Route 
                path="/login" 
                element={
                  <ProtectedRoute requireAuth={false}>
                    <LoginPage />
                  </ProtectedRoute>
                } 
              />
              
              {/* 秒杀页面 - 需要登录 */}
              <Route 
                path="/seckill" 
                element={
                  <ProtectedRoute requireAuth={true}>
                    <SeckillPage />
                  </ProtectedRoute>
                } 
              />
              
              {/* 支持产品ID参数的秒杀页面 */}
              <Route 
                path="/seckill/:productId" 
                element={
                  <ProtectedRoute requireAuth={true}>
                    <SeckillPage />
                  </ProtectedRoute>
                } 
              />
              
              {/* 404 页面 */}
              <Route 
                path="*" 
                element={
                  <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-orange-50 to-red-50">
                    <div className="text-center">
                      <h1 className="text-6xl font-bold text-gray-300 mb-4">404</h1>
                      <p className="text-gray-600 mb-4">页面未找到</p>
                      <button 
                        onClick={() => window.location.href = '/seckill'}
                        className="seckill-button"
                      >
                        返回首页
                      </button>
                    </div>
                  </div>
                } 
              />
            </Routes>
          </div>
        </Router>
      </AuthProvider>
    </ErrorBoundary>
  );
}

export default App; 