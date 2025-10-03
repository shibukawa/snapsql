import type { Mock } from 'vitest'
import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('../../../src/api/client', () => ({
  request: vi.fn(),
}))

import { request } from '../../../src/api/client'
import { createCard, fetchBoardTree, fetchBoards, moveCard, updateCard } from '../../../src/api/kanban'

const requestMock = request as unknown as Mock

describe('kanban API wrappers', () => {
  beforeEach(() => {
    requestMock.mockReset()
  })

  it('fetchBoards passes through data', async () => {
    const boards = [
      {
        id: 1,
        name: 'Sprint 42',
        status: 'active',
        archived_at: null,
        created_at: '2025-09-30T00:00:00Z',
        updated_at: '2025-09-30T00:00:00Z',
      },
    ]
    requestMock.mockResolvedValueOnce(boards)

    const result = await fetchBoards()

    expect(requestMock).toHaveBeenCalledWith('/api/boards', { method: 'GET' })
    expect(result[0].name).toBe('Sprint 42')
    expect(result[0].archivedAt).toBeNull()
  })

  it('fetchBoardTree normalises the payload', async () => {
    requestMock.mockResolvedValueOnce({
      id: 1,
      name: 'Sprint 42',
      status: 'active',
      archived_at: null,
      created_at: '2025-09-30T00:00:00Z',
      updated_at: '2025-09-30T00:00:00Z',
      lists: [
        {
          id: 3,
          board_id: 1,
          name: 'Done',
          stage_order: 4,
          position: 4,
          is_archived: 0,
          created_at: '2025-09-30T00:00:00Z',
          updated_at: '2025-09-30T00:00:00Z',
          cards: [],
        },
      ],
    })

    const tree = await fetchBoardTree(1)

    expect(requestMock).toHaveBeenCalledWith('/api/boards/1/tree', { method: 'GET' })
    expect(tree.board.id).toBe(1)
    expect(tree.lists[0].name).toBe('Done')
  })

  it('createCard posts to the new endpoint', async () => {
    requestMock.mockResolvedValueOnce({
      id: 99,
      list_id: 7,
      title: 'New',
      description: '',
      position: 1,
      created_at: '2025-10-01T00:00:00Z',
      updated_at: '2025-10-01T00:00:00Z',
    })

    const card = await createCard({ title: 'New', description: '' })

    expect(requestMock).toHaveBeenCalledWith('/api/cards', {
      method: 'POST',
      body: JSON.stringify({ title: 'New', description: '', position: 1 }),
    })
    expect(card.id).toBe(99)
    expect(card.listId).toBe(7)
  })

  it('updateCard calls PATCH endpoint', async () => {
    requestMock.mockResolvedValueOnce(undefined)

    await updateCard(5, { title: 'Fix', description: '' })

    expect(requestMock).toHaveBeenCalledWith('/api/cards/5', {
      method: 'PATCH',
      body: JSON.stringify({ title: 'Fix', description: '' }),
    })
  })

  it('moveCard sends move payload', async () => {
    requestMock.mockResolvedValueOnce(undefined)

    await moveCard({ cardId: 5, targetListId: 7, targetPosition: 3 })

    expect(requestMock).toHaveBeenCalledWith('/api/cards/5/move', {
      method: 'POST',
      body: JSON.stringify({ target_list_id: 7, target_position: 3 }),
    })
  })
})
