import axios from 'axios';
import { Todo, ApiResponse, CreateTodoRequest, TodoListResponse } from '../types';

// 创建axios实例
const api = axios.create({
  baseURL: '/api',
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