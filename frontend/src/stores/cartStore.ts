import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';

interface CartItem {
  id: number;
  productName: string;
  price: number;
  quantity: number;
  imageUrl?: string;
}

interface CartState {
  items: CartItem[];
  totalCount: number;
  totalPrice: number;
  addItem: (item: Omit<CartItem, 'quantity'>) => void;
  removeItem: (id: number) => void;
  updateQuantity: (id: number, quantity: number) => void;
  clearCart: () => void;
  calculateTotals: () => void;
}

export const useCartStore = create<CartState>()(
  persist(
    (set, get) => ({
      items: [],
      totalCount: 0,
      totalPrice: 0,
      
      addItem: (newItem) => {
        const state = get();
        const existingItemIndex = state.items.findIndex(item => item.id === newItem.id);
        
        if (existingItemIndex >= 0) {
          // 如果商品已存在，增加数量
          const updatedItems = [...state.items];
          updatedItems[existingItemIndex].quantity += 1;
          set({ items: updatedItems });
        } else {
          // 如果是新商品，添加到购物车
          set({ 
            items: [...state.items, { ...newItem, quantity: 1 }]
          });
        }
        
        // 重新计算总计
        get().calculateTotals();
      },
      
      removeItem: (id) => {
        const state = get();
        const updatedItems = state.items.filter(item => item.id !== id);
        set({ items: updatedItems });
        get().calculateTotals();
      },
      
      updateQuantity: (id, quantity) => {
        if (quantity <= 0) {
          get().removeItem(id);
          return;
        }
        
        const state = get();
        const updatedItems = state.items.map(item =>
          item.id === id ? { ...item, quantity } : item
        );
        set({ items: updatedItems });
        get().calculateTotals();
      },
      
      clearCart: () => {
        set({ 
          items: [], 
          totalCount: 0, 
          totalPrice: 0 
        });
      },
      
      calculateTotals: () => {
        const state = get();
        const totalCount = state.items.reduce((sum, item) => sum + item.quantity, 0);
        const totalPrice = state.items.reduce((sum, item) => sum + (item.price * item.quantity), 0);
        set({ totalCount, totalPrice });
      },
    }),
    {
      name: 'cart-storage',
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        items: state.items,
      }),
      onRehydrateStorage: () => (state) => {
        // 恢复数据后重新计算总计
        if (state) {
          state.calculateTotals();
        }
      },
    }
  )
);

// 选择器函数
export const useCartItems = () => useCartStore((state) => state.items);
export const useCartTotalCount = () => useCartStore((state) => state.totalCount);
export const useCartTotalPrice = () => useCartStore((state) => state.totalPrice); 