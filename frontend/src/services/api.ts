/// <reference types="vite/client" />

import axios, { type AxiosRequestConfig } from 'axios';
import useAuthStore from '@/stores/auth';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface ApiResponse<T = unknown> {
  code: number;
  message: string;
  data: T;
}

// ---------------------------------------------------------------------------
// Axios instance
// ---------------------------------------------------------------------------

const axiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

axiosInstance.interceptors.request.use(
  (config) => {
    const token = useAuthStore.getState().token;
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

axiosInstance.interceptors.response.use(
  ((response) => {
    return response.data;
  }),
  (error) => {
    if (error.response?.status === 401) {
      useAuthStore.getState().logout();
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

async function request<T>(config: AxiosRequestConfig): Promise<T> {
  const wrapper = await axiosInstance.request<ApiResponse<T>>(config);
  return wrapper.data as T;
}

// ---------------------------------------------------------------------------
// API methods
// ---------------------------------------------------------------------------

export async function postLogin(
  username: string,
  password: string
): Promise<{ token: string; user: { id: string; username: string; role: string; tenant_id?: string } }> {
  return request({
    method: 'POST',
    url: '/auth/login',
    data: { username, password },
  });
}

export async function postRegister(
  username: string,
  password: string,
  displayName?: string,
  email?: string,
  tenantId?: string
): Promise<{ token: string; user: { id: string; username: string; role: string; tenant_id?: string } }> {
  return request({
    method: 'POST',
    url: '/auth/register',
    data: {
      username,
      password,
      display_name: displayName,
      email,
      role: 'reviewer',
      tenant_id: tenantId || undefined,
    },
  });
}

export async function getTenants(
  page = 1,
  pageSize = 20
): Promise<{ items: Array<{ id: string; name: string; country_code: string; status: string; created_at: string }>; total: number }> {
  const res = await request<{ tenants: Array<{ id: string; name: string; country_code: string; status: string; created_at: string }>; total: number; page: number; page_size: number }>({
    method: 'GET',
    url: '/tenants',
    params: { page, page_size: pageSize },
  });
  return { items: res.tenants ?? [], total: res.total ?? 0 };
}

export async function createTenant(name: string, countryCode: string): Promise<{ id: string }> {
  return request({
    method: 'POST',
    url: '/tenants',
    data: { name, country_code: countryCode },
  });
}

export async function updateTenant(
  id: string,
  payload: { name?: string; country_code?: string }
): Promise<void> {
  return request({
    method: 'PUT',
    url: `/tenants/${id}`,
    data: payload,
  });
}

export async function deleteTenant(id: string): Promise<void> {
  return request({
    method: 'DELETE',
    url: `/tenants/${id}`,
  });
}

export async function getTeams(tenantId: string): Promise<
  Array<{ id: string; name: string; leader_id: string; member_count: number }>
> {
  return request({
    method: 'GET',
    url: '/teams',
    params: { tenant_id: tenantId },
  });
}

export async function createTeam(
  name: string,
  leaderId: string
): Promise<{ id: string }> {
  return request({
    method: 'POST',
    url: '/teams',
    data: { name, leader_id: leaderId },
  });
}

export async function addTeamMember(
  teamId: string,
  userId: string,
  role: string
): Promise<void> {
  return request({
    method: 'POST',
    url: `/teams/${teamId}/members`,
    data: { user_id: userId, member_role: role },
  });
}

export async function removeTeamMember(teamId: string, userId: string): Promise<void> {
  return request({
    method: 'DELETE',
    url: `/teams/${teamId}/members/${userId}`,
  });
}

export default axiosInstance;
