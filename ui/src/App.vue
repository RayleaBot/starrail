<script setup lang="ts">
import { computed, provide, ref, watch } from 'vue'
import { PluginUIClient, usePluginHost } from '@rayleabot/plugin-ui'
import { arkDefaults, roleKey, shellKey, type Account, type AliasConflict, type CatalogSummary, type Game, type LoadState, type RoleCharacter, type RoleOption, type Settings, type Shell } from './shell/context'
import { useOperation } from './shell/status'
import StatusLine from './shell/StatusLine.vue'
import QueryWorkspace from './workspaces/QueryWorkspace.vue'
import CatalogWorkspace from './workspaces/CatalogWorkspace.vue'
import GachaWorkspace from './workspaces/GachaWorkspace.vue'
import TasksWorkspace from './workspaces/TasksWorkspace.vue'
import AccountWorkspace from './workspaces/AccountWorkspace.vue'
import GroupsWorkspace from './workspaces/GroupsWorkspace.vue'
import CloudWorkspace from './workspaces/CloudWorkspace.vue'
import SettingsWorkspace from './workspaces/SettingsWorkspace.vue'

// One workspace per host page. Pages the manifest no longer declares open the
// workspace that absorbed them, so an old link still lands somewhere useful.
const workspaces = { overview: QueryWorkspace, catalog: CatalogWorkspace, gacha: GachaWorkspace, tasks: TasksWorkspace, account: AccountWorkspace, groups: GroupsWorkspace, cloud: CloudWorkspace, settings: SettingsWorkspace } as const
type Page = keyof typeof workspaces
const absorbed: Record<string, Page> = { history: 'overview', growth: 'overview', resources: 'catalog', guides: 'catalog', content: 'catalog', media: 'catalog', simulation: 'gacha', reminders: 'tasks', signin: 'account', community: 'account', assets: 'account', help: 'settings' }
const accountPages = new Set<Page>(['overview', 'gacha', 'tasks', 'account', 'cloud'])

const host = usePluginHost()
const requested = host.client.pageId || 'overview'
const page: Page = requested in workspaces ? requested as Page : absorbed[requested] ?? 'overview'
const workspace = workspaces[page]

const game = ref<Game | null>(null)
const bots = computed(() => host.bot.value ? [{ source_protocol: host.bot.value.source_protocol, id: host.bot.value.bot_id, nickname: host.bot.value.nickname }] : [])
const catalogVersion = ref('')
const settings = ref<Settings>({ account_provider: 'raylea.mihoyo-accounts', image_replies: true, alias_permission: 0, custom_aliases: {}, ark: { ...arkDefaults } })
const accounts = ref<Account[]>([])
const accountsLoaded = ref(false)
const accountIssue = ref('')
const nextAccountPage = ref<number | null>(null)
const selection = ref('')
const roles = computed<RoleOption[]>(() => accounts.value.flatMap(account => account.roles.map(role => ({ key: roleKey(account.ref, role.ref), label: `${role.nickname} · ${role.uid} · 用户 ${account.owner.actor_id}`, role, account }))))
const selectedRole = computed(() => roles.value.find(item => item.key === selection.value))
const { busy, error, run } = useOperation('页面连接失败。')

async function invoke<T>(action: string, payload: Record<string, unknown> = {}): Promise<T> {
  return await host.client.invokeAction(action, payload) as unknown as T
}
async function loadAccounts(index = 0) {
  try {
    const result = await invoke<{ accounts: { items: Account[]; next_page: number | null } }>('accounts.list', { page: index })
    accounts.value = index === 0 ? result.accounts.items : [...accounts.value, ...result.accounts.items]
    nextAccountPage.value = result.accounts.next_page
    if (!roles.value.some(item => item.key === selection.value)) selection.value = roles.value[0]?.key ?? ''
    accountIssue.value = ''
  } catch (cause) {
    accounts.value = []; selection.value = ''; nextAccountPage.value = null
    accountIssue.value = cause instanceof Error ? cause.message : '账号服务暂不可用。'
  } finally {
    accountsLoaded.value = true
  }
}
async function saveSettings(patch: Partial<Settings>): Promise<AliasConflict[]> {
  const next: Settings = { ...settings.value, ...patch, ark: { ...settings.value.ark, ...patch.ark } }
  const checked = await invoke<{ aliases: Record<string, string>; conflicts: AliasConflict[] }>('aliases.validate', { aliases: next.custom_aliases })
  next.custom_aliases = checked.aliases
  await host.client.saveSettings({ account_provider: next.account_provider.trim(), image_replies: next.image_replies, alias_permission: next.alias_permission, custom_aliases: next.custom_aliases, ark: next.ark })
  settings.value = next
  return checked.conflicts
}
async function load() {
  const result = await invoke<{ game: Game; catalog_version: string; settings: { account_provider: string; image_replies: boolean; alias_permission?: number; custom_aliases?: Record<string, string>; ark?: Partial<Settings['ark']> } }>('status')
  game.value = result.game
  catalogVersion.value = result.catalog_version
  settings.value = { account_provider: result.settings.account_provider, image_replies: result.settings.image_replies, alias_permission: result.settings.alias_permission ?? 0, custom_aliases: result.settings.custom_aliases ?? {}, ark: { ...arkDefaults, ...result.settings.ark } }
  if (accountPages.has(page)) await loadAccounts()
}
function chosen() {
  const current = selectedRole.value
  if (!current) throw new Error('请先选择已授权的角色。')
  return { account_ref: current.account.ref, role_ref: current.role.ref }
}

