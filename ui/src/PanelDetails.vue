<script setup lang="ts">
interface Skill { name: string; level?: number; active: boolean; description?: string }
export interface Panel { id: string; name: string; skills?: Skill[]; ranks?: Skill[] }
// PanelDetails lists a character's skills and eidolons from a query's panel.
defineProps<{ panel: Panel }>()
</script>
<template>
  <section v-if="panel.skills?.length || panel.ranks?.length" class="panel">
    <h2>{{ panel.name }} · 技能与星魂</h2>
    <ul class="rows">
      <li v-for="(skill, index) in panel.skills" :key="`skill-${index}`">
        <div class="row-main"><strong>{{ skill.name }}</strong><p v-if="skill.description" class="hint prose">{{ skill.description }}</p></div>
        <span class="chip" :class="skill.active ? 'success' : ''">{{ skill.active ? `等级 ${skill.level ?? 0}` : '未解锁' }}</span>
      </li>
      <li v-for="(rank, index) in panel.ranks" :key="`rank-${index}`">
        <div class="row-main"><strong>{{ rank.name }}</strong><p v-if="rank.description" class="hint prose">{{ rank.description }}</p></div>
        <span class="chip" :class="rank.active ? 'success' : ''">{{ rank.active ? '已解锁' : '未解锁' }}</span>
      </li>
    </ul>
  </section>
</template>
