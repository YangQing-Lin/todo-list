import React, { memo } from 'react';
import { TodoStats } from '../types';
import '../styles/StatsCard.css';

interface StatsCardProps {
  stats: TodoStats | null;
  loading: boolean;
  refreshing?: boolean;
}

interface StatItemProps {
  value: number;
  label: string;
  variant: string;
  refreshing: boolean;
}

// 单个统计项组件，使用 memo 避免不必要的重渲染
const StatItem = memo<StatItemProps>(({ value, label, variant, refreshing }) => {
  const classes = [
    'stat-card',
    `stat-card-${variant}`,
    refreshing ? 'stat-card-refreshing' : '',
  ].filter(Boolean).join(' ');

  return (
    <div className={classes}>
      <div className="stat-value">{value}</div>
      <div className="stat-label">{label}</div>
    </div>
  );
});

StatItem.displayName = 'StatItem';

const StatsCard: React.FC<StatsCardProps> = ({ stats, loading, refreshing = false }) => {
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
    <div className={`stats-container ${refreshing ? 'stats-container-refreshing' : ''}`}>
      <div className={`stats-grid ${refreshing ? 'is-refreshing' : ''}`}>
        <StatItem value={stats.total} label="总任务" variant="primary" refreshing={refreshing} />
        <StatItem value={stats.pending} label="待完成" variant="warning" refreshing={refreshing} />
        <StatItem value={stats.completed} label="已完成" variant="success" refreshing={refreshing} />
        <StatItem value={stats.overdue} label="已逾期" variant="danger" refreshing={refreshing} />
        <StatItem value={stats.today} label="今天到期" variant="info" refreshing={refreshing} />
        <StatItem value={stats.this_week} label="本周到期" variant="accent" refreshing={refreshing} />
      </div>
    </div>
  );
};

export default memo(StatsCard);
