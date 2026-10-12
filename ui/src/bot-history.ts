import type { BotAccount } from '@rayleabot/plugin-ui'

export interface HistoryRecord {
  kind: string
  ref: string
  owner: { source_protocol: string; bot_id: string; actor_id?: string }
  label: string
  version: string
  count: number
}
export type HistoryEntry = HistoryRecord & { state: 'unassigned' | 'missing' | 'unknown' }
export type APIRequest = <T>(method: string, path: string) => Promise<T>

export const accountPath = (protocol: string, id: string) => `/api/bot-accounts/${encodeURIComponent(protocol)}/${encodeURIComponent(id)}`

// A filtered directory page is not evidence that an account was removed.
// Only its detail endpoint's 404 means the saved account is absent.
export async function historicalRecords(items: HistoryRecord[], request: APIRequest): Promise<HistoryEntry[]> {
  const present = new Map<string, boolean>()
  const result: HistoryEntry[] = []
  for (const item of items) {
    const { source_protocol: protocol, bot_id: id } = item.owner
    if (!protocol || !id) {
      result.push({ ...item, state: protocol ? 'unassigned' : 'unknown' })
      continue
    }
    const path = accountPath(protocol, id)
    if (!present.has(path)) {
      try { await request<BotAccount>('GET', path); present.set(path, true) }
      catch (cause) {
        if (cause && typeof cause === 'object' && 'status' in cause && cause.status === 404) present.set(path, false)
        else throw cause
      }
    }
    if (!present.get(path)) result.push({ ...item, state: 'missing' })
  }
  return result
}

export function sectionStorageKey(page: string, bot: Pick<BotAccount, 'source_protocol' | 'bot_id'> | null): string {
  return `raylea.starrail.section:${JSON.stringify([page, bot?.source_protocol ?? '', bot?.bot_id ?? ''])}`
}
