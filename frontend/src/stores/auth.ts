import { create } from 'zustand';
import { persist } from 'zustand/middleware';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface UserInfo {
  id: string;
  username: string;
  role: string;
  tenantId: string;
}

interface AuthState {
  token: string | null;
  user: UserInfo | null;
  isAuthenticated: boolean;
  login: (token: string, user: UserInfo) => void;
  logout: () => void;
  setUser: (user: UserInfo) => void;
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      token: null,
      user: null,
      isAuthenticated: false,

      login: (token: string, user: UserInfo) => {
        set({ token, user, isAuthenticated: true });
      },

      logout: () => {
        localStorage.removeItem('auth-storage');
        set({ token: null, user: null, isAuthenticated: false });
      },

      setUser: (user: UserInfo) => set({ user }),
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({
        token: state.token,
        user: state.user,
      }),
    }
  )
);

export default useAuthStore;
