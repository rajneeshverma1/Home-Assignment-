'use client';

import React, { createContext, useContext, useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';

export interface User {
  id: number;
  email: string;
  role: string;
  created_at: string;
}

interface AuthContextType {
  user: User | null;
  token: string | null;
  isLoading: boolean;
  login: (token: string, user: User) => void;
  logout: () => void;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  apiFetch: (path: string, options?: RequestInit) => Promise<any>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const router = useRouter();

  // Load auth state from localStorage on startup
  useEffect(() => {
    try {
      const storedToken = localStorage.getItem('tm_token');
      const storedUser = localStorage.getItem('tm_user');
      
      if (storedToken && storedUser) {
        setTimeout(() => {
          setToken(storedToken);
          setUser(JSON.parse(storedUser));
        }, 0);
      }
    } catch (e) {
      console.error('Error loading auth from localStorage:', e);
    } finally {
      setTimeout(() => {
        setIsLoading(false);
      }, 0);
    }
  }, []);

  const login = (newToken: string, newUser: User) => {
    setToken(newToken);
    setUser(newUser);
    localStorage.setItem('tm_token', newToken);
    localStorage.setItem('tm_user', JSON.stringify(newUser));
    router.push('/');
  };

  const logout = () => {
    setToken(null);
    setUser(null);
    localStorage.removeItem('tm_token');
    localStorage.removeItem('tm_user');
    router.push('/login');
  };

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const apiFetch = async (path: string, options: RequestInit = {}): Promise<any> => {
    const headers = new Headers(options.headers || {});
    
    // Auto-inject JWT token if available
    const activeToken = token || localStorage.getItem('tm_token');
    if (activeToken) {
      headers.set('Authorization', `Bearer ${activeToken}`);
    }
    
    if (!(options.body instanceof FormData)) {
      headers.set('Content-Type', 'application/json');
    }
    headers.set('Accept', 'application/json');

    const response = await fetch(`${API_URL}${path}`, {
      ...options,
      headers,
    });

    if (response.status === 204) {
      return null;
    }

    const data = await response.json();

    if (!response.ok) {
      // Auto-logout if unauthorized (expired/invalid token)
      if (response.status === 401 && path !== '/login' && path !== '/signup') {
        logout();
      }
      throw new Error(data.error || 'Something went wrong');
    }

    return data;
  };

  return (
    <AuthContext.Provider value={{ user, token, isLoading, login, logout, apiFetch }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
