import type { CardItem } from '../api/types';

interface CardProps {
  card: CardItem;
  onCardClick: (card: CardItem) => void;
  isDragging?: boolean;
  onDragStart?: (card: CardItem) => void;
  onDragEnd?: () => void;
}

export function Card({ card, onCardClick, isDragging = false, onDragStart, onDragEnd }: CardProps) {
  const handleClick = () => {
    console.log('Card handleClick called:', card.id, card.title);
    onCardClick(card);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      console.log('Card handleKeyDown called:', card.id, card.title);
      onCardClick(card);
    }
  };

  const handleDragStart = (e: React.DragEvent) => {
    e.dataTransfer.effectAllowed = 'move';
    e.dataTransfer.setData('application/json', JSON.stringify({
      cardId: card.id,
      listId: card.listId,
    }));
    onDragStart?.(card);
  };

  const handleDragEnd = () => {
    onDragEnd?.();
  };

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={handleClick}
      onKeyDown={handleKeyDown}
      onDragStart={handleDragStart}
      onDragEnd={handleDragEnd}
      className={`
        bg-white rounded-lg shadow-sm p-3 mb-2 cursor-pointer
        hover:shadow-md transition-shadow
        ${isDragging ? 'opacity-50' : ''}
      `}
      draggable
    >
      <h3 className="font-medium text-gray-900 mb-1">{card.title}</h3>
      {card.description && (
        <p className="text-sm text-gray-600 line-clamp-2">{card.description}</p>
      )}
    </div>
  );
}
