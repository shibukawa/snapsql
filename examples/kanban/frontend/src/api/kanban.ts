import { request } from './client';
import type { BoardTree, BoardTreePayload, BoardSummary, CardItem } from './types';
import { normalizeBoardTree } from './types';

interface BoardSummaryPayload {
  id: number;
  name: string;
  status: string;
  archived_at: string | null;
  created_at: string;
  updated_at: string;
}

interface CardPayload {
  id: number;
  list_id: number;
  title: string;
  description: string;
  position: number;
  created_at: string;
  updated_at: string;
}

const API_PREFIX = '/api';

export type CreateCardPayload = {
  title: string;
  description?: string;
  position?: number;
};

export type UpdateCardPayload = {
  title: string;
  description?: string;
};

export type MoveCardPayload = {
  cardId: number;
  targetListId: number;
  targetPosition: number;
};

function toBoardSummary(row: BoardSummaryPayload): BoardSummary {
  return {
    id: row.id,
    name: row.name,
    status: row.status,
    archivedAt: row.archived_at,
    createdAt: row.created_at,
    updatedAt: row.updated_at,
  };
}

function toCardItem(row: CardPayload): CardItem {
  return {
    id: row.id,
    listId: row.list_id,
    title: row.title,
    description: row.description,
    position: row.position,
    createdAt: row.created_at,
    updatedAt: row.updated_at,
  };
}

export async function fetchBoards(): Promise<BoardSummary[]> {
  const data = await request<BoardSummaryPayload[]>(`${API_PREFIX}/boards`, { method: 'GET' });

  return data.map(toBoardSummary);
}

export async function fetchBoardTree(boardId: number): Promise<BoardTree> {
  const payload = await request<BoardTreePayload>(`${API_PREFIX}/boards/${boardId}/tree`, {
    method: 'GET',
  });

  return normalizeBoardTree(payload);
}

export async function createCard(payload: CreateCardPayload): Promise<CardItem> {
  const body = JSON.stringify({
    title: payload.title,
    description: payload.description ?? '',
    position: payload.position ?? 1,
  });

  const response = await request<CardPayload>(`${API_PREFIX}/cards`, {
    method: 'POST',
    body,
  });

  return toCardItem(response);
}

export async function updateCard(cardId: number, payload: UpdateCardPayload): Promise<void> {
  const body = JSON.stringify({
    title: payload.title,
    description: payload.description ?? '',
  });

  await request<void>(`${API_PREFIX}/cards/${cardId}`, {
    method: 'PATCH',
    body,
  });
}

export async function moveCard(payload: MoveCardPayload): Promise<void> {
  const body = JSON.stringify({
    target_list_id: payload.targetListId,
    target_position: payload.targetPosition,
  });

  await request<void>(`${API_PREFIX}/cards/${payload.cardId}/move`, {
    method: 'POST',
    body,
  });
}

export interface Comment {
  id: number;
  cardId: number;
  text: string;
  createdAt: string;
}

interface CommentPayload {
  id: number;
  card_id: number;
  text: string;
  created_at: string;
}

function toComment(row: CommentPayload): Comment {
  return {
    id: row.id,
    cardId: row.card_id,
    text: row.text,
    createdAt: row.created_at,
  };
}

export async function fetchCardComments(cardId: number): Promise<Comment[]> {
  const data = await request<CommentPayload[]>(`${API_PREFIX}/cards/${cardId}/comments`, {
    method: 'GET',
  });

  return data.map(toComment);
}

export async function createComment(cardId: number, text: string): Promise<Comment> {
  const body = JSON.stringify({ text });

  const response = await request<CommentPayload>(`${API_PREFIX}/cards/${cardId}/comments`, {
    method: 'POST',
    body,
  });

  return toComment(response);
}

export async function createBoard(name: string): Promise<BoardSummary> {
  const body = JSON.stringify({ name });

  const response = await request<BoardSummaryPayload>(`${API_PREFIX}/boards`, {
    method: 'POST',
    body,
  });

  return toBoardSummary(response);
}
