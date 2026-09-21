// Authored from the pinned enhanced Black Swan metadata. Its ultimate table
// labels are reversed: parameter 1 is direct damage, parameter 2 vulnerability.
import { Format } from '#miao'
export const createdBy='RayleaBot'
export const details=[
 {title:'普攻（战技减防已生效）',dmg:({talent},dmg)=>dmg(talent.a['技能伤害'],'a')},
 {title:'战技（减防已生效）',dmg:({talent},dmg)=>dmg(talent.e['技能伤害'],'e')},
 {title:'终结技（揭露＋战技减防）',params:{qBuff:true},dmg:({talent},dmg)=>dmg(talent.q['伤害提高'],'q')},
 ...[1,7,30,50,70].map(stacks=>({title:stacks+'层奥迹（揭露＋减防）',...(stacks===70?{cons:6}:{}),params:{qBuff:true},dmg:({talent},dmg)=>dmg(talent.t['持续伤害']+talent.t['倍率提高']*stacks,'dot','skillDot')})),
 {title:'回合开始奥迹相邻伤害',params:{qBuff:true},dmg:({talent},dmg)=>dmg(talent.t['相邻目标伤害'],'dot','skillDot')},
 {title:'揭露易伤',dmg:({talent,cons})=>({type:'text',avg:Format.percent(talent.q['技能伤害']+(cons>=4?0.2:0))})}
]
export const buffs=[
 {title:'战技：敌方防御降低[enemyDef]%',data:{enemyDef:({talent})=>talent.e['防御力降低']*100}},
 {title:'行迹-烛影朕兆：效果命中转化增伤[dmg]%',tree:3,data:{dmg:({attr})=>Math.min(attr.effPct*0.6,72)}},
 {title:'揭露：敌方易伤[enemydmg]%',check:({params})=>params.qBuff===true,data:{enemydmg:({talent,cons})=>talent.q['技能伤害']*100+(cons>=4?20:0)}},
 {title:'奥迹持续伤害无视防御[dotIgnore]%',data:{dotIgnore:20}},
 {title:'一魂：风属性抗性降低[kx]%',cons:1,data:{kx:25}}
]
