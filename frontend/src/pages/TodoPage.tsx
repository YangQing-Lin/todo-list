import React, { useState, useEffect, useRef } from 'react';
import { Todo, TodoStats, BatchResult } from '../types';
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

  // 多选模式状态
  const [selectionMode, setSelectionMode] = useState(false);
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set());

  // 批量操作确认弹窗状态
  const [batchConfirm, setBatchConfirm] = useState<{
    isOpen: boolean;
    action: 'complete' | 'delete' | null;
  }>({ isOpen: false, action: null });

  // 批量操作进行中状态
  const [batchLoading, setBatchLoading] = useState(false);

  // 导入相关状态
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [importLoading, setImportLoading] = useState(false);

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

  // 获取统计信息（silent=true 时不触发 loading 状态，避免闪烁）
  const fetchStats = async (silent = false) => {
    if (!silent) {
      setStatsLoading(true);
    }
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
      if (!silent) {
        setStatsLoading(false);
      }
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
          fetchStats(true); // 静默刷新统计信息，避免闪烁
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
        fetchStats(true); // 静默刷新统计信息
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
    fetchStats(true);
  };

  // 更新后刷新
  const handleTodoUpdated = (updatedTodo: Todo) => {
    setTodos(prev => prev.map(t => t.id === updatedTodo.id ? updatedTodo : t));
    fetchStats(true);
  };

  // ========================================
  // 多选模式相关函数
  // ========================================

  // 切换多选模式
  const toggleSelectionMode = () => {
    setSelectionMode(prev => !prev);
    setSelectedIds(new Set());
  };

  // 选择/取消选择单个项目
  const handleSelect = (id: number) => {
    setSelectedIds(prev => {
      const updated = new Set(prev);
      if (updated.has(id)) {
        updated.delete(id);
      } else {
        updated.add(id);
      }
      return updated;
    });
  };

  // 全选/取消全选（当前过滤后的列表）
  const handleSelectAll = () => {
    const filteredIds = filteredTodos.map(t => t.id);
    const allSelected = filteredIds.every(id => selectedIds.has(id));

    if (allSelected) {
      // 取消全选
      setSelectedIds(prev => {
        const updated = new Set(prev);
        filteredIds.forEach(id => updated.delete(id));
        return updated;
      });
    } else {
      // 全选
      setSelectedIds(prev => {
        const updated = new Set(prev);
        filteredIds.forEach(id => updated.add(id));
        return updated;
      });
    }
  };

  // 请求批量操作（打开确认弹窗）
  const requestBatchAction = (action: 'complete' | 'delete') => {
    if (selectedIds.size === 0) return;
    setBatchConfirm({ isOpen: true, action });
  };

  // 取消批量操作
  const cancelBatchAction = () => {
    setBatchConfirm({ isOpen: false, action: null });
  };

  // 确认批量完成
  const confirmBatchComplete = async () => {
    setBatchConfirm({ isOpen: false, action: null });
    setBatchLoading(true);

    try {
      const ids = Array.from(selectedIds);
      const response = await todoApi.batchComplete(ids);

      if (response.success) {
        const result = response.data;
        // 显示结果消息
        if (result.failed_count > 0) {
          setError(`批量完成: ${result.success_count} 个成功，${result.failed_count} 个失败`);
        }
        // 刷新数据
        await fetchTodos(true);
        await fetchStats(true);
        // 清空选择
        setSelectedIds(new Set());
        setSelectionMode(false);
      } else {
        setError(response.error?.message || '批量完成失败');
      }
    } catch (err: any) {
      setError(extractErrorMessage(err, '批量完成失败'));
    } finally {
      setBatchLoading(false);
    }
  };

  // 确认批量删除
  const confirmBatchDelete = async () => {
    setBatchConfirm({ isOpen: false, action: null });
    setBatchLoading(true);

    try {
      const ids = Array.from(selectedIds);
      const response = await todoApi.batchDelete(ids);

      if (response.success) {
        const result = response.data;
        // 显示结果消息
        if (result.failed_count > 0) {
          setError(`批量删除: ${result.success_count} 个成功，${result.failed_count} 个失败`);
        }
        // 刷新数据
        await fetchTodos(true);
        await fetchStats(true);
        // 清空选择
        setSelectedIds(new Set());
        setSelectionMode(false);
      } else {
        setError(response.error?.message || '批量删除失败');
      }
    } catch (err: any) {
      setError(extractErrorMessage(err, '批量删除失败'));
    } finally {
      setBatchLoading(false);
    }
  };

  // 处理批量操作确认
  const handleBatchConfirm = () => {
    if (batchConfirm.action === 'complete') {
      confirmBatchComplete();
    } else if (batchConfirm.action === 'delete') {
      confirmBatchDelete();
    }
  };

  // ========================================
  // 导入导出相关函数
  // ========================================

  // 导出 JSON
  const handleExportJSON = () => {
    todoApi.exportTodos('json');
  };

  // 导出 CSV
  const handleExportCSV = () => {
    todoApi.exportTodos('csv');
  };

  // 触发文件选择
  const handleImportClick = () => {
    fileInputRef.current?.click();
  };

  // 处理文件导入
  const handleFileImport = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    setImportLoading(true);

    try {
      const response = await todoApi.importTodosFile(file);

      if (response.success) {
        const result = response.data;
        setError(`成功导入 ${result.imported} 条待办事项`);
        // 刷新数据
        await fetchTodos(true);
        await fetchStats(true);
      } else {
        setError(response.error?.message || '导入失败');
      }
    } catch (err: any) {
      setError(extractErrorMessage(err, '导入失败'));
    } finally {
      setImportLoading(false);
      // 重置 input，允许再次选择同一文件
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
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

        {/* 工具栏 */}
        <div className="toolbar">
          <div className="toolbar-left">
            <button
              className={`btn btn-sm ${selectionMode ? 'btn-primary' : 'btn-secondary'}`}
              onClick={toggleSelectionMode}
            >
              {selectionMode ? '退出多选' : '多选模式'}
            </button>

            {selectionMode && (
              <>
                <button
                  className="btn btn-sm btn-secondary"
                  onClick={handleSelectAll}
                  disabled={filteredTodos.length === 0}
                >
                  {filteredTodos.length > 0 && filteredTodos.every(t => selectedIds.has(t.id))
                    ? '取消全选'
                    : '全选'}
                </button>
                <span className="selection-count">
                  已选 {selectedIds.size} 项
                </span>
              </>
            )}
          </div>

          <div className="toolbar-right">
            {selectionMode ? (
              <>
                <button
                  className="btn btn-sm btn-success"
                  onClick={() => requestBatchAction('complete')}
                  disabled={selectedIds.size === 0 || batchLoading}
                >
                  {batchLoading ? '处理中...' : '批量完成'}
                </button>
                <button
                  className="btn btn-sm btn-danger"
                  onClick={() => requestBatchAction('delete')}
                  disabled={selectedIds.size === 0 || batchLoading}
                >
                  {batchLoading ? '处理中...' : '批量删除'}
                </button>
              </>
            ) : (
              <>
                <button
                  className="btn btn-sm btn-secondary"
                  onClick={handleExportJSON}
                >
                  导出 JSON
                </button>
                <button
                  className="btn btn-sm btn-secondary"
                  onClick={handleExportCSV}
                >
                  导出 CSV
                </button>
                <button
                  className="btn btn-sm btn-primary"
                  onClick={handleImportClick}
                  disabled={importLoading}
                >
                  {importLoading ? '导入中...' : '导入'}
                </button>
                <input
                  ref={fileInputRef}
                  type="file"
                  accept=".json,.csv"
                  onChange={handleFileImport}
                  style={{ display: 'none' }}
                />
              </>
            )}
          </div>
        </div>

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
                    onUpdate={handleTodoUpdated}
                    isLeaving={leavingIds.has(todo.id)}
                    selectionMode={selectionMode}
                    isSelected={selectedIds.has(todo.id)}
                    onSelect={handleSelect}
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

      {/* 批量操作确认弹窗 */}
      <ConfirmDialog
        isOpen={batchConfirm.isOpen}
        title={batchConfirm.action === 'delete' ? '批量删除确认' : '批量完成确认'}
        message={
          batchConfirm.action === 'delete'
            ? `确定要删除选中的 ${selectedIds.size} 个待办事项吗？此操作无法撤销。`
            : `确定要将选中的 ${selectedIds.size} 个待办事项标记为已完成吗？`
        }
        confirmText={batchConfirm.action === 'delete' ? '批量删除' : '批量完成'}
        cancelText="取消"
        variant={batchConfirm.action === 'delete' ? 'danger' : 'info'}
        onConfirm={handleBatchConfirm}
        onCancel={cancelBatchAction}
      />
    </div>
  );
};

export default TodoPage;
