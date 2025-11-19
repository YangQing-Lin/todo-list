export interface Todo {
  id: number;
  version: number;
  title: string;
  description: string;
  status: 'pending' | 'completed';
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

export interface TodoStats {
  total: number;
  pending: number;
  completed: number;
  overdue: number;
  today: number;
  this_week: number;
}
