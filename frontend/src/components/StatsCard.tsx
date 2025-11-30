import React, { memo } from 'react';
import { TodoStats } from '../types';
import '../styles/StatsCard.css';

interface StatsCardProps {
  stats: TodoStats | null;
  loading: boolean;
}

interface StatItemProps {
  value: number;
  label: string;
  variant: string;
}

// 单个统计项组件，使用 memo 避免不必要的重渲染
const StatItem = memo<StatItemProps>(({ value, label, variant }) => (
  <div className={`stat-card stat-card-${variant}`}>
    <div className="stat-value">{value}</div>
    <div className="stat-label">{label}</div>
  </div>
));

StatItem.displayName = 'StatItem';

const StatsCard: React.FC<StatsCardProps> = ({ stats, loading }) => {
  // 只在首次加载且没有数据时显示 loading
  if (loading && !stats) {
    return (
      <div className="stats-container">
        <div className="stats-loading">加载统计数据中...</div>
      </div>
    );
  }

  // 有数据时即使在刷新也显示现有数据（避免闪烁）
  if (!stats) {
    return (
      <div className="stats-container">
        <div className="stats-loading">暂无数据</div>
      </div>
    );
  }

  return (
    <div className="stats-container">
      <div className="stats-grid">
        <StatItem value={stats.total} label="总任务" variant="primary" />
        <StatItem value={stats.pending} label="待完成" variant="warning" />
        <StatItem value={stats.completed} label="已完成" variant="success" />
        <StatItem value={stats.overdue} label="已逾期" variant="danger" />
        <StatItem value={stats.today} label="今天到期" variant="info" />
        <StatItem value={stats.this_week} label="本周到期" variant="accent" />
      </div>
    </div>
  );
};

export default memo(StatsCard);
