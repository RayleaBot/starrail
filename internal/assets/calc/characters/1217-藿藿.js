const characterRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.createdBy = exports.defDmgIdx = exports.mainAttr = exports.details = void 0;
exports.details = [{
        title: '普攻伤害',
        dmg: ({ talent, attr }, { basic }) => basic(talent.a['技能伤害'] * attr.hp, 'a')
    }, {
        title: '战技生命回复·主目标',
        dmg: ({ talent, attr }, { heal }) => heal(attr.hp * talent.e['目标治疗·百分比生命'] + talent.e['目标治疗·固定值'])
    }, {
        title: '战技生命回复·相邻目标',
        dmg: ({ talent, attr }, { heal }) => heal(attr.hp * talent.e['相邻目标治疗·百分比生命'] + talent.e['相邻目标治疗·固定值'])
    }, {
        title: '天赋禳命生命回复',
        dmg: ({ talent, attr }, { heal }) => heal(attr.hp * talent.t['治疗·百分比生命'] + talent.t['治疗·百分比生命'])
    }];
exports.mainAttr = 'hp,cpct,cdmg,heal';
exports.defDmgIdx = 1;
exports.createdBy = 'Aluxes';

return exports;})();
