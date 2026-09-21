const characterRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.createdBy = exports.buffs = exports.defDmgIdx = exports.mainAttr = exports.details = void 0;
exports.details = [{
        title: '普攻伤害',
        dmg: ({ talent }, dmg) => dmg(talent.a['技能伤害'], 'a')
    }, {
        title: '战技提升攻击力',
        dmg: ({ talent }) => {
            return {
                avg: Format.percent(talent.e['攻击力提高']),
                type: 'text'
            };
        }
    }, {
        title: '终结技提升暴击率',
        dmg: ({ talent }) => {
            return {
                avg: Format.percent(talent.q['暴击率提高']),
                type: 'text'
            };
        }
    }, {
        title: '终结技提升暴击伤害',
        dmg: ({ talent }) => {
            return {
                avg: Format.percent(talent.q['暴击伤害提高']),
                type: 'text'
            };
        }
    }, {
        title: '终结技伤害',
        dmg: ({ talent }, dmg) => dmg(talent.q['技能伤害'], 'q')
    }];
exports.mainAttr = 'atk,cpct,cdmg';
exports.defDmgIdx = 2;
exports.buffs = [{
        title: '战技-天阙鸣弦：攻击力提高[atkPct]%',
        data: {
            atkPct: ({ talent }) => talent.e['攻击力提高'] * 100
        }
    }, {
        title: '终结技-贯云饮羽：暴击率提高[cpct]%，暴击伤害提高[cdmg]%',
        data: {
            cpct: ({ talent }) => talent.q['暴击率提高'] * 100,
            cdmg: ({ talent }) => talent.q['暴击伤害提高'] * 100
        }
    }, {
        title: '行迹-迟彝：造成虚数伤害提高[dmg]%',
        tree: 1,
        data: {
            dmg: 12
        }
    }, {
        title: '驭空1命：速度提高[speedPct]%',
        cons: 1,
        data: {
            speedPct: 10
        }
    }, {
        title: '驭空4命：造成伤害提高[dmg]%',
        cons: 4,
        data: {
            dmg: 30
        }
    }];
exports.createdBy = 'Aluxes';

return exports;})();
