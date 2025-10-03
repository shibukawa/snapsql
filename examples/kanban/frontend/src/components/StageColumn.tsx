import { useState } from 'react';
import { Card } from './Card';
import type { ListColumn, CardItem } from '../api/types';

interface StageColumnProps {
  list: ListColumn;
  isDone: boolean;
  onCardClick: (card: CardItem) => void;
  onCardDrop: (cardId: number, sourceListId: number, targetListId: number) => void;
  draggingCard: CardItem | null;
  onDragStart?: (card: CardItem) => void;
  onDragEnd?: () => void;
}

export function StageColumn({ 
  list, 
  isDone, 
  onCardClick, 
  onCardDrop, 
  draggingCard,
  onDragStart,
  onDragEnd,
}: StageColumnProps) {
  const [isDragOver, setIsDragOver] = useState(false);

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'move';
    setIsDragOver(true);
  };

  const handleDragLeave = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragOver(false);
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragOver(false);

    try {
      const data = JSON.parse(e.dataTransfer.getData('application/json'));
      const { cardId, listId: sourceListId } = data;
      
      if (sourceListId !== list.id) {
        onCardDrop(cardId, sourceListId, list.id);
      }
    } catch (err) {
      console.error('Failed to parse drag data:', err);
    }
  };

  return (
    <div
      className={`flex-shrink-0 w-80 bg-gray-100 rounded-lg p-4 transition-colors ${
        isDragOver ? 'bg-blue-50 ring-2 ring-blue-400' : ''
      }`}
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
    >
      <div className="flex items-center justify-between mb-4">
        <h2 className="font-semibold text-gray-900 flex items-center gap-2">
          {list.name}
          {isDone && (
            <span className="text-xs bg-green-100 text-green-700 px-2 py-1 rounded">
              Done
            </span>
          )}
          <span className="text-sm text-gray-500 font-normal">
            ({list.cards.length})
          </span>
        </h2>
      </div>

      <div className="space-y-2 min-h-[100px]">
        {list.cards.map((card) => (
          <Card
            key={card.id}
            card={card}
            onCardClick={onCardClick}
            isDragging={draggingCard?.id === card.id}
            onDragStart={onDragStart}
            onDragEnd={onDragEnd}
          />
        ))}
        {isDragOver && list.cards.length === 0 && (
          <div className="text-center text-gray-400 py-8 border-2 border-dashed border-blue-300 rounded-lg">
            Drop card here
          </div>
        )}
      </div>
    </div>
  );
}
