export interface BoardSummary {
  id: number;
  name: string;
  status: string;
  archivedAt: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface CardItem {
  id: number;
  listId: number;
  title: string;
  description: string;
  position: number;
  createdAt: string;
  updatedAt: string;
}

export interface ListColumn {
  id: number;
  name: string;
  stageOrder: number;
  position: number;
  isArchived: boolean;
  createdAt: string;
  updatedAt: string;
  cards: CardItem[];
}

export interface BoardTreePayload {
  id: number;
  name: string;
  status: string;
  archived_at: string | null;
  created_at: string;
  updated_at: string;
  lists: Array<{
    id: number;
    board_id: number;
    name: string;
    stage_order: number;
    position: number;
    is_archived: number;
    created_at: string;
    updated_at: string;
    cards: Array<{
      id: number;
      list_id: number;
      title: string;
      description: string;
      position: number;
      created_at: string;
      updated_at: string;
    }> | null;
  }> | null;
}

export interface BoardTree {
  board: BoardSummary;
  lists: ListColumn[];
}

export function normalizeBoardTree(payload: BoardTreePayload): BoardTree {
  const board: BoardSummary = {
    id: payload.id,
    name: payload.name,
    status: payload.status,
    archivedAt: payload.archived_at,
    createdAt: payload.created_at,
    updatedAt: payload.updated_at,
  };

  const lists: ListColumn[] = (payload.lists ?? [])
    .map((list) => ({
      id: list.id,
      name: list.name,
      stageOrder: list.stage_order,
      position: list.position,
      isArchived: Boolean(list.is_archived),
      createdAt: list.created_at,
      updatedAt: list.updated_at,
      cards: [...(list.cards ?? [])]
        .sort((a, b) => a.position - b.position)
        .map<CardItem>((card) => ({
          id: card.id,
          listId: card.list_id,
          title: card.title,
          description: card.description,
          position: card.position,
          createdAt: card.created_at,
          updatedAt: card.updated_at,
        })),
    }))
    .sort((a, b) => {
      if (a.stageOrder !== b.stageOrder) {
        return a.stageOrder - b.stageOrder;
      }

      return a.position - b.position;
    });

  return { board, lists };
}
