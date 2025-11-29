import React from 'react';
import { Todo, getDefaultColorById } from '../types';
import '../styles/TodoItem.css';

interface TodoItemProps {
  todo: Todo;
  onToggle: (id: number) => void;
  onDelete: (id: number) => void;
  isLeaving?: boolean;
}

const TodoItem: React.FC<TodoItemProps> = ({ todo, onToggle, onDelete, isLeaving = false }) => {
  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const isCompleted = todo.status === 'completed';
  // 使用 Todo 自带颜色，旧数据则根据 ID 计算稳定默认色
  const backgroundColor = todo.color || getDefaultColorById(todo.id);

  return (
    <div
      className={`todo-item ${isCompleted ? 'completed' : ''} ${isLeaving ? 'is-leaving' : ''}`}
      style={{ backgroundColor: isCompleted ? undefined : backgroundColor }}
    >
      <div className="todo-content">
        <div className="todo-left">
          <input
            type="checkbox"
            className="todo-checkbox"
            checked={isCompleted}
            onChange={() => onToggle(todo.id)}
          />
          <div className="todo-text">
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
        </div>
        <div className="todo-actions">
          <button
            className="btn btn-danger btn-sm"
            onClick={() => onDelete(todo.id)}
          >
            删除
          </button>
        </div>
      </div>
    </div>
  );
};

export default TodoItem;
