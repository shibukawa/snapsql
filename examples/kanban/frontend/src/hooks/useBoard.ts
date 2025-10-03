import { useState, useEffect, useCallback } from 'react';
import type { BoardTree } from '../api/types';
import * as api from '../api/kanban';
import { calculatePosition } from '../utils/position';

export function useBoard(boardId: number | null) {
  const [boardTree, setBoardTree] = useState<BoardTree | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadBoard = useCallback(async () => {
    if (!boardId) {
      setBoardTree(null);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const tree = await api.fetchBoardTree(boardId);
      setBoardTree(tree);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load board');
      console.error('Failed to load board:', err);
    } finally {
      setLoading(false);
    }
  }, [boardId]);

  useEffect(() => {
    void loadBoard();
  }, [loadBoard]);

  const createCard = useCallback(
    async (title: string, description: string = '') => {
      if (!boardTree || boardTree.lists.length === 0) return;

      // Get the first list (backlog) to calculate position
      const firstList = boardTree.lists[0];
      const position = calculatePosition(firstList.cards, firstList.cards.length);

      try {
        await api.createCard({ title, description, position });
        await loadBoard();
      } catch (err) {
        console.error('Failed to create card:', err);
        throw err;
      }
    },
    [boardTree, loadBoard]
  );

  const updateCard = useCallback(
    async (cardId: number, title: string, description: string) => {
      try {
        await api.updateCard(cardId, { title, description });
        await loadBoard();
      } catch (err) {
        console.error('Failed to update card:', err);
        throw err;
      }
    },
    [loadBoard]
  );

  const moveCard = useCallback(
    async (cardId: number, targetListId: number, targetIndex: number) => {
      if (!boardTree) return;

      const targetList = boardTree.lists.find((l) => l.id === targetListId);
      if (!targetList) return;

      // Calculate new position
      const newCards = targetList.cards.filter((c) => c.id !== cardId);
      const position = calculatePosition(newCards, targetIndex);

      try {
        await api.moveCard({ cardId, targetListId, targetPosition: position });
        await loadBoard();
      } catch (err) {
        console.error('Failed to move card:', err);
        throw err;
      }
    },
    [boardTree, loadBoard]
  );

  const getDoneStageOrder = useCallback((): number => {
    if (!boardTree || boardTree.lists.length === 0) return 999;
    return Math.max(...boardTree.lists.map((l) => l.stageOrder));
  }, [boardTree]);

  return {
    boardTree,
    loading,
    error,
    loadBoard,
    createCard,
    updateCard,
    moveCard,
    getDoneStageOrder,
  };
}
