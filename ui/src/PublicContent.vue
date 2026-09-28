<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import StatusLine from './shell/StatusLine.vue'
import EmptyState from './shell/EmptyState.vue'
import { useShell } from './shell/context'
import { formatTime } from './shell/status'
import { cloudTask } from './cloud'

// 公告与活动: the official public content, read anonymously as a content
// task. The groups' news pushes live in the groups workspace.
interface Post { id: string; title: string; url: string; created: string; author?: string; parts?: string[]; images?: string[] }
interface Activity { id: string; title: string; text: string; start?: string; end?: string; time_status: string }
const shell = useShell()
const task = cloudTask(shell.invoke, 'content')
const kind = ref('news'), type = ref(1), query = ref(''), postID = ref(''), part = ref(0), selected = ref(''), date = ref('')
const result = computed(() => task.job.value?.result)
const posts = computed(() => (result.value?.items ?? []) as Post[])
const post = computed(() => result.value?.post as Post | undefined)
const activities = computed(() => ((result.value?.items ?? []) as Activity[]).filter(item => !date.value || item.start && item.end && item.start.slice(0, 10) <= date.value && item.end.slice(0, 10) >= date.value))
const detail = computed(() => activities.value.find(item => item.id === selected.value))
const running = computed(() => task.job.value?.state === 'running')
watch([kind, type, query, postID], () => { task.clear(); selected.value = ''; part.value = 0 })
async function start(next = false) {
  selected.value = ''; part.value = 0
  const previous = result.value
  await task.start({ kind: kind.value, type: type.value, query: query.value.trim(), post_id: postID.value.trim(), last_id: next ? previous?.next_id ?? '' : '', offset: next ? previous?.next_offset ?? 0 : 0 })
}
onUnmounted(task.dispose)
</script>

<template>
  <section class="section" aria-labelledby="content-title">
    <div class="section-heading"><div class="titles"><h2 id="content-title">公告与活动</h2><p>匿名读取官方公开内容，不需要账号。活动时间未确定时保留公告原文；时间按国服显示。各群的公告推送在“群管理”的公告推送分区管理。</p></div></div>
    <form class="field-row" @submit.prevent="start()">
      <fieldset :disabled="task.busy.value">
        <label class="field"><span>查询内容</span><select v-model="kind"><option value="news">公告、资讯与活动帖子</option><option value="calendar">活动日历与游戏公告</option><option value="search">米游社搜索</option><option value="estimate">版本资源预估与盘点</option><option value="post">文章详情与原图链接</option></select></label>
        <label v-if="kind === 'news'" class="field"><span>帖子类型</span><select v-model="type"><option :value="1">公告</option><option :value="2">活动</option><option :value="3">资讯</option></select></label>
        <label v-if="kind === 'search'" class="field"><span>搜索关键词</span><input v-model="query" maxlength="100" required></label>
        <label v-if="kind === 'post'" class="field"><span>文章编号或官方链接</span><input v-model="postID" required></label>
        <button class="primary" type="submit" :disabled="running">{{ running ? '正在读取…' : '查询公开内容' }}</button>
      </fieldset>
    </form>
    <StatusLine kind="error" :text="task.error.value || task.job.value?.message || ''" />
    <StatusLine v-if="running" kind="info" text="正在读取官方内容…"><template #actions><button type="button" class="compact" :disabled="task.busy.value" @click="task.cancel">取消</button><button type="button" class="compact" @click="task.poll">刷新进度</button></template></StatusLine>
    <template v-if="task.job.value?.state === 'completed'">
      <p v-if="result?.note" class="hint">{{ result.note }}</p>
      <p class="hint">获取时间：{{ formatTime(Number(result?.fetched_at_ms)) }}</p>
      <template v-if="kind === 'news' || kind === 'search' || kind === 'estimate'">
        <EmptyState v-if="!posts.length" icon="megaphone" title="没有匹配帖子" text="换一个类型或关键词再查询。" />
        <ul v-else class="rows">
          <li v-for="item in posts" :key="item.id">
            <div class="row-main"><strong>{{ item.title }}</strong><div class="row-meta"><span>{{ item.created }}</span><span v-if="item.author">{{ item.author }}</span></div></div>
            <div class="row-actions"><button type="button" class="compact" @click="kind = 'post'; postID = item.id">填入文章详情</button><a :href="item.url" target="_blank" rel="noreferrer">官方原文</a></div>
          </li>
        </ul>
        <div v-if="result?.more" class="actions"><button type="button" :disabled="task.busy.value" @click="start(true)">下一页帖子</button></div>
      </template>
      <article v-if="kind === 'post' && post" class="panel">
        <header><h3>{{ post.title }}</h3><p class="muted">{{ post.author }} {{ post.created }} · <a :href="post.url" target="_blank" rel="noreferrer">官方原文</a></p></header>
        <div class="actions"><button type="button" class="compact" :disabled="part <= 0" @click="part--">上一段</button><span class="muted">{{ part + 1 }} / {{ Math.max(1, post.parts?.length ?? 0) }}</span><button type="button" class="compact" :disabled="part + 1 >= (post.parts?.length ?? 0)" @click="part++">下一段</button></div>
        <p class="prose">{{ post.parts?.[part] || '本文主要内容为图片或视频，请查看官方原文。' }}</p>
        <div v-if="post.images?.length" class="actions"><a v-for="(url, index) in post.images" :key="url" :href="url" target="_blank" rel="noreferrer">查看官方原图 {{ index + 1 }}</a></div>
      </article>
      <template v-if="kind === 'calendar'">
        <div class="field-row"><label class="field"><span>查看覆盖日期</span><input v-model="date" type="date"><span class="hint">留空时也列出时间未确定的活动。</span></label></div>
        <ul v-if="activities.length" class="entry-list">
          <li v-for="item in activities" :key="item.id"><button type="button" :aria-current="item.id === selected ? 'true' : undefined" @click="selected = item.id"><strong>{{ item.title }}</strong><span>{{ item.end ? `${item.start || '开始时间未确定'} 至 ${item.end}` : '活动时间未确定 · 可查看公告' }}</span></button></li>
        </ul>
        <EmptyState v-else icon="calendar" title="此范围没有可确认活动" text="换一个日期，或留空查看全部。" />
        <article v-if="detail" class="panel"><header><h3>{{ detail.title }}</h3></header><p class="prose">{{ detail.text }}</p></article>
      </template>
    </template>
    <EmptyState v-else-if="!task.job.value && !task.busy.value" icon="megaphone" title="选择内容类型后查询" text="公告、活动日历、搜索结果与文章正文会显示在这里。" />
  </section>
</template>
