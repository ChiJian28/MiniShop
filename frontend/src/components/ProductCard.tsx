import React, { useState } from 'react';
import { Product, SeckillStatus } from '../types';
import CountdownTimer from './CountdownTimer';
import { useSeckillPurchase } from '../hooks/useSeckill';

interface ProductCardProps {
  product: Product;
  onSeckillComplete?: (status: SeckillStatus) => void;
}

const ProductCard: React.FC<ProductCardProps> = ({ product, onSeckillComplete }) => {
  const [seckillStatus, setSeckillStatus] = useState<SeckillStatus | null>(null);
  const [isActive, setIsActive] = useState(product.status === 'active');
  const seckillMutation = useSeckillPurchase();

  const handleSeckill = async () => {
    if (seckillMutation.isPending || product.status !== 'active') return;

    setSeckillStatus(null);

    try {
      const response = await seckillMutation.mutateAsync(product.id);
      
      const status: SeckillStatus = {
        success: response.code === 0,
        message: response.message,
        type: response.code === 0 ? 'success' : 
              response.data?.reason === 'sold_out' ? 'soldout' : 'error'
      };

      setSeckillStatus(status);
      onSeckillComplete?.(status);

    } catch (error: any) {
      const errorResponse = error.response?.data;
      const status: SeckillStatus = {
        success: false,
        message: errorResponse?.message || '网络错误，请重试',
        type: errorResponse?.data?.reason === 'sold_out' ? 'soldout' : 'error'
      };

      setSeckillStatus(status);
      onSeckillComplete?.(status);
    }
  };

  const handleCountdownComplete = () => {
    setIsActive(true);
  };

  const getStatusMessage = () => {
    if (!seckillStatus) return null;

    const statusConfig = {
      success: {
        text: seckillStatus.message,
        className: 'text-green-600 bg-green-50 border-green-200',
        icon: '🎉'
      },
      error: {
        text: seckillStatus.message,
        className: 'text-red-600 bg-red-50 border-red-200',
        icon: '❌'
      },
      soldout: {
        text: '商品已抢光，下次要快一点哦~',
        className: 'text-orange-600 bg-orange-50 border-orange-200',
        icon: '😢'
      }
    };

    const config = statusConfig[seckillStatus.type as keyof typeof statusConfig];
    
    return (
      <div className={`p-3 rounded-lg border text-center font-medium ${config.className}`}>
        <span className="mr-2">{config.icon}</span>
        {config.text}
      </div>
    );
  };

  const getButtonText = () => {
    if (seckillMutation.isPending) return '';
    if (product.status === 'waiting') return '等待开始';
    if (product.status === 'ended') return '活动已结束';
    if (seckillStatus?.success) return '秒杀成功';
    return '立即秒杀';
  };

  const isButtonDisabled = () => {
    return seckillMutation.isPending || 
           product.status !== 'active' || 
           !isActive || 
           seckillStatus?.success;
  };

  return (
    <div className="product-card max-w-md mx-auto">
      {/* Product Image */}
      <div className="relative">
        <img
          src={product.imageUrl}
          alt={product.productName}
          className="w-full h-64 object-cover"
          onError={(e) => {
            (e.target as HTMLImageElement).src = 'https://via.placeholder.com/400x400/f0f0f0/666?text=商品图片';
          }}
        />
        {product.status === 'waiting' && (
          <div className="absolute top-4 left-4 bg-yellow-500 text-white px-3 py-1 rounded-full text-sm font-medium">
            即将开始
          </div>
        )}
        {product.status === 'active' && (
          <div className="absolute top-4 left-4 bg-red-500 text-white px-3 py-1 rounded-full text-sm font-medium animate-pulse">
            🔥 进行中
          </div>
        )}
        {product.status === 'ended' && (
          <div className="absolute top-4 left-4 bg-gray-500 text-white px-3 py-1 rounded-full text-sm font-medium">
            已结束
          </div>
        )}
      </div>

      {/* Product Info */}
      <div className="p-6">
        <h2 className="text-lg font-bold text-gray-900 mb-3 line-clamp-2">
          {product.productName}
        </h2>

        {/* Price */}
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center space-x-3">
            <span className="price-seckill">
              ¥{product.seckillPrice.toLocaleString()}
            </span>
            <span className="price-original">
              ¥{product.originalPrice.toLocaleString()}
            </span>
          </div>
          <div className="text-right">
            <div className="text-sm text-gray-500">节省</div>
            <div className="text-red-500 font-bold">
              ¥{(product.originalPrice - product.seckillPrice).toLocaleString()}
            </div>
          </div>
        </div>

        {/* Stock Info */}
        <div className="stock-info text-center mb-4">
          <span>剩余库存：{product.stock} 件</span>
          {product.stock < 20 && (
            <span className="ml-2 text-red-500 animate-flash">库存紧张！</span>
          )}
        </div>

        {/* Countdown Timer */}
        {product.status === 'waiting' && (
          <CountdownTimer
            targetTime={product.startTime}
            onComplete={handleCountdownComplete}
            className="mb-6"
          />
        )}

        {product.status === 'active' && (
          <CountdownTimer
            targetTime={product.endTime}
            prefix="距离秒杀结束还有："
            className="mb-6"
          />
        )}

        {/* Seckill Button */}
        <button
          onClick={handleSeckill}
          disabled={isButtonDisabled()}
          className="seckill-button w-full text-lg font-bold relative"
        >
          {seckillMutation.isPending && (
            <div className="loading-spinner mr-2" />
          )}
          {getButtonText()}
          {seckillMutation.isPending && (
            <span className="ml-2">处理中...</span>
          )}
        </button>

        {/* Status Message */}
        {seckillStatus && (
          <div className="mt-4">
            {getStatusMessage()}
          </div>
        )}

        {/* Additional Info */}
        <div className="mt-4 text-xs text-gray-500 text-center space-y-1">
          <div>✓ 正品保障</div>
          <div>✓ 7天无理由退货</div>
          <div>✓ 全国包邮</div>
        </div>
      </div>
    </div>
  );
};

export default ProductCard; 