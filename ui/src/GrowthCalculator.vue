<script setup lang="ts">
import { ref, watch } from 'vue'
import BusinessView, { type View } from './BusinessView.vue'
import CharacterSelect from './shell/CharacterSelect.vue'
import StatusLine from './shell/StatusLine.vue'
import EmptyState from './shell/EmptyState.vue'
import { useShell } from './shell/context'
import { useOperation } from './shell/status'
import { useReader } from './reader'

// 养成计算: the official calculator's material needs for chosen targets. The
// chosen character's current levels are read as soon as it is picked; the
// calculation itself stays one explicit action.
interface Skill { id: number; name: string; current_level: number; target_level: number; max_level: number }
interface Plan { character_id: string; name: string; current_level: number; target_level: number; max_level: number; weapon_id: number; weapon_name: string; current_weapon_level: number; target_weapon_level: number; max_weapon_level: number; skills: Skill[] }
const shell = useShell()
const operation = useOperation('养成查询未完成，请重试。')
const { busy, error, run } = operation
const character = ref(''), plan = ref<Plan | null>(null), view = ref<View | null>(null)
const reader = useReader(operation, async stale => {
  plan.value = null; view.value = null
  if (!shell.selectedRole.value || !character.value) return
  const result = (await shell.invoke<{ plan: Plan }>('growth.prepare', { ...shell.chosen(), character_id: character.value })).plan
  if (!stale()) plan.value = result
})
// A character belongs to its role: a role change clears the choice, and the
// list of the new role picks its first character, which reads by itself.
watch(shell.selection, () => { character.value = '' })
watch(character, () => { plan.value = null; view.value = null; reader.request() })
watch(plan, () => { view.value = null }, { deep: true })
async function compute() { if (!plan.value) return; await run(async () => { view.value = (await shell.invoke<{ view: View }>('growth.compute', { ...shell.chosen(), character_id: plan.value!.character_id, plan: plan.value })).view }) }
function maximize() {
  if (!plan.value) return
  plan.value.target_level = plan.value.max_level
  if (plan.value.weapon_id) plan.value.target_weapon_level = plan.value.max_weapon_level
  for (const skill of plan.value.skills) skill.target_level = skill.max_level
}
</script>

<template>
  <section class="section" aria-labelledby="growth-title">
    <div class="section-heading"><div class="titles"><h2 id="growth-title">养成计算</h2><p>选择角色后读取它当前的等级、光锥与行迹等级，填好目标后由官方养成计算器列出所需材料；只做计算，不改动游戏。</p></div></div>
    <EmptyState v-if="!shell.selectedRole.value" icon="user" title="请先选择游戏角色" text="在右上角选择已绑定的角色后，这里会列出它拥有的角色，选择后自动读取当前等级。" />
    <template v-else>
      <form class="field-row" @submit.prevent="reader.request()">
        <fieldset :disabled="busy && !reader.loading.value">
          <CharacterSelect v-model="character" source="role" autoselect label="角色" />
          <button type="submit" :disabled="busy || !character">{{ reader.loading.value ? '正在读取…' : '重新读取' }}</button>
        </fieldset>
      </form>
      <StatusLine v-if="reader.loading.value" kind="info" text="正在读取角色当前的等级、光锥与行迹…" />
      <StatusLine kind="error" :text="error"><template v-if="reader.failed.value" #actions><button class="compact" type="button" :disabled="busy" @click="reader.request()">重试</button></template></StatusLine>
      <form v-if="plan" class="subsection" @submit.prevent="compute">
        <fieldset :disabled="busy">
          <h3>{{ plan.name }} · 目标等级</h3>
          <div class="field-grid">
            <label class="field"><span>角色等级（当前 {{ plan.current_level }}）</span><input v-model.number="plan.target_level" type="number" :min="plan.current_level" :max="plan.max_level" step="1" required></label>
            <label v-if="plan.weapon_id" class="field"><span>{{ plan.weapon_name }}等级（当前 {{ plan.current_weapon_level }}）</span><input v-model.number="plan.target_weapon_level" type="number" :min="plan.current_weapon_level" :max="plan.max_weapon_level" step="1" required></label>
            <label v-for="skill in plan.skills" :key="skill.id" class="field"><span>{{ skill.name }}（当前 {{ skill.current_level }}）</span><input v-model.number="skill.target_level" type="number" :min="skill.current_level" :max="skill.max_level" step="1" required></label>
            <div class="actions"><button type="button" @click="maximize">目标全部设为上限</button><button class="primary" type="submit">计算所需材料</button></div>
          </div>
        </fieldset>
      </form>
      <BusinessView v-if="view" :value="view" />
    </template>
  </section>
</template>
