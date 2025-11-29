// Neo-Brutalism 调色板
export const TODO_COLORS = [
  '#FFE066', // Yellow
  '#7DFFAF', // Mint
  '#FF9ECD', // Pink
  '#9EE5FF', // Cyan
  '#D4ADFF', // Purple
  '#FFB399', // Peach
] as const;

export type TodoColor = typeof TODO_COLORS[number];

// 根据 ID 获取稳定的默认颜色（用于旧数据兼容）
export const getDefaultColorById = (id: number): TodoColor => {
  return TODO_COLORS[id % TODO_COLORS.length];
};

// 随机选择一个颜色
export const getRandomColor = (): TodoColor => {
  return TODO_COLORS[Math.floor(Math.random() * TODO_COLORS.length)];
};

export interface Todo {
  id: number;
  version: number;
  title: string;
  description: string;
  status: 'pending' | 'completed';
  color?: TodoColor;  // 待办事项的固定颜色
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
  color?: TodoColor;
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
