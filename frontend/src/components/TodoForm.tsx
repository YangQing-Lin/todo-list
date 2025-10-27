import React, { useState } from 'react';
import { todoApi } from '../services/api';
import '../styles/TodoForm.css';

interface TodoFormProps {
  onTodoCreated: () => void;
}

const TodoForm: React.FC<TodoFormProps> = ({ onTodoCreated }) => {
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!title.trim()) {
      setError('请输入待办事项标题');
      return;
    }

    setLoading(true);
    setError('');

    try {
      await todoApi.createTodo({
        title: title.trim(),
        description: description.trim() || undefined,
      });

      setTitle('');
      setDescription('');
      onTodoCreated();
    } catch (err: any) {
      setError(err.response?.data?.error?.message || '创建失败，请重试');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="todo-form card">
      <h3>添加新的待办事项</h3>
      {error && <div className="error">{error}</div>}
      <form onSubmit={handleSubmit}>
        <div className="form-group">
          <input
            type="text"
            className="form-input"
            placeholder="输入标题..."
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            disabled={loading}
            maxLength={100}
          />
        </div>
        <div className="form-group">
          <textarea
            className="form-textarea"
            placeholder="添加描述（可选）..."
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            disabled={loading}
            rows={3}
            maxLength={500}
          />
        </div>
        <button
          type="submit"
          className="btn btn-primary"
          disabled={loading || !title.trim()}
        >
          {loading ? '添加中...' : '添加'}
        </button>
      </form>
    </div>
  );
};

export default TodoForm;