import { describe, expect, it } from 'vitest'
import { historicalRecords, sectionStorageKey, type APIRequest, type HistoryRecord } from '../src/bot-history'

const row = (protocol: string, id: string): HistoryRecord => ({ kind: 'groups', ref: id, owner: { source_protocol: protocol, bot_id: id }, label: '', version: 'v', count: 1 })

describe('historical account ownership', () => {
  it('keeps offline accounts and uses detail 404 for absent accounts, including beyond the first directory page', async () => {
    const calls: string[] = []
    const request: APIRequest = async <T>(_method: string, path: string): Promise<T> => {
      calls.push(path)
      if (path.endsWith('/removed')) throw { status: 404 }
      return { state: 'no_connection' } as T
    }
    const records = await historicalRecords([row('onebot11', 'page-2'), row('onebot11', 'removed'), row('onebot11', 'removed'), row('qqofficial', ''), row('', '')], request)
    expect(records.map(item => item.state)).toEqual(['missing', 'missing', 'unassigned', 'unknown'])
    expect(calls).toEqual(['/api/bot-accounts/onebot11/page-2', '/api/bot-accounts/onebot11/removed'])
  })
  it('propagates directory failures instead of treating them as absent accounts', async () => {
    const failure = { status: 503 }
    await expect(historicalRecords([row('onebot11', 'a')], async () => { throw failure })).rejects.toBe(failure)
  })
  it('separates persisted page selection by protocol and account', () => {
    const keys = [null, { source_protocol: 'onebot11' as const, bot_id: 'a' }, { source_protocol: 'qqofficial' as const, bot_id: 'a' }, { source_protocol: 'onebot11' as const, bot_id: 'b' }].map(bot => sectionStorageKey('groups', bot))
    expect(new Set(keys).size).toBe(4)
  })
})
