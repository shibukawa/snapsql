import { useState } from 'react';
import { useBoard } from '../hooks/useBoard';
import { StageColumn } from './StageColumn';
import { CardDetailDrawer } from './CardDetailDrawer';
import { CreateCardDialog } from './Dialog';
import type { CardItem } from '../api/types';
import * as api from '../api/kanban';

interface BoardViewProps {
  boardId: number;
  onCompleteIteration: () => void;
}

export function BoardView({ boardId, onCompleteIteration }: BoardViewProps) {
  const { boardTree, loading, error, createCard, updateCard, moveCard, getDoneStageOrder } =
    useBoard(boardId);

  const [selectedCardId, setSelectedCardId] = useState<number | null>(null);
  const [draggingCard, setDraggingCard] = useState<CardItem | null>(null);
  const [createCardDialog, setCreateCardDialog] = useState<{
    listName: string;
  } | null>(null);

  const handleCardClick = (card: CardItem) => {
    console.log('Card clicked:', card);
    console.log('Setting selectedCardId to:', card.id);
    setSelectedCardId(card.id);
  };

  const handleCardDrop = async (cardId: number, sourceListId: number, targetListId: number) => {
    console.log(`Moving card ${cardId} from list ${sourceListId} to list ${targetListId}`);
    
    if (!boardTree) return;
    
    const targetList = boardTree.lists.find((l) => l.id === targetListId);
    if (!targetList) return;
    
    // Add card to the end of the target list
    const targetIndex = targetList.cards.length;
    
    try {
      await moveCard(cardId, targetListId, targetIndex);
      setDraggingCard(null);
    } catch (err) {
      console.error('Failed to move card:', err);
      alert('Failed to move card. Please try again.');
    }
  };

  const handleDragStart = (card: CardItem) => {
    setDraggingCard(card);
  };

  const handleDragEnd = () => {
    setDraggingCard(null);
  };

  const handleCreateCard = () => {
    if (!boardTree || boardTree.lists.length === 0) return;
    
    // Always add to the first list (Backlog)
    const firstList = boardTree.lists[0];
    setCreateCardDialog({ listName: firstList.name });
  };

  const handleCreateCardSubmit = async (title: string, description: string) => {
    await createCard(title, description);
  };

  const handleCompleteIteration = async () => {
    if (!confirm('Complete this iteration and create a new board?')) {
      return;
    }

    try {
      await api.createBoard(`Board ${new Date().toISOString().split('T')[0]}`);
      onCompleteIteration();
    } catch (err) {
      console.error('Failed to complete iteration:', err);
      alert('Failed to complete iteration. Please try again.');
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <p className="text-lg text-gray-600">Loading board...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <p className="text-lg text-red-600 mb-2">Error loading board</p>
          <p className="text-sm text-gray-600">{error}</p>
        </div>
      </div>
    );
  }

  if (!boardTree) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <p className="text-lg text-gray-600">Board not found</p>
      </div>
    );
  }

  const doneStageOrder = getDoneStageOrder();

  // Find the selected card from the current board tree
  const selectedCard = selectedCardId
    ? boardTree.lists
        .flatMap((list) => list.cards)
        .find((card) => card.id === selectedCardId) ?? null
    : null;

  console.log('selectedCardId:', selectedCardId);
  console.log('selectedCard:', selectedCard);
  console.log('Will show drawer:', !!(selectedCard && boardTree));

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-white shadow-sm border-b border-gray-200 px-6 py-4">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">{boardTree.board.name}</h1>
            <p className="text-sm text-gray-500 mt-1">
              Status: {boardTree.board.status}
            </p>
          </div>
          <div className="flex gap-3">
            <button
              onClick={handleCreateCard}
              className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
            >
              + Add Card
            </button>
            <button
              onClick={handleCompleteIteration}
              className="px-4 py-2 bg-green-600 text-white rounded-md hover:bg-green-700 transition-colors"
            >
              Complete Iteration
            </button>
          </div>
        </div>
      </div>

      {/* Board Content */}
      <div className="p-6 overflow-x-auto">
        <div className="flex gap-4">
          {boardTree.lists.map((list) => (
            <StageColumn
              key={list.id}
              list={list}
              isDone={list.stageOrder === doneStageOrder}
              onCardClick={handleCardClick}
              onCardDrop={handleCardDrop}
              draggingCard={draggingCard}
              onDragStart={handleDragStart}
              onDragEnd={handleDragEnd}
            />
          ))}
        </div>
      </div>

      {/* Card Detail Drawer */}
      {selectedCard && boardTree && (
        <CardDetailDrawer
          card={selectedCard}
          lists={boardTree.lists}
          onClose={() => setSelectedCardId(null)}
          onUpdate={updateCard}
          onMove={moveCard}
        />
      )}

      {/* Create Card Dialog */}
      {createCardDialog && (
        <CreateCardDialog
          isOpen={true}
          listName={createCardDialog.listName}
          onClose={() => setCreateCardDialog(null)}
          onCreate={handleCreateCardSubmit}
        />
      )}
    </div>
  );
}
