import { inject, type ComputedRef, type InjectionKey, type Ref } from 'vue'
import type { BotAccount } from '@rayleabot/plugin-ui'

export interface Operation { name: string; label: string; command: string; input: string }
export interface Game { id: string; name: string; prefix: string; region: string; operations: Operation[] }
export interface Role { ref: string; uid: string; nickname: string; region: string; level: number }
export interface Account { cloud_configured?: string[]; ref: string; account_label: string; roles: Role[]; status: string; owner: { actor_id: string } }
export interface RoleOption { key: string; label: string; role: Role; account: Account }
// CatalogSummary is one bundled catalog entry without its details.
export interface CatalogSummary { id: string; name: string; kind: string; rarity: number; element?: string; weapon?: string; abbr?: string; aliases?: string[] }
// RoleCharacter is one character the selected role owns, from the official overview.
export interface RoleCharacter { id: string; name: string; level: number; rank: number; element?: string }
export type LoadState = 'idle' | 'loading' | 'ready' | 'failed'
export type Invoke = <T>(action: string, payload?: Record<string, unknown>) => Promise<T>

// ark-plugin's settings, under its names and with its defaults.
export interface ArkSettings { panel_rank: boolean; query_type: number; rank_type: number; local_panel_rank: boolean; mark_rank_type: boolean; profile_change_diff: boolean; deal_long_dmg_title: number; profile_change_ocr: boolean; export_panel_data: number; import_panel_data: number; export_panel_require: number; group_rank: boolean; local_group_rank: boolean; new_user_panel: boolean }
export const arkDefaults: ArkSettings = { panel_rank: true, query_type: 3, rank_type: 2, local_panel_rank: true, mark_rank_type: false, profile_change_diff: true, deal_long_dmg_title: 1, profile_change_ocr: true, export_panel_data: 1, import_panel_data: 2, export_panel_require: 1, group_rank: true, local_group_rank: true, new_user_panel: false }
export interface Settings { account_provider: string; image_replies: boolean; alias_permission: number; custom_aliases: Record<string, string>; ark: ArkSettings }
export interface AliasConflict { name: string; target: string; builtin: string[] }

// Shell is what every workspace and section shares: the plugin's descriptor,
// the operator's bound accounts with one selected role, and the settings.
export interface Shell {
  page: string
  invoke: Invoke
  game: Ref<Game | null>
  bot: ComputedRef<BotAccount | null>
  catalogVersion: Ref<string>
  settings: Ref<Settings>
  // saveSettings validates aliases, writes the merged settings and returns
  // the alias conflicts the plugin reported.
  saveSettings(patch: Partial<Settings>): Promise<AliasConflict[]>
  accounts: Ref<Account[]>
  accountsLoaded: Ref<boolean>
  accountIssue: Ref<string>
  nextAccountPage: Ref<number | null>
  loadAccounts(page?: number): Promise<void>
  roles: ComputedRef<RoleOption[]>
  selection: Ref<string>
  selectedRole: ComputedRef<RoleOption | undefined>
  chosen(): { account_ref: string; role_ref: string }
  // The bundled catalog's entries, read once for every list that offers a
  // character or light cone to pick.
  catalog: Ref<CatalogSummary[]>
  catalogState: Ref<LoadState>
  loadCatalog(): Promise<void>
  // The selected role's own characters from the official overview, cached
  // per role; lists that act on the role's characters pick from these.
  characters: Ref<RoleCharacter[]>
  charactersState: Ref<LoadState>
  charactersIssue: Ref<string>
  loadCharacters(force?: boolean): Promise<void>
}

export const shellKey: InjectionKey<Shell> = Symbol('shell')

export function useShell(): Shell {
  const shell = inject(shellKey)
  if (!shell) throw new Error('useShell() outside the plugin shell')
  return shell
}

export const regions = [
  { id: 'prod_gf_cn', name: '国服' },
  { id: 'prod_qd_cn', name: '国服渠道服' },
  { id: 'prod_official_usa', name: '美服' },
  { id: 'prod_official_eur', name: '欧服' },
  { id: 'prod_official_asia', name: '亚服' },
  { id: 'prod_official_cht', name: '港澳台服' },
]
export function regionName(id: string): string { return regions.find(region => region.id === id)?.name ?? id }
export function roleKey(accountRef: string, roleRef: string): string { return JSON.stringify([accountRef, roleRef]) }
export function overseas(region: string): boolean { return region.startsWith('prod_official_') }
