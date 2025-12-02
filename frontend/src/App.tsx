import { AnimatePresence } from 'framer-motion';
import TodoPage from './pages/TodoPage';
import './styles/App.css';

function App() {
  return (
    <AnimatePresence mode="wait" initial={false}>
      <div className="App" key="app-shell">
        <TodoPage />
      </div>
    </AnimatePresence>
  );
}

export default App;
