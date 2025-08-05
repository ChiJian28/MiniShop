import React from 'react';
import { Navigate, useLocation } from 'react-router-dom';
import { useIsAuthenticated, useAuthLoading } from '../stores/authStore';
import LoadingSpinner from './LoadingSpinner';

interface ProtectedRouteProps {
  children: React.ReactNode;
  requireAuth?: boolean;
}

const ProtectedRoute: React.FC<ProtectedRouteProps> = ({ 
  children, 
  requireAuth = true 
}) => {
  const isAuthenticated = useIsAuthenticated();
  const loading = useAuthLoading();
  const location = useLocation();

  console.log('ProtectedRoute:', { 
    requireAuth, 
    isAuthenticated, 
    loading, 
    pathname: location.pathname 
  });

  // 如果正在加载认证状态，显示加载动画
  if (loading) {
    console.log('ProtectedRoute: showing loading screen');
    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-orange-50 to-red-50">
        <LoadingSpinner size="lg" message="正在验证身份..." />
      </div>
    );
  }

  // 如果需要认证但用户未登录，重定向到登录页
  if (requireAuth && !isAuthenticated) {
    console.log('ProtectedRoute: redirecting to login');
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  // 如果不需要认证但用户已登录，重定向到秒杀页
  if (!requireAuth && isAuthenticated) {
    console.log('ProtectedRoute: redirecting to seckill');
    return <Navigate to="/seckill" replace />;
  }

  console.log('ProtectedRoute: rendering children');
  return <>{children}</>;
};

export default ProtectedRoute; 