import React, { useState, useEffect, useRef } from 'react';
import { Todo, TodoStats } from '../types';
import { todoApi, extractErrorMessage } from '../services/api';
import TodoItem from '../components/TodoItem';
import TodoForm from '../components/TodoForm';
import StatsCard from '../components/StatsCard';
import '../styles/TodoPage.css';

const LEAVE_ANIMATION_MS = 260;

const TodoPage: React.FC = () => {
  const [todos, setTodos] = useState<Todo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [filter, setFilter] = useState<'all' | 'pending' | 'completed'>('all');
  const [stats, setStats] = useState<TodoStats | null>(null);
  const [statsLoading, setStatsLoading] = useState(true);
  const [leavingIds, setLeavingIds] = useState<Set<number>>(new Set());
  const leaveTimersRef = useRef<Map<number, number>>(new Map());

  // 清理定时器
  useEffect(() => {
    const timers = leaveTimersRef.current;
    return () => {
      timers.forEach(timerId => clearTimeout(timerId));
      timers.clear();
    };
  }, []);

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

  // 获取统计信息
  const fetchStats = async () => {
    setStatsLoading(true);
    try {
      const response = await todoApi.getStats();
      if (response.success) {
        setStats(response.data);
      } else {
        console.error('获取统计信息失败:', response.error?.message);
      }
    } catch (err: any) {
      console.error('获取统计信息失败:', extractErrorMessage(err, '获取统计信息失败'));
    } finally {
      setStatsLoading(false);
    }
  };

  // 初始化加载
  useEffect(() => {
    fetchTodos();
    fetchStats();
  }, []);

  // 处理删除
  const handleDelete = async (id: number) => {
    if (!confirm('确定要删除这个待办事项吗？')) {
      return;
    }

    try {
      const response = await todoApi.deleteTodo(id);
      if (response.success) {
        setLeavingIds(prev => {
          const updated = new Set(prev);
          updated.add(id);
          return updated;
        });

        const timerId = window.setTimeout(() => {
          setTodos(prev => prev.filter(todo => todo.id !== id));
          setLeavingIds(prev => {
            const updated = new Set(prev);
            updated.delete(id);
            return updated;
          });
          leaveTimersRef.current.delete(id);
          fetchStats(); // 动画结束后刷新统计信息
        }, LEAVE_ANIMATION_MS);

        leaveTimersRef.current.set(id, timerId);
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
        fetchStats(); // 刷新统计信息
      } else {
        setError(response.error?.message || '更新失败');
      }
    } catch (err: any) {
      const message = extractErrorMessage(err, '更新失败');
      if (err.response?.status === 409) {
        setError(`${message}（请刷新或重试以获取最新数据）`);
      } else {
        setError(message);
      }
    }
  };

  // 过滤Todos
  const filteredTodos = todos.filter(todo => {
    if (filter === 'pending') return todo.status === 'pending';
    if (filter === 'completed') return todo.status === 'completed';
    return true;
  });

  // 本地统计数据（用于过滤按钮）
  const localStats = {
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
        </header>

        {error && <div className="error">{error}</div>}

        <div className="page-layout">
          <aside className="sidebar">
            <StatsCard stats={stats} loading={statsLoading} />
          </aside>

          <main className="main-content">
            <TodoForm onTodoCreated={() => { fetchTodos(); fetchStats(); }} />

            <div className="todo-filters">
              <button
                className={`filter-btn ${filter === 'all' ? 'active' : ''}`}
                onClick={() => setFilter('all')}
              >
                全部 ({localStats.total})
              </button>
              <button
                className={`filter-btn ${filter === 'pending' ? 'active' : ''}`}
                onClick={() => setFilter('pending')}
              >
                待办 ({localStats.pending})
              </button>
              <button
                className={`filter-btn ${filter === 'completed' ? 'active' : ''}`}
                onClick={() => setFilter('completed')}
              >
                已完成 ({localStats.completed})
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
                    isLeaving={leavingIds.has(todo.id)}
                  />
                ))
              )}
            </div>
          </main>
        </div>
      </div>
    </div>
  );
};

export default TodoPage;
