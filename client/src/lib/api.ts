import axios from 'axios';

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export const api = axios.create({
  baseURL: API_URL,
  headers: { 'Content-Type': 'application/json' },
});

api.interceptors.request.use((config) => {
  const authStorage = localStorage.getItem('auth-storage');
  if (authStorage) {
    try {
      const token = JSON.parse(authStorage)?.state?.token;
      if (token) config.headers.Authorization = `Bearer ${token}`;
    } catch { /* ignore */ }
  }
  return config;
});

export const authAPI = {
  signup: (data: { name: string; email: string; password: string }) =>
    api.post('/api/auth/signup', data),
  login: (data: { email: string; password: string }) =>
    api.post('/api/auth/login', data),
  me: () => api.get('/api/auth/me'),
};

export const documentsAPI = {
  list: () => api.get('/api/documents'),
  get: (id: string) => api.get(`/api/documents/${id}`),
  create: (data: { title: string; content: string; content_format?: string }) =>
    api.post('/api/documents', data),
  update: (id: string, data: { title?: string; content?: string; content_format?: string }) =>
    api.put(`/api/documents/${id}`, data),
  delete: (id: string) => api.delete(`/api/documents/${id}`),
  getActiveUsers: (id: string) => api.get(`/api/documents/${id}/active-users`),
};

export const collaboratorsAPI = {
  list: (documentId: string) =>
    api.get(`/api/documents/${documentId}/collaborators`),
  add: (documentId: string, data: { email: string; permission: string }) =>
    api.post(`/api/documents/${documentId}/collaborators`, data),
  update: (documentId: string, userId: string, data: { permission: string }) =>
    api.put(`/api/documents/${documentId}/collaborators/${userId}`, data),
  remove: (documentId: string, userId: string) =>
    api.delete(`/api/documents/${documentId}/collaborators/${userId}`),
};

export const notificationsAPI = {
  list: () => api.get('/api/notifications'),
  markAsRead: (id: string) => api.put(`/api/notifications/${id}/read`),
  markAllAsRead: () => api.put('/api/notifications/read-all'),
  delete: (id: string) => api.delete(`/api/notifications/${id}`),
};
