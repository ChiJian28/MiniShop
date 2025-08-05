import { useMutation, useQueryClient } from '@tanstack/react-query';
import { seckillApi } from '../services/api';
import { useAuthStore } from '../stores/authStore';

interface LoginCredentials {
  username: string;
  password: string;
}

// 登录 mutation
export const useLogin = () => {
  const login = useAuthStore((state) => state.login);
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ username, password }: LoginCredentials) => 
      seckillApi.login(username, password),
    
    onSuccess: (data) => {
      if (data.code === 0 && data.data?.token) {
        // 更新认证状态
        login(data.data.token, data.data.user);
        
        // 清除所有查询缓存，因为用户已经登录
        queryClient.clear();
        
        console.log('Login successful via React Query:', data);
      }
    },
    
    onError: (error) => {
      console.error('Login failed via React Query:', error);
    },
  });
};

// 登出 mutation
export const useLogout = () => {
  const logout = useAuthStore((state) => state.logout);
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async () => {
      // 这里可以调用后端登出接口，目前只是本地清理
      return Promise.resolve();
    },
    
    onSuccess: () => {
      // 清除认证状态
      logout();
      
      // 清除所有查询缓存
      queryClient.clear();
      
      console.log('Logout successful via React Query');
    },
  });
}; 