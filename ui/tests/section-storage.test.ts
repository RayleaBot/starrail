import { describe, expect, it } from 'vitest'
import { sectionStorageKey } from '../src/shell/section-storage'

describe('section memory', () => {
  it('separates persisted page selection by protocol and account', () => {
    const keys = [null, { source_protocol: 'onebot11' as const, bot_id: 'a' }, { source_protocol: 'qqofficial' as const, bot_id: 'a' }, { source_protocol: 'onebot11' as const, bot_id: 'b' }].map(bot => sectionStorageKey('groups', bot))
    expect(new Set(keys).size).toBe(4)
  })
})
