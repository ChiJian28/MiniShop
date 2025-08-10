export interface Product {
  id: number;
  productName: string;
  imageUrl: string;
  originalPrice: number;
  seckillPrice: number;
  stock: number;
  startTime: string;
  endTime: string;
  status: 'waiting' | 'active' | 'ended';
}

export interface SeckillResponse {
  code: number;
  message: string;
  data?: {
    success: boolean;
    orderId?: string;
    reason?: string;
  };
}

export interface SeckillStatus {
  success: boolean;
  message: string;
  type: 'success' | 'error' | 'soldout' | 'already_purchased' | 'waiting' | 'ended';
}

export interface CountdownTime {
  days: number;
  hours: number;
  minutes: number;
  seconds: number;
} 