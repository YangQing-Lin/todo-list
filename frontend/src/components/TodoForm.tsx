import React, { useState } from 'react';
import { AnimatePresence, motion } from 'framer-motion';
import { todoApi } from '../services/api';
import { getRandomColor } from '../types';
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
        color: getRandomColor(),
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
      <AnimatePresence>
        {error && (
          <motion.div
            className="error"
            initial={{ opacity: 0, y: -10 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -6 }}
            transition={{ type: 'spring', stiffness: 320, damping: 20 }}
          >
            {error}
          </motion.div>
        )}
      </AnimatePresence>
      <form onSubmit={handleSubmit}>
        <div className="form-group">
          <motion.input
            type="text"
            className="form-input"
            placeholder="输入标题..."
            value={title}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) => setTitle(e.target.value)}
            disabled={loading}
            maxLength={100}
            whileFocus={{
              scale: 1.02,
              boxShadow: '10px 10px 0 var(--neo-pink)',
            }}
            transition={{ type: 'spring', stiffness: 320, damping: 24 }}
          />
        </div>
        <div className="form-group">
          <motion.textarea
            className="form-textarea"
            placeholder="添加描述（可选）..."
            value={description}
            onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) => setDescription(e.target.value)}
            disabled={loading}
            rows={3}
            maxLength={500}
            whileFocus={{
              scale: 1.02,
              boxShadow: '10px 10px 0 var(--neo-pink)',
            }}
            transition={{ type: 'spring', stiffness: 320, damping: 24 }}
          />
        </div>
        <motion.button
          type="submit"
          className="btn btn-primary"
          disabled={loading || !title.trim()}
          whileTap={{ scale: 0.95 }}
          transition={{ type: 'spring', stiffness: 420, damping: 35 }}
        >
          {loading ? '添加中...' : '添加'}
        </motion.button>
      </form>
    </div>
  );
};

export default TodoForm;
