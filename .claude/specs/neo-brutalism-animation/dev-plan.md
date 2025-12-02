# 前端 Neo-Brutalism 风格强化 + Framer Motion 动画交互 - 开发计划

## 概述
为 Todo List 前端应用引入柔和现代的 Neo-Brutalism 设计语言（适中圆角、柔和配色、粗边框、硬阴影），并通过 Framer Motion 实现流畅的微交互、列表过渡与页面动画，全面提升用户体验与视觉吸引力。

## 任务拆解

### 任务 1：Neo-Brutalism 设计令牌升级
- **ID**: task-1
- **描述**:
  - 调整设计令牌以实现柔和现代风格：引入适中圆角（4px-12px）、优化阴影层级、微调配色饱和度
  - 更新全局背景图形（可选微妙纹理或几何装饰）
  - 优化排版节奏（标题/正文尺寸、行高、字重层级）
  - 确保响应式设计在移动端（320px+）、平板（768px+）、桌面（1024px+）上均友好
- **文件范围**:
  - `frontend/src/styles/index.css` (设计令牌：圆角、阴影、颜色变量)
  - `frontend/src/styles/App.css` (全局布局、背景装饰)
- **依赖关系**: 无
- **测试命令**:
  ```bash
  cd /Users/yangqing/Projects/todo-list/frontend && npm run build
  cd /Users/yangqing/Projects/todo-list/frontend && npm run dev
  ```
- **测试重点**:
  - 在 Chrome/Firefox/Safari 检查视觉一致性
  - 使用开发者工具模拟移动设备（iPhone SE、iPad、大屏手机）验证响应式布局
  - 确认圆角、阴影、配色符合设计基线（参考 https://www.neobrutalism.dev/ 示例）
  - 检查排版在不同屏幕尺寸下的可读性

---

### 任务 2：Framer Motion 基础架构搭建
- **ID**: task-2
- **描述**:
  - 安装 `framer-motion` 依赖（`npm install framer-motion`）
  - 在 `App.tsx` 或 `TodoPage.tsx` 建立 `AnimatePresence` 外壳，为路由/弹窗过渡做准备
  - 定义全局 motion 变体配置（如 `fadeIn`, `slideUp`, `staggerContainer`）
  - 规划动效节奏参数（duration、easing、stagger delay）并与 CSS 变量对齐
  - 为列表容器添加 `layout` 动画基础（`layout="position"` 或 `layoutId`）
- **文件范围**:
  - `frontend/package.json` (依赖)
  - `frontend/src/App.tsx` (AnimatePresence 外壳)
  - `frontend/src/pages/TodoPage.tsx` (列表容器 motion wrapper)
  - `frontend/src/styles/index.css` (可选：新增动效相关 CSS 变量)
- **依赖关系**: 依赖 task-1（需色板与阴影基线定义完成）
- **测试命令**:
  ```bash
  cd /Users/yangqing/Projects/todo-list/frontend && npm run build
  cd /Users/yangqing/Projects/todo-list/frontend && npm run dev
  ```
- **测试重点**:
  - 验证 `framer-motion` 安装成功（无构建错误）
  - 检查 `AnimatePresence` 是否正确包裹目标组件（可通过 React DevTools 确认）
  - 测试基础 motion 变体（如 `motion.div` 的 `initial/animate/exit` 是否生效）
  - 确认无性能问题（Chrome DevTools Performance 面板无长帧卡顿）

---

### 任务 3：列表与 TodoItem 微交互增强
- **ID**: task-3
- **描述**:
  - 为过滤按钮切换添加 `AnimatePresence` + `motion.div` 的淡入/滑动过渡
  - 为 Todo 列表添加 stagger 动画（子项依次进入，使用 `staggerChildren`）
  - 为 `TodoItem` 添加 hover/drag/完成状态的 motion 交互：
    - Hover: 轻微上浮 + 阴影扩大（`whileHover={{ y: -4, boxShadow: "..." }}`）
    - 完成切换: 淡出 + 缩小动画（`animate={{ opacity: 0.6, scale: 0.98 }}`）
    - 删除退出: 滑出 + 淡出（`exit={{ x: -100, opacity: 0 }}`）
  - 与现有 `isLeaving` 状态对齐或替换（保留 260ms 退出时长）
  - 为多选模式下的选中反馈添加 scale/color 动画
