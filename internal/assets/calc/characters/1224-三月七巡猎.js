const characterRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.createdBy = exports.buffs = exports.mainAttr = exports.defDmgIdx = exports.details = void 0;
exports.details = [{
        title: '强化普攻单次伤害',
        params: { EnhancedBasicAtk: true },
        dmg: ({ talent }, dmg) => dmg(talent.a2['技能伤害'], 'a')
    }, {
        title: '终结技伤害',
        dmg: ({ talent }, dmg) => dmg(talent.q['技能伤害'], 'q')
    }];
exports.defDmgIdx = 0;
exports.mainAttr = 'atk,cpct,cdmg';
exports.buffs = [{
        title: '三月七·巡猎天赋：充能大于等于7点时，三月七立即行动，造成的伤害提高[dmg]%',
        cons: 1,
        data: {
            dmg: ({ talent }) => talent.t['伤害提高'] * 100
        }
    }, {
        check: ({ params }) => params.EnhancedBasicAtk === true,
        title: '三月七·巡猎6魂：施放终结技后，下一次强化普攻造成的暴击伤害提高[aCdmg]%',
        cons: 6,
        data: {
            aCdmg: 50
        }
    }];
exports.createdBy = '冰翼';

return exports;})();
