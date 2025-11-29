import React, { useState, useEffect, useRef } from 'react';
import { Todo, TodoStats } from '../types';
import { todoApi, extractErrorMessage } from '../services/api';
import TodoItem from '../components/TodoItem';
import TodoForm from '../components/TodoForm';
import StatsCard from '../components/StatsCard';
import ConfirmDialog from '../components/ConfirmDialog';
import '../styles/TodoPage.css';

const LEAVE_ANIMATION_MS = 260;

const TodoPage: React.FC = () => {
  const [todos, setTodos] = useState<Todo[]>([]);
  const [isInitialLoad, setIsInitialLoad] = useState(true);
  const [error, setError] = useState('');
  const [filter, setFilter] = useState<'all' | 'pending' | 'completed'>('all');
  const [stats, setStats] = useState<TodoStats | null>(null);
  const [statsLoading, setStatsLoading] = useState(true);
  const [leavingIds, setLeavingIds] = useState<Set<number>>(new Set());
  const leaveTimersRef = useRef<Map<number, number>>(new Map());

  // 删除确认弹窗状态
  const [deleteConfirm, setDeleteConfirm] = useState<{
    isOpen: boolean;
    todoId: number | null;
    todoTitle: string;
  }>({ isOpen: false, todoId: null, todoTitle: '' });

  // 清理定时器
  useEffect(() => {
    const timers = leaveTimersRef.current;
    return () => {
      timers.forEach(timerId => clearTimeout(timerId));
      timers.clear();
    };
  }, []);

  // 获取Todos（静默刷新，不触发全屏loading）
  const fetchTodos = async (silent = false) => {
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
      if (!silent) {
        setIsInitialLoad(false);
      }
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

  // 请求删除（打开确认弹窗）
  const requestDelete = (id: number) => {
    const todo = todos.find(t => t.id === id);
    setDeleteConfirm({
      isOpen: true,
      todoId: id,
      todoTitle: todo?.title || '',
    });
  };

  // 取消删除
  const cancelDelete = () => {
    setDeleteConfirm({ isOpen: false, todoId: null, todoTitle: '' });
  };

  // 确认删除
  const confirmDelete = async () => {
    const id = deleteConfirm.todoId;
    if (!id) return;

    // 先关闭弹窗
    setDeleteConfirm({ isOpen: false, todoId: null, todoTitle: '' });

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

    // 乐观更新：立即更新UI
    setTodos(prev => prev.map(t => {
      if (t.id === id) {
        return { ...t, status: newStatus };
      }
      return t;
    }));

    try {
      // 包含version字段以支持并发控制
      const response = await todoApi.updateTodo(id, {
        status: newStatus,
        version: todo.version
      });
      if (response.success) {
        // 用服务器返回的数据更新（包含新的 version）
        setTodos(prev => prev.map(t => {
          if (t.id === id) {
            return response.data;
          }
          return t;
        }));
        fetchStats(); // 刷新统计信息
      } else {
        // 回滚
        setTodos(prev => prev.map(t => {
          if (t.id === id) {
            return todo;
          }
          return t;
        }));
        setError(response.error?.message || '更新失败');
      }
    } catch (err: any) {
      // 回滚
      setTodos(prev => prev.map(t => {
        if (t.id === id) {
          return todo;
        }
        return t;
      }));
      const message = extractErrorMessage(err, '更新失败');
      if (err.response?.status === 409) {
        setError(`${message}（请刷新或重试以获取最新数据）`);
      } else {
        setError(message);
      }
    }
  };

  // 创建后刷新（静默）
  const handleTodoCreated = () => {
    fetchTodos(true);
    fetchStats();
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

  // 只在首次加载时显示全屏loading
  if (isInitialLoad) {
    return <div className="loading">加载中...</div>;
  }

  return (
    <div className="todo-page">
      <div className="container">
        <header className="page-header">
          <h1>我的待办事项</h1>
        </header>

        {error && (
          <div className="error" onClick={() => setError('')}>
            {error}
            <span className="error-close">×</span>
          </div>
        )}

        <div className="page-layout">
          <aside className="sidebar">
            <StatsCard stats={stats} loading={statsLoading} />
          </aside>

          <main className="main-content">
            <TodoForm onTodoCreated={handleTodoCreated} />

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
                    onDelete={requestDelete}
                    isLeaving={leavingIds.has(todo.id)}
                  />
                ))
              )}
            </div>
          </main>
        </div>
      </div>

      <ConfirmDialog
        isOpen={deleteConfirm.isOpen}
        title="删除确认"
        message={`确定要删除「${deleteConfirm.todoTitle}」吗？此操作无法撤销。`}
        confirmText="删除"
        cancelText="取消"
        variant="danger"
        onConfirm={confirmDelete}
        onCancel={cancelDelete}
      />
    </div>
  );
};

export default TodoPage;
