const characterRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.createdBy = exports.buffs = exports.mainAttr = exports.defDmgIdx = exports.details = void 0;
exports.details = [{
        title: '普攻伤害',
        dmg: ({ talent }, dmg) => dmg(talent.a['技能伤害'], 'a')
    }, {
        title: '战技伤害',
        dmg: ({ talent }, dmg) => dmg(talent.e['单体伤害'], 'e')
    }, {
        title: '强化战技伤害·主目标',
        dmg: ({ talent }, dmg) => dmg(talent.e2['技能伤害'], 'e')
    }, {
        title: '强化战技伤害·副目标',
        dmg: ({ talent }, dmg) => dmg(talent.e2['相邻目标伤害'], 'e')
    }, {
        title: '终结技伤害',
        dmg: ({ talent }, dmg) => dmg(talent.q['技能伤害'], 'q')
    }, {
        title: '战技持续伤害',
        dmg: ({ talent }, dmg) => dmg(talent.e['持续伤害'], 'dot', 'skillDot')
    }];
exports.defDmgIdx = 2;
exports.mainAttr = 'atk,cpct,cdmg,dmg';
exports.buffs = [{
        title: '虎克1命：强化战技造成的伤害提高[eDmg]%',
        cons: 1,
        data: {
            eDmg: 20
        }
    }, {
        title: '虎克6命：对灼烧状态下的地方目标造成的伤害提高[dmg]%',
        cons: 6,
        data: {
            dmg: 20
        }
    }];
exports.createdBy = 'Aluxes';

return exports;})();
