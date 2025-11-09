import axios from 'axios';
import { Todo, ApiResponse, CreateTodoRequest, TodoListResponse } from '../types';

// 创建axios实例
const api = axios.create({
  baseURL: '/api/v1',
  headers: {
    'Content-Type': 'application/json',
  },
});

// 响应拦截器
api.interceptors.response.use(
  (response) => response.data,
  (error) => {
    console.error('API Error:', error);
    return Promise.reject(error);
  }
);

// 提取错误消息的工具函数
export const extractErrorMessage = (error: any, defaultMessage: string = '操作失败，请重试'): string => {
  // 处理网络错误（没有响应）
  if (!error.response) {
    return '网络错误，请检查网络连接';
  }

  // 提取后端返回的错误消息
  const backendMessage = error.response?.data?.error?.message;
  if (backendMessage) {
    return backendMessage;
  }

  // 根据HTTP状态码返回友好提示
  const status = error.response?.status;
  if (status === 404) {
    return '请求的资源不存在';
  } else if (status === 409) {
    return '数据已被其他人修改，请刷新后重试';
  } else if (status === 500) {
    return '服务器错误，请稍后重试';
  }

  return defaultMessage;
};

// API方法
export const todoApi = {
  // 获取所有Todos
  getTodos: (): Promise<ApiResponse<TodoListResponse>> => {
    return api.get('/todos');
  },

  // 创建新Todo
  createTodo: (data: CreateTodoRequest): Promise<ApiResponse<Todo>> => {
    return api.post('/todos', data);
  },

  // 更新Todo（预留）
  updateTodo: (id: number, data: Partial<Todo>): Promise<ApiResponse<Todo>> => {
    return api.put(`/todos/${id}`, data);
  },

  // 删除Todo（预留）
  deleteTodo: (id: number): Promise<ApiResponse<null>> => {
    return api.delete(`/todos/${id}`);
  },
};

export default api;
