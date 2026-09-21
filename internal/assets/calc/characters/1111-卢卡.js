const characterRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.createdBy = exports.buffs = exports.mainAttr = exports.defDmgIdx = exports.details = void 0;
exports.details = [{
        title: '普攻伤害',
        dmg: ({ talent }, dmg) => dmg(talent.a['技能伤害'], 'a')
    }, {
        title: '强化普攻伤害',
        dmg: ({ talent }, dmg) => dmg(talent.a2['直冲拳每段伤害'] * 3 + talent.a2['碎天拳伤害'], 'a')
    }, {
        title: '战技伤害',
        dmg: ({ talent }, dmg) => dmg(talent.e['单体伤害'], 'e')
    }, {
        title: '战技持续伤害',
        dmg: ({ talent }, dmg) => dmg(talent.e['持续伤害'], 'dot', 'skillDot')
    }, {
        title: '终结技伤害',
        dmg: ({ talent }, dmg) => dmg(talent.q['技能伤害'], 'q')
    }];
exports.defDmgIdx = 3;
exports.mainAttr = 'atk,cpct,cdmg';
exports.buffs = [{
        title: '终结技：使敌方受到的伤害提高[enemydmg]%',
        data: {
            enemydmg: ({ talent }) => talent.q['伤害提高'] * 100
        }
    }, {
        title: '卢卡1命：对处于裂伤状态的敌人造成伤害提高[dmg]%',
        cons: 1,
        data: {
            dmg: 15
        }
    }, {
        title: '卢卡4命：4层【斗志】使攻击力提高[atkPct]%',
        cons: 4,
        data: {
            atkPct: 20
        }
    }];
exports.createdBy = 'Aluxes';

return exports;})();
