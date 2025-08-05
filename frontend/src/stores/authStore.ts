import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';

interface User {
  id: number;
  username: string;
}

interface AuthState {
  isAuthenticated: boolean;
  user: User | null;
  token: string | null;
  loading: boolean;
  login: (token: string, user?: User) => void;
  logout: () => void;
  setLoading: (loading: boolean) => void;
  initializeAuth: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      isAuthenticated: false,
      user: null,
      token: null,
      loading: true,
      
      login: (newToken: string, newUser?: User) => {
        console.log('AuthStore login called:', { newToken, newUser });
        set({
          token: newToken,
          isAuthenticated: true,
          user: newUser || null,
          loading: false,
        });
        console.log('AuthStore: state updated', { 
          isAuthenticated: true, 
          token: newToken, 
          user: newUser 
        });
      },
      
      logout: () => {
        console.log('AuthStore logout called');
        set({
          token: null,
          user: null,
          isAuthenticated: false,
          loading: false,
        });
      },
      
      setLoading: (loading: boolean) => {
        set({ loading });
      },
      
      initializeAuth: () => {
        const state = get();
        if (state.token && state.isAuthenticated) {
          // 如果已经从持久化存储中恢复了状态，直接设置加载完成
          set({ loading: false });
          return;
        }
        
        // 否则设置为未认证状态
        set({ 
          loading: false,
          isAuthenticated: false,
          token: null,
          user: null,
        });
      },
    }),
    {
      name: 'auth-storage',
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        token: state.token,
        user: state.user,
        isAuthenticated: state.isAuthenticated,
      }),
    }
  )
);

// 选择器函数，用于优化性能
export const useAuthToken = () => useAuthStore((state) => state.token);
export const useIsAuthenticated = () => useAuthStore((state) => state.isAuthenticated);
export const useAuthUser = () => useAuthStore((state) => state.user);
export const useAuthLoading = () => useAuthStore((state) => state.loading); 