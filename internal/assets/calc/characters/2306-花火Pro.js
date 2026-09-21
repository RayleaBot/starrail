const characterRule=(()=>{const exports={};
"use strict";
// Authored from the pinned Sparkle enhancement skill/trace descriptions.
// The combat scenario has four talent stacks and Cipher; skill buffs target
// a teammate, so they are reported as effects rather than added to Sparkle.
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = exports.details = exports.createdBy = void 0;
exports.createdBy = 'RayleaBot';
exports.details = [{
        title: '普攻（四层天赋＋谜诡）', dmg: ({ talent }, dmg) => dmg(talent.a['技能伤害'], 'a')
    }, {
        title: '战技给予队友的暴伤加成', dmg: ({ talent, attr, cons, calc }) => ({ type: 'text', avg: Format.percent((talent.e['百分比爆伤'] + (cons >= 6 ? 0.3 : 0)) * calc(attr.cdmg) / 100 + talent.e['额外爆伤']) })
    }, {
        title: '四层天赋＋谜诡：敌方易伤', dmg: ({ talent }) => ({ type: 'text', avg: Format.percent(4 * (talent.t['伤害提高'] + talent.q['天赋增伤每层额外提高'])) })
    }, {
        title: '二魂四层天赋：敌方防御降低', cons: 2, dmg: () => ({ type: 'text', avg: '32.0%' })
    }, {
        title: '计算条件', dmg: () => ({ type: 'text', avg: '天赋四层、谜诡生效；战技目标为队友，自身不计战技暴伤与夜想曲抗穿。' })
    }];
exports.buffs = [{
        title: '四层天赋＋谜诡：敌方受到伤害提高[enemydmg]%', data: { enemydmg: ({ talent }) => 400 * (talent.t['伤害提高'] + talent.q['天赋增伤每层额外提高']) }
    }, { title: '行迹-夜想曲：全队攻击力提高[atkPct]%', tree: 3, data: { atkPct: 45 } },
    { title: '一魂：谜诡攻击力提高[atkPct]%，战技后自身速度提高[speedPct]%', cons: 1, data: { atkPct: 40, speedPct: 15 } },
    { title: '二魂：四层天赋降低敌方防御[enemyDef]%', cons: 2, data: { enemyDef: 32 } }];

return exports;})();
