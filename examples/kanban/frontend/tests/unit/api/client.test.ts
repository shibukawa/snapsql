import { beforeEach, describe, expect, it, vi } from 'vitest'

import { ApiError, request } from '../../../src/api/client'

declare global {
  // eslint-disable-next-line no-var
  var fetch: typeof fetch
}

describe('request', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('appends the API base URL and serialises query parameters', async () => {
    const responseBody = { message: 'ok' }
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(responseBody), { status: 200 }),
    )
    global.fetch = fetchMock

    const result = await request<{ message: string }>('/api/example', {
      method: 'GET',
      query: { page: 2, search: 'board', flag: true, skip: undefined },
    })

    expect(result).toEqual(responseBody)
    expect(fetchMock).toHaveBeenCalledTimes(1)

    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('http://localhost:8080/api/example?page=2&search=board&flag=true')
    expect(init?.method).toBe('GET')
    expect(init?.headers).toBeInstanceOf(Headers)
    const headers = init?.headers as Headers
    expect(headers.get('Accept')).toBe('application/json')
  })

  it('throws ApiError when the response is not ok', async () => {
    const errorBody = { error: 'invalid' }
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(errorBody), { status: 400 }),
    )
    global.fetch = fetchMock

    await expect(
      request('/api/bad', { method: 'POST', body: JSON.stringify({}) }),
    ).rejects.toBeInstanceOf(ApiError)
  })
})
