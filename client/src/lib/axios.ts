import axios from "axios";
import { useAuthStore } from "@/store/authStore";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export const axiosInstance = axios.create({
  baseURL: API_URL,
  headers: { "Content-Type": "application/json" },
});

axiosInstance.interceptors.request.use((config) => {
  const token = useAuthStore.getState().token;
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

axiosInstance.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      useAuthStore.getState().logout();

      if (typeof window !== "undefined") {
        window.location.href = "/login";
      }
    }
    return Promise.reject(error);
  },
);

export default axiosInstance;
