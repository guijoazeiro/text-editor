import axiosInstance from "./axios";

export { axiosInstance as api };

export const authAPI = {
  signup: (data: { name: string; email: string; password: string }) =>
    axiosInstance.post("/api/auth/signup", data),
  login: (data: { email: string; password: string }) =>
    axiosInstance.post("/api/auth/login", data),
  me: () => axiosInstance.get("/api/auth/me"),
  updateMe: (data: { name: string }) =>
    axiosInstance.patch("/api/auth/me", data),
  refresh: () =>
    axiosInstance.post("/api/auth/refresh", {}, { withCredentials: true }),
  logout: () => axiosInstance.post("/api/auth/logout"),
};


export const documentsAPI = {
  list: (params?: { page?: number; limit?: number; q?: string }) =>
    axiosInstance.get("/api/documents", { params }),
  trash: () => axiosInstance.get("/api/documents/trash"),
  restore: (id: string) => axiosInstance.post(`/api/documents/${id}/restore`),
  get: (id: string) => axiosInstance.get(`/api/documents/${id}`),
  create: (data: { title: string; content: string; content_format?: string }) =>
    axiosInstance.post("/api/documents", data),
  update: (
    id: string,
    data: { title?: string; content?: string; content_format?: string },
  ) => axiosInstance.put(`/api/documents/${id}`, data),
  delete: (id: string) => axiosInstance.delete(`/api/documents/${id}`),
  getActiveUsers: (id: string) =>
    axiosInstance.get(`/api/documents/${id}/active-users`),
};

export const collaboratorsAPI = {
  list: (documentId: string) =>
    axiosInstance.get(`/api/documents/${documentId}/collaborators`),
  add: (documentId: string, data: { email: string; permission: string }) =>
    axiosInstance.post(`/api/documents/${documentId}/collaborators`, data),
  update: (documentId: string, userId: string, data: { permission: string }) =>
    axiosInstance.put(
      `/api/documents/${documentId}/collaborators/${userId}`,
      data,
    ),
  remove: (documentId: string, userId: string) =>
    axiosInstance.delete(
      `/api/documents/${documentId}/collaborators/${userId}`,
    ),
};

export const shareLinksAPI = {
  create: (
    documentId: string,
    data: { permission: "viewer" | "editor"; expires_at?: string },
  ) => axiosInstance.post(`/api/documents/${documentId}/share-link`, data),
  remove: (documentId: string) =>
    axiosInstance.delete(`/api/documents/${documentId}/share-link`),
};

export const versionsAPI = {
  list: (documentId: string, params?: { page?: number; limit?: number }) =>
    axiosInstance.get(`/api/documents/${documentId}/versions`, { params }),
  get: (documentId: string, versionNumber: number) =>
    axiosInstance.get(`/api/documents/${documentId}/versions/${versionNumber}`),
  restore: (documentId: string, versionNumber: number) =>
    axiosInstance.post(
      `/api/documents/${documentId}/versions/${versionNumber}/restore`,
    ),
  compare: (documentId: string, v1: number, v2: number) =>
    axiosInstance.get(
      `/api/documents/${documentId}/versions/compare?v1=${v1}&v2=${v2}`,
    ),
};

export const notificationsAPI = {
  list: () => axiosInstance.get("/api/notifications"),
  markAsRead: (id: string) =>
    axiosInstance.put(`/api/notifications/${id}/read`),
  markAllAsRead: () => axiosInstance.put("/api/notifications/read-all"),
  delete: (id: string) => axiosInstance.delete(`/api/notifications/${id}`),
};
