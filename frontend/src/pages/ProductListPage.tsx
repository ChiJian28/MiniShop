import React from 'react';
import { useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { seckillApi } from '../services/api';
import { Product } from '../types';
import Header from '../components/Header';
import LoadingSpinner from '../components/LoadingSpinner';
import CountdownTimer from '../components/CountdownTimer';
import { useCartTotalCount } from '../stores/cartStore';

const ProductListPage: React.FC = () => {
  const navigate = useNavigate();
  const cartCount = useCartTotalCount();

  // 获取商品列表
  const { 
    data: products = [], 
    isLoading, 
    error,
    refetch 
  } = useQuery({
    queryKey: ['productList'],
    queryFn: seckillApi.getProductList,
    refetchInterval: 30000, // 30秒自动刷新
  });

  const handleProductClick = (productId: number) => {
    navigate(`/seckill/${productId}`);
  };

  const formatPrice = (price: number) => {
    return `¥${price.toLocaleString()}`;
  };

  const getStatusBadge = (product: Product) => {
    switch (product.status) {
      case 'waiting':
        return (
          <div className="bg-yellow-100 text-yellow-800 px-2 py-1 rounded-full text-xs font-medium">
            即将开始
          </div>
        );
      case 'active':
        return (
          <div className="bg-green-100 text-green-800 px-2 py-1 rounded-full text-xs font-medium">
            进行中
          </div>
        );
      case 'ended':
        return (
          <div className="bg-gray-100 text-gray-800 px-2 py-1 rounded-full text-xs font-medium">
            已结束
          </div>
        );
    }
  };

  if (isLoading) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-orange-50 to-red-50">
        <Header cartCount={cartCount} />
        <div className="flex items-center justify-center min-h-[calc(100vh-4rem)]">
          <LoadingSpinner size="lg" message="正在加载商品列表..." />
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-orange-50 to-red-50">
        <Header cartCount={cartCount} />
        <div className="flex items-center justify-center min-h-[calc(100vh-4rem)]">
          <div className="text-center">
            <div className="text-red-500 text-lg mb-4">加载商品列表失败</div>
            <button 
              onClick={() => refetch()}
              className="seckill-button"
            >
              重试
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-orange-50 to-red-50">
      <Header cartCount={cartCount} />
      
      <div className="container mx-auto px-4 py-8">
        {/* 页面标题 */}
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold text-gray-800 mb-2">
            🔥 限时秒杀
          </h1>
          <p className="text-gray-600">
            超值好货，限时抢购！
          </p>
        </div>

        {/* 商品列表 */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {products.map((product) => (
            <div
              key={product.id}
              className="bg-white rounded-2xl shadow-lg overflow-hidden hover:shadow-xl transition-all duration-300 cursor-pointer transform hover:scale-105"
              onClick={() => handleProductClick(product.id)}
            >
              {/* 商品图片 */}
              <div className="relative">
                <img
                  src={product.imageUrl}
                  alt={product.productName}
                  className="w-full h-48 object-cover"
                />
                {/* 状态标签 */}
                <div className="absolute top-3 right-3">
                  {getStatusBadge(product)}
                </div>
                {/* 折扣标签 */}
                <div className="absolute top-3 left-3 bg-seckill-orange text-white px-2 py-1 rounded-full text-xs font-bold">
                  {Math.round((1 - product.seckillPrice / product.originalPrice) * 100)}% OFF
                </div>
              </div>

              {/* 商品信息 */}
              <div className="p-4">
                <h3 className="font-semibold text-gray-800 mb-2 line-clamp-2">
                  {product.productName}
                </h3>
                
                {/* 价格信息 */}
                <div className="flex items-center justify-between mb-3">
                  <div className="flex items-center space-x-2">
                    <span className="text-2xl font-bold text-seckill-orange">
                      {formatPrice(product.seckillPrice)}
                    </span>
                    <span className="text-sm text-gray-500 line-through">
                      {formatPrice(product.originalPrice)}
                    </span>
                  </div>
                </div>

                {/* 库存信息 */}
                <div className="flex items-center justify-between mb-3">
                  <span className="text-sm text-gray-600">
                    剩余: {product.stock} 件
                  </span>
                  <div className="flex items-center">
                    <div className="w-2 h-2 bg-green-500 rounded-full mr-1"></div>
                    <span className="text-xs text-gray-500">库存充足</span>
                  </div>
                </div>

                {/* 倒计时或状态 */}
                {product.status === 'waiting' && (
                  <div className="mb-3">
                    <div className="text-xs text-gray-500 mb-1">距离开始:</div>
                    <CountdownTimer
                      targetTime={product.startTime}
                      onComplete={() => window.location.reload()}
                      className="text-sm font-mono text-seckill-orange"
                    />
                  </div>
                )}

                {product.status === 'active' && (
                  <div className="mb-3">
                    <div className="text-xs text-gray-500 mb-1">距离结束:</div>
                    <CountdownTimer
                      targetTime={product.endTime}
                      onComplete={() => window.location.reload()}
                      className="text-sm font-mono text-red-500"
                    />
                  </div>
                )}

                {/* 操作按钮 */}
                <button
                  className={`w-full py-2 px-4 rounded-lg font-medium transition-all duration-200 ${
                    product.status === 'active'
                      ? 'bg-seckill-orange text-white hover:bg-orange-600 shadow-md hover:shadow-lg'
                      : product.status === 'waiting'
                      ? 'bg-yellow-100 text-yellow-800 cursor-not-allowed'
                      : 'bg-gray-100 text-gray-500 cursor-not-allowed'
                  }`}
                  disabled={product.status !== 'active'}
                >
                  {product.status === 'active' && '立即抢购'}
                  {product.status === 'waiting' && '即将开始'}
                  {product.status === 'ended' && '已结束'}
                </button>
              </div>
            </div>
          ))}
        </div>

        {/* 空状态 */}
        {products.length === 0 && (
          <div className="text-center py-16">
            <div className="text-gray-400 text-lg mb-4">暂无商品</div>
            <button 
              onClick={() => refetch()}
              className="seckill-button"
            >
              刷新列表
            </button>
          </div>
        )}
      </div>
    </div>
  );
};

export default ProductListPage;
