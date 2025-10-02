import { describe, expect, it } from 'vitest'

import { normalizeBoardTree } from '../../../src/api/types'

describe('normalizeBoardTree', () => {
  it('converts API payload into BoardTree structure', () => {
    const payload = {
      id: 1,
      name: 'Sprint 42',
      status: 'active',
      archived_at: null,
      created_at: '2025-09-30T00:00:00Z',
      updated_at: '2025-09-30T00:00:00Z',
      lists: [
        {
          id: 10,
          board_id: 1,
          name: 'Review',
          stage_order: 3,
          position: 3,
          is_archived: 0,
          created_at: '2025-09-30T00:00:00Z',
          updated_at: '2025-09-30T00:00:00Z',
          cards: [
            {
              id: 100,
              list_id: 10,
              title: 'Check QA',
              description: 'Run regression',
              position: 2,
              created_at: '2025-09-30T00:00:00Z',
              updated_at: '2025-09-30T00:00:00Z',
            },
          ],
        },
        {
          id: 11,
          board_id: 1,
          name: 'Backlog',
          stage_order: 1,
          position: 1,
          is_archived: 0,
          created_at: '2025-09-30T00:00:00Z',
          updated_at: '2025-09-30T00:00:00Z',
          cards: [
            {
              id: 101,
              list_id: 11,
              title: 'Spec API',
              description: 'Write spec',
              position: 1,
              created_at: '2025-09-29T00:00:00Z',
              updated_at: '2025-09-29T00:00:00Z',
            },
          ],
        },
      ],
    }

    const tree = normalizeBoardTree(payload)

    expect(tree.board).toEqual({
      id: 1,
      name: 'Sprint 42',
      status: 'active',
      archivedAt: null,
      createdAt: '2025-09-30T00:00:00Z',
      updatedAt: '2025-09-30T00:00:00Z',
    })
    expect(tree.lists.map((l) => l.name)).toEqual(['Backlog', 'Review'])
    expect(tree.lists[0].cards[0].title).toBe('Spec API')
  })
})