- **文件范围**:
  - `frontend/src/pages/TodoPage.tsx` (列表容器、过滤按钮)
  - `frontend/src/components/TodoItem.tsx` (条目微交互)
  - `frontend/src/styles/TodoItem.css` (可选：调整 hover 样式)
  - `frontend/src/styles/TodoPage.css` (过滤按钮动画)
- **依赖关系**: 依赖 task-2（需 AnimatePresence 与 motion 变体基础）
- **测试命令**:
  ```bash
  cd /Users/yangqing/Projects/todo-list/frontend && npm run dev
  ```
- **测试重点**:
  - 测试过滤切换（all → pending → completed）时列表过渡流畅度
  - 验证新建 Todo 时 stagger 进入效果（多个 Todo 依次出现）
  - 测试单个 TodoItem 的 hover、完成、删除动画是否自然
  - 检查多选模式下选中反馈的响应性（checkbox 切换 + 卡片缩放）
  - 确认退出动画与数据更新同步（避免闪烁或重复渲染）

---

### 任务 4：TodoForm 与 ConfirmDialog 体验优化
- **ID**: task-4
- **描述**:
  - 为 `TodoForm` 添加 motion 动效：
    - 输入框 focus 时轻微缩放 + 阴影扩大（`whileFocus={{ scale: 1.02, boxShadow: "..." }}`）
    - 提交按钮 hover/tap 状态动画（`whileTap={{ scale: 0.95 }}`）
    - 错误提示淡入/弹跳效果（`initial={{ opacity: 0, y: -10 }}`）
  - 为 `ConfirmDialog` 添加弹窗过渡：
    - 背景遮罩淡入（`initial={{ opacity: 0 }}`）
    - 对话框从中心弹出 + 轻微回弹（`initial={{ scale: 0.9 }}`, `animate={{ scale: 1 }}`, `transition={{ type: "spring" }}`）
    - 关闭时缩小淡出（`exit={{ scale: 0.95, opacity: 0 }}`）
  - 更新按钮与输入框样式：引入软圆角（4px-8px）与层级阴影
- **文件范围**:
  - `frontend/src/components/TodoForm.tsx` (表单动画)
  - `frontend/src/components/ConfirmDialog.tsx` (弹窗动画)
  - `frontend/src/styles/TodoForm.css` (输入框/按钮样式)
  - `frontend/src/styles/ConfirmDialog.css` (对话框样式)
- **依赖关系**: 依赖 task-2（需 AnimatePresence 与 motion 变体基础）
- **测试命令**:
  ```bash
  cd /Users/yangqing/Projects/todo-list/frontend && npm run dev
  ```
- **测试重点**:
  - 测试输入框 focus 状态动画是否平滑（无卡顿）
  - 验证提交按钮 tap 反馈的触感（是否有按下效果）
  - 测试错误提示的进入/退出动画（显示和关闭时）
  - 验证 `ConfirmDialog` 打开/关闭时的弹出与退出效果
  - 检查 ESC 键关闭对话框时的动画完整性
  - 确认移动端触摸反馈一致性

---

### 任务 5：StatsCard 与工具栏动态效果增强
- **ID**: task-5
- **描述**:
  - 为 `StatsCard` 添加动态效果：
    - Hover 时数字放大 + 卡片轻微上浮（`whileHover={{ scale: 1.05, y: -2 }}`）
    - 数据刷新时 shimmer/pulse 动画（使用 `animate={{ opacity: [1, 0.8, 1] }}`）
    - 不同统计项使用不同颜色标识符（task-1 定义的配色）
  - 为工具栏按钮添加微交互：
    - 导入/导出按钮 hover 时图标晃动或高亮
    - 多选模式切换按钮的状态过渡（背景色 + 阴影变化）
    - 批量操作按钮的 loading 状态动画（spinner 或脉冲）
  - 优化工具栏响应式布局（移动端折叠或堆叠）