// The catalog is read once; the role's characters once per role, through the
// same official overview the 角色 query shows.
const catalog = ref<CatalogSummary[]>([]), catalogState = ref<LoadState>('idle')
let catalogPending: Promise<void> | undefined
function loadCatalog(): Promise<void> {
  if (catalogState.value === 'ready') return Promise.resolve()
  if (catalogPending) return catalogPending
  catalogState.value = 'loading'
  catalogPending = invoke<{ entries: CatalogSummary[] }>('catalog.list')
    .then(result => { catalog.value = result.entries; catalogState.value = 'ready' })
    .catch(() => { catalogState.value = 'failed' })
    .finally(() => { catalogPending = undefined })
  return catalogPending
}
const characters = ref<RoleCharacter[]>([]), charactersState = ref<LoadState>('idle'), charactersIssue = ref('')
const characterCache = new Map<string, RoleCharacter[]>()
let charactersPending: { key: string; promise: Promise<void> } | undefined
async function loadCharacters(force = false): Promise<void> {
  const key = selection.value, operation = game.value?.operations.find(item => item.input === 'characters_optional')
  if (!selectedRole.value || !operation) { characters.value = []; charactersState.value = 'idle'; return }
  const cached = characterCache.get(key)
  if (cached && !force) { characters.value = cached; charactersState.value = 'ready'; return }
  if (charactersPending?.key === key) return charactersPending.promise
  characters.value = []; charactersState.value = 'loading'; charactersIssue.value = ''
  const promise = invoke<{ panels?: { id: string; name: string; level: number; rank: number; element?: string }[] }>('query', { ...chosen(), operation: operation.name, input: {} })
    .then(result => {
      const list = (result.panels ?? []).map(panel => ({ id: panel.id, name: panel.name, level: panel.level, rank: panel.rank, element: panel.element }))
      characterCache.set(key, list)
      if (selection.value === key) { characters.value = list; charactersState.value = 'ready' }
    })
    .catch(cause => { if (selection.value === key) { charactersState.value = 'failed'; charactersIssue.value = cause instanceof Error && cause.message ? cause.message : '角色列表读取失败。' } })
    .finally(() => { if (charactersPending?.key === key) charactersPending = undefined })
  charactersPending = { key, promise }
  return promise
}
// A role change shows its cached characters at once and leaves the rest to
// the lists that need them.
watch(selection, () => { const cached = characterCache.get(selection.value); characters.value = cached ?? []; charactersState.value = cached ? 'ready' : 'idle'; charactersIssue.value = '' })
const shell: Shell = { page, invoke, game, bots, bot: host.bot,
  apiRequest: (method, path) => host.client.apiRequest(method, path),
  assignHistory: (bot, payload) => new PluginUIClient({ pluginId: host.client.pluginId, pageId: 'groups', bot }).invokeAction('history.assign', payload),
  catalogVersion, settings, saveSettings, accounts, accountsLoaded, accountIssue, nextAccountPage, loadAccounts, roles, selection, selectedRole, chosen, catalog, catalogState, loadCatalog, characters, charactersState, charactersIssue, loadCharacters }
provide(shellKey, shell)
void host.ready.then(() => run(load)).catch(cause => { error.value = cause instanceof Error ? cause.message : '页面连接失败。' })
</script>

<template>
  <main class="shell" :aria-busy="busy || host.loading.value">
    <div v-if="error" class="shell-status">
      <StatusLine kind="error" :text="error"><template #actions><button type="button" :disabled="busy" @click="run(load)">重新连接</button></template></StatusLine>
    </div>
    <div v-else-if="!game" class="skeleton" role="status" aria-label="正在读取插件资料"><span /><span /><span /></div>
    <component :is="workspace" v-else />
  </main>
</template>
