import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { seckillApi } from '../services/api';
import { useCartStore } from '../stores/cartStore';
import { Product, SeckillStatus } from '../types';

// 查询键
export const seckillKeys = {
  all: ['seckill'] as const,
  products: () => [...seckillKeys.all, 'products'] as const,
  product: (id: number) => [...seckillKeys.products(), id] as const,
  productStatus: (id: number) => [...seckillKeys.product(id), 'status'] as const,
};

// 获取商品状态
export const useProductStatus = (productId: number) => {
  return useQuery({
    queryKey: seckillKeys.productStatus(productId),
    queryFn: () => seckillApi.getProductStatus(productId),
    enabled: !!productId,
    // 秒杀商品状态变化较快，设置较短的缓存时间
    staleTime: 30 * 1000, // 30秒
    gcTime: 60 * 1000, // 1分钟
    // 每5秒自动刷新
    refetchInterval: 5000,
    // 窗口聚焦时刷新
    refetchOnWindowFocus: true,
  });
};

// 参与秒杀
export const useSeckillPurchase = () => {
  const queryClient = useQueryClient();
  const addToCart = useCartStore((state) => state.addItem);

  return useMutation({
    mutationFn: (productId: number) => seckillApi.participateSeckill(productId),
    
    onSuccess: (data, productId) => {
      if (data.code === 0) {
        // 秒杀成功，添加到购物车
        const product = queryClient.getQueryData<Product>(
          seckillKeys.productStatus(productId)
        );
        
        if (product) {
          addToCart({
            id: product.id,
            productName: product.productName,
            price: product.seckillPrice,
            imageUrl: product.imageUrl,
          });
        }
        
        // 刷新商品状态
        queryClient.invalidateQueries({
          queryKey: seckillKeys.productStatus(productId)
        });
        
        console.log('Seckill successful via React Query:', data);
      }
    },
    
    onError: (error, productId) => {
      console.error('Seckill failed via React Query:', error);
      
      // 即使失败也刷新商品状态，以获取最新信息
      queryClient.invalidateQueries({
        queryKey: seckillKeys.productStatus(productId)
      });
    },
  });
};

// 预加载商品数据
export const usePrefetchProduct = () => {
  const queryClient = useQueryClient();
  
  return (productId: number) => {
    queryClient.prefetchQuery({
      queryKey: seckillKeys.productStatus(productId),
      queryFn: () => seckillApi.getProductStatus(productId),
      staleTime: 30 * 1000,
    });
  };
}; 