- **文件范围**:
  - `frontend/src/components/StatsCard.tsx` (统计卡片动画)
  - `frontend/src/pages/TodoPage.tsx` (工具栏按钮)
  - `frontend/src/styles/StatsCard.css` (卡片样式与动画)
  - `frontend/src/styles/TodoPage.css` (工具栏布局与按钮动画)
- **依赖关系**: 依赖 task-1（需主题配色与阴影定义）
- **测试命令**:
  ```bash
  cd /Users/yangqing/Projects/todo-list/frontend && npm run dev
  ```
- **测试重点**:
  - 测试 `StatsCard` hover 效果是否流畅
  - 验证数据刷新时的 shimmer 动画（可手动调用 `fetchStats()` 触发）
  - 测试工具栏按钮的 hover/tap 反馈
  - 检查批量操作按钮在 `batchLoading` 状态下的动画表现
  - 验证移动端（<768px）下工具栏布局是否合理（无溢出或重叠）
  - 确认多选模式切换按钮的状态指示清晰

---

## 执行顺序
1. **独立任务**: Task 1
2. **第二阶段**: Task 2（依赖 Task 1 完成后）
3. **第三阶段**: Task 3、Task 4、Task 5（可并行执行，均依赖 Task 2）

## 验收标准
- [ ] 设计令牌符合柔和现代 Neo-Brutalism 规范（适中圆角、硬阴影、粗边框、柔和配色）
- [ ] 响应式设计在移动端（320px+）、平板（768px+）、桌面（1024px+）均流畅运行
- [ ] Framer Motion 集成无构建错误，动画帧率 ≥60fps（无长帧卡顿）
- [ ] 列表过渡与 TodoItem 微交互自然流畅（stagger、hover、完成、删除动画）
- [ ] 表单与对话框交互体验优化（输入反馈、弹窗过渡、错误提示动画）
- [ ] 统计卡片与工具栏按钮具有明确的视觉反馈（hover、loading、状态切换）
- [ ] 所有动画与现有业务逻辑兼容（无数据不同步或闪烁问题）
- [ ] 无可访问性退步（键盘导航、屏幕阅读器兼容性保持）
- [ ] 构建产物大小增幅 ≤100KB（gzip 后）
- [ ] 全功能可通过 `npm run build` 构建成功，`npm run dev` 本地验证通过

## 技术要点
- **Framer Motion 最佳实践**:
  - 使用 `AnimatePresence` 管理列表/弹窗的进入/退出动画
  - 优先使用 `layout` 动画而非手动计算位置（利用 FLIP 技术）
  - 避免在长列表中滥用复杂 spring 动画（性能考虑）
  - 为 exit 动画显式设置 `mode="wait"` 或 `mode="sync"`

- **性能约束**:
  - 列表超过 50 项时考虑虚拟化（react-window）或分页
  - 动画使用 `transform` 和 `opacity`（GPU 加速），避免 `width`/`height` 过渡
  - 在 `motion` 组件上添加 `layoutId` 实现共享元素过渡

- **与现有代码集成**:
  - 保留现有 `isLeaving` 状态机制，或用 `AnimatePresence` + `exit` 完全替换
  - 确保 `leavingIds` 状态与动画时长（LEAVE_ANIMATION_MS）同步
  - 批量操作时避免触发单个条目的退出动画（性能优化）

- **样式系统**:
  - CSS 变量与 motion props 协同（如 `--shadow-lg` 映射到 `boxShadow`）
  - 优先使用 CSS 定义静态样式，motion props 仅控制动态属性
  - 保持样式模块化（每个组件独立 CSS 文件）

- **可访问性**:
  - 为动画添加 `prefers-reduced-motion` 媒体查询支持
  - 确保键盘导航不被动画阻断（focus 状态可见）
  - 对话框保持 `aria-modal`、`role="dialog"` 等属性

- **浏览器兼容性**:
  - 目标浏览器: Chrome 90+, Firefox 88+, Safari 14+, Edge 90+
  - Framer Motion 依赖 `IntersectionObserver`（需 polyfill for IE11，但本项目无需支持）
