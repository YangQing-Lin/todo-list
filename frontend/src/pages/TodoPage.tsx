import React, { useState, useEffect } from 'react';
import { Todo } from '../types';
import { todoApi, extractErrorMessage } from '../services/api';
import TodoItem from '../components/TodoItem';
import TodoForm from '../components/TodoForm';
import '../styles/TodoPage.css';

const TodoPage: React.FC = () => {
  const [todos, setTodos] = useState<Todo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [filter, setFilter] = useState<'all' | 'pending' | 'completed'>('all');

  // 获取Todos
  const fetchTodos = async () => {
    setLoading(true);
    try {
      const response = await todoApi.getTodos();
      if (response.success) {
        setTodos(response.data.todos || []);
      } else {
        setError(response.error?.message || '获取数据失败');
      }
    } catch (err: any) {
      setError(extractErrorMessage(err, '获取数据失败'));
    } finally {
      setLoading(false);
    }
  };

  // 初始化加载
  useEffect(() => {
    fetchTodos();
  }, []);

  // 处理删除
  const handleDelete = async (id: number) => {
    if (!confirm('确定要删除这个待办事项吗？')) {
      return;
    }

    try {
      const response = await todoApi.deleteTodo(id);
      if (response.success) {
        setTodos(todos.filter(todo => todo.id !== id));
      } else {
        setError(response.error?.message || '删除失败');
      }
    } catch (err: any) {
      setError(extractErrorMessage(err, '删除失败'));
    }
  };

  // 处理完成状态切换
  const handleToggle = async (id: number) => {
    const todo = todos.find(t => t.id === id);
    if (!todo) return;

    const newStatus = todo.status === 'pending' ? 'completed' : 'pending';

    try {
      // 包含version字段以支持并发控制
      const response = await todoApi.updateTodo(id, {
        status: newStatus,
        version: todo.version
      });
      if (response.success) {
        const updatedTodos = todos.map(t => {
          if (t.id === id) {
            return response.data;
          }
          return t;
        });
        setTodos(updatedTodos);
      } else {
        setError(response.error?.message || '更新失败');
      }
    } catch (err: any) {
      setError(extractErrorMessage(err, '更新失败'));
      // 如果是版本冲突，刷新数据
      if (err.response?.status === 409) {
        fetchTodos();
      }
    }
  };

  // 过滤Todos
  const filteredTodos = todos.filter(todo => {
    if (filter === 'pending') return todo.status === 'pending';
    if (filter === 'completed') return todo.status === 'completed';
    return true;
  });

  // 统计数据
  const stats = {
    total: todos.length,
    pending: todos.filter(t => t.status === 'pending').length,
    completed: todos.filter(t => t.status === 'completed').length,
  };

  if (loading) {
    return <div className="loading">加载中...</div>;
  }

  return (
    <div className="todo-page">
      <div className="container">
        <header className="page-header">
          <h1>我的待办事项</h1>
          <div className="stats">
            <span>总计: {stats.total}</span>
            <span>待办: {stats.pending}</span>
            <span>已完成: {stats.completed}</span>
          </div>
        </header>

        {error && <div className="error">{error}</div>}

        <TodoForm onTodoCreated={fetchTodos} />

        <div className="todo-filters">
          <button
            className={`filter-btn ${filter === 'all' ? 'active' : ''}`}
            onClick={() => setFilter('all')}
          >
            全部 ({stats.total})
          </button>
          <button
            className={`filter-btn ${filter === 'pending' ? 'active' : ''}`}
            onClick={() => setFilter('pending')}
          >
            待办 ({stats.pending})
          </button>
          <button
            className={`filter-btn ${filter === 'completed' ? 'active' : ''}`}
            onClick={() => setFilter('completed')}
          >
            已完成 ({stats.completed})
          </button>
        </div>

        <div className="todo-list">
          {filteredTodos.length === 0 ? (
            <div className="empty-state">
              <h3>
                {filter === 'completed' ? '还没有完成的任务' :
                 filter === 'pending' ? '没有待办任务了！' :
                 '还没有待办事项'}
              </h3>
              <p>
                {filter === 'all' && '添加你的第一个待办事项吧！'}
              </p>
            </div>
          ) : (
            filteredTodos.map(todo => (
              <TodoItem
                key={todo.id}
                todo={todo}
                onToggle={handleToggle}
                onDelete={handleDelete}
              />
            ))
          )}
        </div>
      </div>
    </div>
  );
};

export default TodoPage;