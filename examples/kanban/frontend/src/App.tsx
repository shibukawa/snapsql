import { useState, useEffect } from 'react';
import { BoardView } from './components/BoardView';
import * as api from './api/kanban';
import type { BoardSummary } from './api/types';
import './App.css';

function App() {
  const [activeBoard, setActiveBoard] = useState<BoardSummary | null>(null);
  const [loading, setLoading] = useState(true);

  const loadBoards = async () => {
    setLoading(true);
    try {
      const data = await api.fetchBoards();

      // Find the active board (status === 'active')
      const active = data.find((b) => b.status === 'active');
      setActiveBoard(active || null);
    } catch (err) {
      console.error('Failed to load boards:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void loadBoards();
  }, []);

  const handleCompleteIteration = () => {
    void loadBoards();
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen bg-gray-50">
        <p className="text-lg text-gray-600">Loading...</p>
      </div>
    );
  }

  if (!activeBoard) {
    return (
      <div className="flex items-center justify-center min-h-screen bg-gray-50">
        <div className="text-center">
          <p className="text-lg text-gray-600 mb-4">No active board found</p>
          <button
            onClick={async () => {
              try {
                await api.createBoard('Initial Board');
                await loadBoards();
              } catch (err) {
                console.error('Failed to create board:', err);
              }
            }}
            className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
          >
            Create Initial Board
          </button>
        </div>
      </div>
    );
  }

  return <BoardView boardId={activeBoard.id} onCompleteIteration={handleCompleteIteration} />;
}

export default App;
