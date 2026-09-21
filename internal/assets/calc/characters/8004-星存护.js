const characterRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.createdBy = exports.buffs = exports.mainAttr = exports.defDmgIdx = exports.details = void 0;
exports.details = [{
        title: '普攻伤害',
        dmg: ({ talent, cons, attr }, { basic }) => {
            let atkTd = talent.a['技能伤害'];
            let defTd = cons >= 1 ? 0.25 : 0;
            return basic(atkTd * attr.atk + defTd * attr.def, 'a');
        }
    }, {
        title: '强化普攻扩散伤害',
        dmg: ({ talent, cons, attr }, { basic }) => {
            let atkTd = talent.a2['技能伤害'] + talent.a2['相邻目标伤害'] * 2;
            let defTd = cons >= 1 ? 0.5 * 3 : 0;
            return basic(atkTd * attr.atk + defTd * attr.def, 'a');
        }
    }, {
        title: '终结技伤害',
        dmg: ({ talent, attr }, { basic }) => {
            let atkTd = talent.q['攻击力倍率'];
            let defTd = talent.q['防御力倍率'];
            return basic(atkTd * attr.atk + defTd * attr.def, 'q');
        }
    }, {
        title: '天赋护盾量',
        dmg: ({ talent, attr, cons }, { shield }) => {
            let extraDefTd = cons >= 2 ? 0.02 : 0;
            let extraConstTd = cons >= 2 ? 27 : 0;
            let defTd = extraDefTd + talent.t['百分比防御'];
            let constTd = extraConstTd + talent.t['固定值'];
            return shield(defTd * attr.def + constTd);
        }
    }];
exports.defDmgIdx = 3;
exports.mainAttr = 'atk,cpct,cdmg,def';
exports.buffs = [{
        title: '开拓者6命：提高防御力[defPct]%',
        cons: 6,
        data: {
            defPct: 30
        }
    }];
exports.createdBy = 'Aluxes';

return exports;})();
