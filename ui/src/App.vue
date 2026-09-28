<script setup lang="ts">
import { computed, provide, ref } from 'vue'
import { usePluginHost } from '@rayleabot/plugin-ui'
import { arkDefaults, roleKey, shellKey, type Account, type AliasConflict, type Bot, type Game, type RoleOption, type Settings, type Shell } from './shell/context'
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
const bots = ref<Bot[]>([])
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
  const result = await invoke<{ game: Game; bots?: Bot[]; catalog_version: string; settings: { account_provider: string; image_replies: boolean; alias_permission?: number; custom_aliases?: Record<string, string>; ark?: Partial<Settings['ark']> } }>('status')
  game.value = result.game
  bots.value = result.bots ?? []
  catalogVersion.value = result.catalog_version
  settings.value = { account_provider: result.settings.account_provider, image_replies: result.settings.image_replies, alias_permission: result.settings.alias_permission ?? 0, custom_aliases: result.settings.custom_aliases ?? {}, ark: { ...arkDefaults, ...result.settings.ark } }
  if (accountPages.has(page)) await loadAccounts()
}
function chosen() {
  const current = selectedRole.value
  if (!current) throw new Error('请先选择已授权的角色。')
  return { account_ref: current.account.ref, role_ref: current.role.ref }
}
const shell: Shell = { page, invoke, game, bots, catalogVersion, settings, saveSettings, accounts, accountsLoaded, accountIssue, nextAccountPage, loadAccounts, roles, selection, selectedRole, chosen }
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
    <footer class="shell-footer"><a href="./source.zip" download>本版本对应源码</a></footer>
  </main>
</template>
