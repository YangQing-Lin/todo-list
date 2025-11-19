import React from 'react';
import { TodoStats } from '../types';
import '../styles/StatsCard.css';

interface StatsCardProps {
  stats: TodoStats | null;
  loading: boolean;
}

const StatsCard: React.FC<StatsCardProps> = ({ stats, loading }) => {
  if (loading || !stats) {
    return (
      <div className="stats-container">
        <div className="stats-loading">加载统计数据中...</div>
      </div>
    );
  }

  return (
    <div className="stats-container">
      <div className="stats-grid">
        <div className="stat-card stat-card-primary">
          <div className="stat-value">{stats.total}</div>
          <div className="stat-label">总任务</div>
        </div>

        <div className="stat-card stat-card-warning">
          <div className="stat-value">{stats.pending}</div>
          <div className="stat-label">待完成</div>
        </div>

        <div className="stat-card stat-card-success">
          <div className="stat-value">{stats.completed}</div>
          <div className="stat-label">已完成</div>
        </div>

        <div className="stat-card stat-card-danger">
          <div className="stat-value">{stats.overdue}</div>
          <div className="stat-label">已逾期</div>
        </div>

        <div className="stat-card stat-card-info">
          <div className="stat-value">{stats.today}</div>
          <div className="stat-label">今天到期</div>
        </div>

        <div className="stat-card stat-card-accent">
          <div className="stat-value">{stats.this_week}</div>
          <div className="stat-label">本周到期</div>
        </div>
      </div>
    </div>
  );
};

export default StatsCard;
