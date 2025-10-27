export interface Todo {
  id: number;
  title: string;
  description: string;
  status: 'pending' | 'completed';
  priority: number; // 1=低, 2=中, 3=高
  due_date?: string;
  created_at: string;
  updated_at: string;
  completed_at?: string;
}

export interface ApiResponse<T> {
  success: boolean;
  data: T;
  error?: {
    code: string;
    message: string;
  };
  message?: string;
}

export interface CreateTodoRequest {
  title: string;
  description?: string;
}

export interface TodoListResponse {
  todos: Todo[];
  total: number;
}