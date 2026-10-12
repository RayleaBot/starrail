import type { BotAccount } from '@rayleabot/plugin-ui'

export function sectionStorageKey(page: string, bot: Pick<BotAccount, 'source_protocol' | 'bot_id'> | null): string {
  return `raylea.starrail.section:${JSON.stringify([page, bot?.source_protocol ?? '', bot?.bot_id ?? ''])}`
}
