import React, { useState, useEffect, useRef } from 'react';
import { Todo, getDefaultColorById } from '../types';
import { useTodoUpdate } from '../hooks/useTodoUpdate';
import '../styles/TodoItem.css';

interface TodoItemProps {
  todo: Todo;
  onToggle: (id: number) => void;
  onDelete: (id: number) => void;
  onUpdate?: (todo: Todo) => void;
  isLeaving?: boolean;
}

const TodoItem: React.FC<TodoItemProps> = ({ todo, onToggle, onDelete, onUpdate, isLeaving = false }) => {
  const [isEditing, setIsEditing] = useState(false);
  const [editTitle, setEditTitle] = useState(todo.title);
  const [editDescription, setEditDescription] = useState(todo.description);
  const [error, setError] = useState('');
  const titleInputRef = useRef<HTMLInputElement>(null);
  const saveTimeoutRef = useRef<number>();

  const { updateTodo, getDraft, clearDraft } = useTodoUpdate({
    onSuccess: (updatedTodo) => {
      setIsEditing(false);
      setError('');
      onUpdate?.(updatedTodo);
    },
    onError: (errorMessage) => {
      setError(errorMessage);
    },
  });

  // 组件挂载时检查是否有草稿
  useEffect(() => {
    const draft = getDraft(todo.id);
    if (draft && (draft.title || draft.description)) {
      const restore = window.confirm(
        `检测到未保存的草稿:\n标题: ${draft.title || '(未修改)'}\n\n是否恢复？`
      );
      if (restore) {
        setEditTitle(draft.title || todo.title);
        setEditDescription(draft.description || todo.description);
        setIsEditing(true);
      } else {
        clearDraft(todo.id);
      }
    }
  }, [todo.id]);

  // 编辑模式激活时聚焦输入框
  useEffect(() => {
    if (isEditing && titleInputRef.current) {
      titleInputRef.current.focus();
    }
  }, [isEditing]);

  // 清理定时器
  useEffect(() => {
    return () => {
      if (saveTimeoutRef.current) {
        clearTimeout(saveTimeoutRef.current);
      }
    };
  }, []);
  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  // 进入编辑模式
  const handleEdit = () => {
    setEditTitle(todo.title);
    setEditDescription(todo.description);
    setError('');
    setIsEditing(true);
  };

  // 保存编辑
  const handleSave = async () => {
    if (!editTitle.trim()) {
      setError('标题不能为空');
      return;
    }

    const success = await updateTodo(todo.id, todo, {
      title: editTitle.trim(),
      description: editDescription.trim(),
    });

    // 如果成功，onSuccess 回调会自动关闭编辑模式
  };

  // 取消编辑
  const handleCancel = () => {
    setEditTitle(todo.title);
    setEditDescription(todo.description);
    setError('');
    setIsEditing(false);
    clearDraft(todo.id);
  };

  // 处理键盘事件
  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && e.ctrlKey) {
      handleSave();
    } else if (e.key === 'Escape') {
      handleCancel();
    }
  };

  const isCompleted = todo.status === 'completed';
  // 使用 Todo 自带颜色，旧数据则根据 ID 计算稳定默认色
  const backgroundColor = todo.color || getDefaultColorById(todo.id);

  return (
    <div
      className={`todo-item ${isCompleted ? 'completed' : ''} ${isLeaving ? 'is-leaving' : ''} ${isEditing ? 'editing' : ''}`}
      style={{ backgroundColor: isCompleted ? undefined : backgroundColor }}
    >
      {error && (
        <div className="todo-error">
          ⚠️ {error}
          <button onClick={() => setError('')}>×</button>
        </div>
      )}

      <div className="todo-content">
        <div className="todo-left">
          <input
            type="checkbox"
            className="todo-checkbox"
            checked={isCompleted}
            onChange={() => onToggle(todo.id)}
            disabled={isEditing}
          />

          {isEditing ? (
            <div className="todo-edit-form" onKeyDown={handleKeyDown}>
              <input
                ref={titleInputRef}
                type="text"
                className="todo-edit-input"
                value={editTitle}
                onChange={(e) => setEditTitle(e.target.value)}
                placeholder="标题"
              />
              <textarea
                className="todo-edit-textarea"
                value={editDescription}
                onChange={(e) => setEditDescription(e.target.value)}
                placeholder="描述（可选）"
                rows={2}
              />
              <div className="todo-edit-hint">
                提示: Ctrl+Enter 保存，Esc 取消
              </div>
            </div>
          ) : (
            <div className="todo-text" onDoubleClick={handleEdit}>
              <h4 className="todo-title">{todo.title}</h4>
              {todo.description && (
                <p className="todo-description">{todo.description}</p>
              )}
              <div className="todo-meta">
                <span className="todo-date">
                  创建于: {formatDate(todo.created_at)}
                </span>
                {todo.completed_at && (
                  <span className="todo-completed">
                    完成于: {formatDate(todo.completed_at)}
                  </span>
                )}
              </div>
            </div>
          )}
        </div>

        <div className="todo-actions">
          {isEditing ? (
            <>
              <button
                className="btn btn-primary btn-sm"
                onClick={handleSave}
              >
                保存
              </button>
              <button
                className="btn btn-secondary btn-sm"
                onClick={handleCancel}
              >
                取消
              </button>
            </>
          ) : (
            <>
              <button
                className="btn btn-secondary btn-sm"
                onClick={handleEdit}
              >
                编辑
              </button>
              <button
                className="btn btn-danger btn-sm"
                onClick={() => onDelete(todo.id)}
              >
                删除
              </button>
            </>
          )}
        </div>
      </div>
    </div>
  );
};

export default TodoItem;
