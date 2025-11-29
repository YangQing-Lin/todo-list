import { useState, useEffect } from 'react';
import { Todo } from '../types';
import { todoApi } from '../services/api';

interface UseTodoUpdateOptions {
  onSuccess?: (todo: Todo) => void;
  onError?: (error: string) => void;
}

export function useTodoUpdate(options: UseTodoUpdateOptions = {}) {
  const { onSuccess, onError } = options;

  // 从 localStorage 获取草稿
  const getDraft = (todoId: number): Partial<Todo> | null => {
    try {
      const draft = localStorage.getItem(`todo-draft-${todoId}`);
      return draft ? JSON.parse(draft) : null;
    } catch {
      return null;
    }
  };

  // 保存草稿到 localStorage
  const saveDraft = (todoId: number, data: Partial<Todo>) => {
    try {
      localStorage.setItem(`todo-draft-${todoId}`, JSON.stringify(data));
    } catch (error) {
      console.warn('Failed to save draft:', error);
    }
  };

  // 清除草稿
  const clearDraft = (todoId: number) => {
    localStorage.removeItem(`todo-draft-${todoId}`);
  };

  // 更新 Todo（带版本冲突处理）
  const updateTodo = async (
    id: number,
    currentTodo: Todo,
    changes: Partial<Todo>
  ): Promise<boolean> => {
    // 先保存草稿
    saveDraft(id, changes);

    try {
      // 发送更新请求（包含 version）
      const response = await todoApi.updateTodo(id, {
        ...changes,
        version: currentTodo.version,
      });

      if (response.success) {
        // 成功：清除草稿，调用成功回调
        clearDraft(id);
        onSuccess?.(response.data);
        return true;
      } else {
        // 失败：保留草稿
        onError?.(response.error?.message || '更新失败');
        return false;
      }
    } catch (err: any) {
      // 检查是否是版本冲突
      if (err.response?.status === 409) {
        await handleVersionConflict(id, currentTodo, changes);
        return false;
      } else {
        // 其他错误：保留草稿
        const message = err.response?.data?.error?.message || '更新失败';
        onError?.(message);
        return false;
      }
    }
  };

  // 处理版本冲突
  const handleVersionConflict = async (
    id: number,
    localTodo: Todo,
    localChanges: Partial<Todo>
  ) => {
    try {
      // 获取服务器最新版本
      const response = await todoApi.getTodos();
      if (!response.success) {
        onError?.('无法获取最新数据');
        return;
      }

      const serverTodo = response.data.todos.find(t => t.id === id);
      if (!serverTodo) {
        onError?.('该待办事项已被删除');
        clearDraft(id);
        return;
      }

      // 构建冲突提示信息
      const conflicts: string[] = [];
      if (localChanges.title && localChanges.title !== serverTodo.title) {
        conflicts.push(`标题:\n  你的修改: "${localChanges.title}"\n  服务器版本: "${serverTodo.title}"`);
      }
      if (localChanges.description && localChanges.description !== serverTodo.description) {
        conflicts.push(`描述:\n  你的修改: "${localChanges.description}"\n  服务器版本: "${serverTodo.description}"`);
      }

      const conflictMessage = conflicts.length > 0
        ? `检测到冲突:\n${conflicts.join('\n\n')}`
        : '数据已被其他人修改';

      // 询问用户如何处理
      const useLocal = window.confirm(
        `⚠️ ${conflictMessage}\n\n` +
        `点击"确定"使用你的版本（覆盖服务器）\n` +
        `点击"取消"使用服务器版本（你的草稿已保存，可稍后恢复）`
      );

      if (useLocal) {
        // 用户选择使用本地版本：用最新的 version 重新提交
        try {
          const retryResponse = await todoApi.updateTodo(id, {
            ...localChanges,
            version: serverTodo.version,
          });

          if (retryResponse.success) {
            clearDraft(id);
            onSuccess?.(retryResponse.data);
          } else {
            onError?.('保存失败，草稿已保留');
          }
        } catch (retryErr: any) {
          onError?.('保存失败，草稿已保留');
        }
      } else {
        // 用户选择使用服务器版本：草稿保留但不自动应用
        onError?.('已使用服务器版本，你的草稿已保存');
        onSuccess?.(serverTodo);
      }
    } catch (err: any) {
      onError?.('处理冲突失败，草稿已保留');
    }
  };

  return {
    updateTodo,
    getDraft,
    saveDraft,
    clearDraft,
  };
}
