import { useState, useEffect } from 'react';
import type { CardItem, ListColumn } from '../api/types';
import * as api from '../api/kanban';

interface CardDetailDrawerProps {
  card: CardItem;
  lists: ListColumn[];
  onClose: () => void;
  onUpdate: (cardId: number, title: string, description: string) => Promise<void>;
  onMove: (cardId: number, targetListId: number, targetIndex: number) => Promise<void>;
}

export function CardDetailDrawer({
  card,
  lists,
  onClose,
  onUpdate,
  onMove,
}: CardDetailDrawerProps) {
  const [title, setTitle] = useState(card.title);
  const [description, setDescription] = useState(card.description);
  const [isEditing, setIsEditing] = useState(false);
  const [comments, setComments] = useState<api.Comment[]>([]);
  const [newComment, setNewComment] = useState('');
  const [loadingComments, setLoadingComments] = useState(false);

  const currentList = lists.find((l) => l.id === card.listId);

  useEffect(() => {
    setTitle(card.title);
    setDescription(card.description);
  }, [card]);

  useEffect(() => {
    const loadComments = async () => {
      if (!card.id) {
        console.error('Card ID is missing:', card);
        return;
      }
      
      setLoadingComments(true);
      try {
        const data = await api.fetchCardComments(card.id);
        setComments(data);
      } catch (err) {
        console.error('Failed to load comments:', err);
      } finally {
        setLoadingComments(false);
      }
    };

    void loadComments();
  }, [card.id]);

  const handleSave = async () => {
    try {
      await onUpdate(card.id, title, description);
      setIsEditing(false);
    } catch (err) {
      console.error('Failed to update card:', err);
    }
  };

  const handleAddComment = async () => {
    if (!newComment.trim()) return;

    try {
      const comment = await api.createComment(card.id, newComment.trim());
      setComments([...comments, comment]);
      setNewComment('');
    } catch (err) {
      console.error('Failed to add comment:', err);
    }
  };

  const handleMoveToList = async (targetListId: number) => {
    if (targetListId === card.listId) return;

    try {
      await onMove(card.id, targetListId, 0);
      onClose();
    } catch (err) {
      console.error('Failed to move card:', err);
    }
  };

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black bg-opacity-50 z-40"
        onClick={onClose}
        role="presentation"
      />

      {/* Drawer */}
      <div className="fixed right-0 top-0 h-full w-full max-w-2xl bg-white shadow-xl z-50 overflow-y-auto">
        <div className="p-6">
          {/* Header */}
          <div className="flex items-start justify-between mb-6">
            <div className="flex-1">
              {isEditing ? (
                <input
                  type="text"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  className="w-full text-2xl font-bold border-b-2 border-blue-500 focus:outline-none mb-2"
                />
              ) : (
                <h2 className="text-2xl font-bold text-gray-900 mb-2">{card.title}</h2>
              )}
              <p className="text-sm text-gray-500">
                in list <span className="font-medium">{currentList?.name}</span>
              </p>
            </div>
            <button
              onClick={onClose}
              className="text-gray-400 hover:text-gray-600 text-2xl leading-none"
            >
              ×
            </button>
          </div>

          {/* Move to list */}
          <div className="mb-6">
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Move to List
            </label>
            <select
              value={card.listId}
              onChange={(e) => handleMoveToList(Number(e.target.value))}
              className="w-full border border-gray-300 rounded-md px-3 py-2 focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            >
              {lists.map((list) => (
                <option key={list.id} value={list.id}>
                  {list.name}
                </option>
              ))}
            </select>
          </div>

          {/* Description */}
          <div className="mb-6">
            <h3 className="text-lg font-semibold text-gray-900 mb-2">Description</h3>
            {isEditing ? (
              <textarea
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                className="w-full border border-gray-300 rounded-md p-3 min-h-[120px] focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              />
            ) : (
              <div
                onClick={() => setIsEditing(true)}
                className="border border-gray-300 rounded-md p-3 min-h-[120px] cursor-pointer hover:bg-gray-50"
                role="button"
                tabIndex={0}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') setIsEditing(true);
                }}
              >
                {card.description || (
                  <span className="text-gray-400">Add a description...</span>
                )}
              </div>
            )}
            {isEditing && (
              <div className="flex gap-2 mt-2">
                <button
                  onClick={handleSave}
                  className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
                >
                  Save
                </button>
                <button
                  onClick={() => {
                    setTitle(card.title);
                    setDescription(card.description);
                    setIsEditing(false);
                  }}
                  className="px-4 py-2 bg-gray-200 text-gray-700 rounded-md hover:bg-gray-300"
                >
                  Cancel
                </button>
              </div>
            )}
          </div>

          {/* Comments */}
          <div>
            <h3 className="text-lg font-semibold text-gray-900 mb-3">Comments</h3>

            {loadingComments ? (
              <p className="text-gray-500">Loading comments...</p>
            ) : (
              <div className="space-y-3 mb-4">
                {comments.length === 0 ? (
                  <p className="text-gray-500 text-sm">No comments yet.</p>
                ) : (
                  comments.map((comment) => (
                    <div key={comment.id} className="bg-gray-50 rounded-lg p-3">
                      <p className="text-sm text-gray-800">{comment.text}</p>
                      <p className="text-xs text-gray-500 mt-1">
                        {new Date(comment.createdAt).toLocaleString()}
                      </p>
                    </div>
                  ))
                )}
              </div>
            )}

            <div className="flex gap-2">
              <input
                type="text"
                value={newComment}
                onChange={(e) => setNewComment(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && !e.shiftKey) {
                    e.preventDefault();
                    void handleAddComment();
                  }
                }}
                placeholder="Add a comment..."
                className="flex-1 border border-gray-300 rounded-md px-3 py-2 focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              />
              <button
                onClick={handleAddComment}
                disabled={!newComment.trim()}
                className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:bg-gray-300 disabled:cursor-not-allowed"
              >
                Post
              </button>
            </div>
          </div>
        </div>
      </div>
    </>
  );
}
