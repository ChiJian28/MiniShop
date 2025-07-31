import React, { useState, useEffect } from 'react';
import { Product, SeckillStatus } from '../types';
import Header from '../components/Header';
import ProductCard from '../components/ProductCard';
import LoadingSpinner from '../components/LoadingSpinner';
import { seckillApi } from '../services/api';

const SeckillPage: React.FC = () => {
  const [product, setProduct] = useState<Product | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [cartCount, setCartCount] = useState(0);

  // 从URL获取产品ID，默认为123
  const getProductId = (): number => {
    const pathParts = window.location.pathname.split('/');
    const idFromPath = pathParts[pathParts.length - 1];
    return parseInt(idFromPath) || 123;
  };

  const productId = getProductId();

  useEffect(() => {
    loadProduct();
  }, [productId]);

  const loadProduct = async () => {
    try {
      setLoading(true);
      setError(null);
      
      const productData = await seckillApi.getProductStatus(productId);
      setProduct(productData);
    } catch (err) {
      console.error('Failed to load product:', err);
      setError('加载商品信息失败，请刷新重试');
    } finally {
      setLoading(false);
    }
  };

  const handleSeckillComplete = (status: SeckillStatus) => {
    if (status.success) {
      // 秒杀成功，增加购物车数量
      setCartCount(prev => prev + 1);
      
      // 可以显示成功提示
      console.log('Seckill successful!', status);
    }
  };

  const handleRetry = () => {
    loadProduct();
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-orange-50 to-red-50">
        <Header cartCount={cartCount} />
        <div className="flex items-center justify-center min-h-[calc(100vh-4rem)]">
          <LoadingSpinner size="lg" message="正在加载商品信息..." />
        </div>
      </div>
    );
  }

  if (error || !product) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-orange-50 to-red-50">
        <Header cartCount={cartCount} />
        <div className="flex items-center justify-center min-h-[calc(100vh-4rem)]">
          <div className="max-w-md w-full bg-white rounded-lg shadow-lg p-6 text-center mx-4">
            <div className="text-6xl mb-4">😕</div>
            <h2 className="text-xl font-bold text-gray-900 mb-2">
              加载失败
            </h2>
            <p className="text-gray-600 mb-4">
              {error || '商品信息加载失败'}
            </p>
            <button
              onClick={handleRetry}
              className="w-full bg-seckill-orange hover:bg-seckill-orange-dark text-white font-medium py-2 px-4 rounded-lg transition-colors"
            >
              重新加载
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-orange-50 to-red-50">
      <Header cartCount={cartCount} />
      
      {/* Hero Section */}
      <div className="bg-gradient-to-r from-seckill-orange to-red-500 text-white py-8">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 text-center">
          <h1 className="text-3xl md:text-4xl font-bold mb-2">
            🔥 限时秒杀 🔥
          </h1>
          <p className="text-lg opacity-90">
            超低价格，限量抢购，手慢无！
          </p>
        </div>
      </div>

      {/* Main Content */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="flex justify-center">
          <ProductCard 
            product={product} 
            onSeckillComplete={handleSeckillComplete}
          />
        </div>
      </div>

      {/* Features Section */}
      <div className="bg-white py-12">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-8">
            <h2 className="text-2xl font-bold text-gray-900 mb-2">
              为什么选择我们？
            </h2>
            <p className="text-gray-600">
              MiniShop 秒杀，品质保障，服务一流
            </p>
          </div>
          
          <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
            <div className="text-center">
              <div className="w-16 h-16 bg-seckill-orange bg-opacity-10 rounded-full flex items-center justify-center mx-auto mb-4">
                <svg className="w-8 h-8 text-seckill-orange" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                </svg>
              </div>
              <h3 className="text-lg font-semibold text-gray-900 mb-2">正品保障</h3>
              <p className="text-gray-600">所有商品均为正品，假一赔十</p>
            </div>
            
            <div className="text-center">
              <div className="w-16 h-16 bg-seckill-orange bg-opacity-10 rounded-full flex items-center justify-center mx-auto mb-4">
                <svg className="w-8 h-8 text-seckill-orange" fill="currentColor" viewBox="0 0 20 20">
                  <path d="M3 4a1 1 0 011-1h12a1 1 0 011 1v2a1 1 0 01-1 1H4a1 1 0 01-1-1V4zM3 10a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H4a1 1 0 01-1-1v-6zM14 9a1 1 0 00-1 1v6a1 1 0 001 1h2a1 1 0 001-1v-6a1 1 0 00-1-1h-2z" />
                </svg>
              </div>
              <h3 className="text-lg font-semibold text-gray-900 mb-2">极速发货</h3>
              <p className="text-gray-600">下单后24小时内发货</p>
            </div>
            
            <div className="text-center">
              <div className="w-16 h-16 bg-seckill-orange bg-opacity-10 rounded-full flex items-center justify-center mx-auto mb-4">
                <svg className="w-8 h-8 text-seckill-orange" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M11.3 1.046A1 1 0 0112 2v5h4a1 1 0 01.82 1.573l-7 10A1 1 0 018 18v-5H4a1 1 0 01-.82-1.573l7-10a1 1 0 011.12-.38z" clipRule="evenodd" />
                </svg>
              </div>
              <h3 className="text-lg font-semibold text-gray-900 mb-2">售后无忧</h3>
              <p className="text-gray-600">7天无理由退货，30天换新</p>
            </div>
          </div>
        </div>
      </div>

      {/* Footer */}
      <footer className="bg-gray-800 text-white py-8">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 text-center">
          <p className="text-gray-400">
            © 2024 MiniShop. All rights reserved. | 秒杀系统演示
          </p>
        </div>
      </footer>
    </div>
  );
};

export default SeckillPage; 