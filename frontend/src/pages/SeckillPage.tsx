import React from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { SeckillStatus } from '../types';
import Header from '../components/Header';
import ProductCard from '../components/ProductCard';
import LoadingSpinner from '../components/LoadingSpinner';
import { useCartTotalCount } from '../stores/cartStore';
import { useProductStatus } from '../hooks/useSeckill';
import { seckillApi } from '../services/api';

const SeckillPage: React.FC = () => {
  const cartCount = useCartTotalCount();
  const navigate = useNavigate();
  const { productId: paramProductId } = useParams<{ productId?: string }>();

  // 如果有产品ID参数，显示单个商品；否则显示所有商品
  const showSingleProduct = !!paramProductId;
  const productId = paramProductId ? parseInt(paramProductId) : 1001;
  
  // 获取单个商品数据
  const { 
    data: product, 
    isLoading: singleLoading, 
    error: singleError, 
    refetch: loadProduct 
  } = useProductStatus(productId, showSingleProduct ? { enabled: true } : { enabled: false });

  // 获取所有商品数据
  const { 
    data: products = [], 
    isLoading: listLoading, 
    error: listError,
    refetch: loadProducts 
  } = useQuery({
    queryKey: ['productList'],
    queryFn: seckillApi.getProductList,
    refetchInterval: 30000, // 30秒自动刷新
    enabled: !showSingleProduct
  });

  const loading = showSingleProduct ? singleLoading : listLoading;
  const error = showSingleProduct ? singleError : listError;

  const handleSeckillComplete = (status: SeckillStatus) => {
    if (status.success) {
      // 秒杀成功，React Query hook 会自动处理购物车逻辑
      console.log('Seckill successful!', status);
    }
  };

  const handleRetry = () => {
    if (showSingleProduct) {
      loadProduct();
    } else {
      loadProducts();
    }
  };

  const handleProductClick = (clickedProductId: number) => {
    navigate(`/seckill/${clickedProductId}`);
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-orange-50 to-red-50">
        <Header cartCount={cartCount} />
        <div className="flex items-center justify-center min-h-[calc(100vh-4rem)]">
          <LoadingSpinner size="lg" message={showSingleProduct ? "正在加载商品信息..." : "正在加载商品列表..."} />
        </div>
      </div>
    );
  }

  if (error || (showSingleProduct && !product) || (!showSingleProduct && products.length === 0)) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-orange-50 to-red-50">
        <Header cartCount={cartCount} />
        <div className="flex items-center justify-center min-h-[calc(100vh-4rem)]">
          <div className="text-center">
            <div className="text-red-500 text-lg mb-4">
              {error ? '加载失败' : showSingleProduct ? '商品不存在' : '暂无商品'}
            </div>
            <div className="space-x-4">
              <button 
                onClick={handleRetry}
                className="seckill-button"
              >
                重试
              </button>
              {showSingleProduct && (
                <button 
                  onClick={() => navigate('/seckill')}
                  className="bg-gray-500 hover:bg-gray-600 text-white font-bold py-2 px-6 rounded-lg transition-colors"
                >
                  返回商品列表
                </button>
              )}
            </div>
          </div>
        </div>
      </div>
    );
  }

  // 显示单个商品详情
  if (showSingleProduct && product) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-orange-50 to-red-50">
        <Header cartCount={cartCount} />
        
        <div className="container mx-auto px-4 py-8">
          {/* 返回按钮 */}
          <div className="mb-6">
            <button 
              onClick={() => navigate('/seckill')}
              className="flex items-center text-gray-600 hover:text-seckill-orange transition-colors"
            >
              <svg className="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
              </svg>
              返回商品列表
            </button>
          </div>

          <div className="max-w-2xl mx-auto">
            <ProductCard 
              product={product} 
              onSeckillComplete={handleSeckillComplete}
            />
          </div>
        </div>
      </div>
    );
  }

  // 显示所有商品列表
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
          {products.map((productItem) => (
            <div
              key={productItem.id}
              className="cursor-pointer"
              onClick={() => handleProductClick(productItem.id)}
            >
              <ProductCard 
                product={productItem} 
                onSeckillComplete={handleSeckillComplete}
              />
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

export default SeckillPage;