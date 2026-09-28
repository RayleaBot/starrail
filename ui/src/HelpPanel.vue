<script setup lang="ts">
import { ref } from 'vue'
import StatusLine from './shell/StatusLine.vue'
import EmptyState from './shell/EmptyState.vue'
import { useShell } from './shell/context'
import { useOperation } from './shell/status'

// 命令与版本: the plugin's identity and its commands, searchable.
interface Command { id: string; name: string; description: string; usage: string; permission: string; trigger: { names?: string[] | null } }
interface Info { plugin: { name: string; version: string; license: string; min_core_version: string }; commands: Command[]; catalog_version: string; resource_version: string }
const shell = useShell()
const { busy, error, run } = useOperation('帮助读取未完成，请重试。')
const query = ref(''), info = ref<Info | null>(null)
async function load() { await run(async () => { info.value = await shell.invoke<Info>('help.query', { query: query.value.trim() }) }) }
void load()
</script>

<template>
  <section class="section" aria-labelledby="help-title">
    <div class="section-heading"><div class="titles"><h2 id="help-title">命令与版本</h2><p>命令示例中的前缀按机器人配置替换。插件更新在宿主的插件安装入口进行；本地资料随插件版本更新。</p></div></div>
    <dl v-if="info" class="facts">
      <div><dt>插件</dt><dd>{{ info.plugin.name }} {{ info.plugin.version }}</dd></div>
      <div><dt>许可证</dt><dd>{{ info.plugin.license }}</dd></div>
      <div><dt>最低宿主版本</dt><dd>{{ info.plugin.min_core_version }}</dd></div>
      <div><dt>图鉴资料</dt><dd>{{ info.catalog_version }}</dd></div>
      <div><dt>材料与卡池</dt><dd>{{ info.resource_version }}</dd></div>
    </dl>
    <form class="field-row" @submit.prevent="load">
      <label class="field"><span>筛选命令</span><input v-model="query" :disabled="busy" placeholder="功能名或关键词"></label>
      <button :disabled="busy">查找</button>
    </form>
    <StatusLine kind="error" :text="error" />
    <EmptyState v-if="info && !info.commands.length" icon="search" title="没有匹配的命令" text="换一个关键词，或清空后查看全部命令。" />
    <ul v-else-if="info" class="rows">
      <li v-for="c in info.commands" :key="c.id">
        <div class="row-main"><strong class="mono">{{ c.usage || c.name }}</strong><div class="row-meta"><span>{{ c.description }}</span><span v-if="(c.trigger.names?.length ?? 0) > 1">入口：{{ c.trigger.names!.join('、') }}</span></div></div>
      </li>
    </ul>
  </section>
</template>
