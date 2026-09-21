const artifactBuffs=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
let attr = (key, val) => {
    let data = {};
    data[key] = val;
    return {
        isStatic: true,
        data
    };
};
exports.default = {
    云无留迹的过客: {
        2: attr('heal', 10)
    },
    野穗伴行的快枪手: {
        2: attr('atkPct', 12),
        4: [attr('speedPct', 6), {
                title: '普攻伤害提高10%',
                data: {
                    aDmg: 10
                }
            }]
    },
    净庭教宗的圣骑士: {
        2: attr('defPct', 15),
        4: {
            title: '护盾强效提高20%',
            data: {
                shield: 20
            }
        }
    },
    密林卧雪的猎人: {
        2: attr('ice', 10),
        4: {
            title: '释放终结技后2回合，爆伤提高25%',
            data: {
                cdmg: 25
            }
        }
    },
    街头出身的拳王: {
        2: attr('phy', 10),
        4: {
            title: '攻击或被攻击5层Buff提高攻击力25%',
            data: {
                atkPct: 25
            }
        }
    },
    熔岩锻铸的火匠: {
        2: attr('fire', 10),
        4: {
            title: '战技造成的伤害提高12%，释放终结技下一次的火属性伤害提高12%',
            data: {
                eDmg: 12,
                dmg: 12 // todo: 检查属性
            }
        }
    },
    繁星璀璨的天才: {
        2: attr('quantum', 10),
        4: {
            title: '对有量子弱点的目标造成伤害时，无视其20%的防御力',
            data: {
                ignore: 20
            }
        }
    },
    激奏雷电的乐队: {
        2: attr('elec', 10),
        4: {
            title: '释放战技时，攻击力提高20%',
            data: {
                atkPct: 20
            }
        }
    },
    晨昏交界的翔鹰: {
        2: attr('wind', 10)
    },
    流星追迹的怪盗: {
        2: attr('stance', 16),
        4: attr('stance', 16)
    },
    盗匪荒漠的废土客: {
        2: attr('imaginary', 10),
        4: {
            title: '对陷入负面效果的目标的暴击率提高10%，对禁锢状态的目标暴击率提高20%',
            data: {
                cpct: 10
            }
        }
    },
    宝命长存的莳者: {
        2: attr('hpPct', 12),
        4: {
            title: '消耗生命2层提高暴击率16%',
            data: {
                cpct: 16
            }
        }
    },
    骇域漫游的信使: {
        2: attr('speedPct', 6),
        4: {
            title: '对我方释放终结技时速度提高12%',
            data: {
                speedPct: 12
            }
        }
    },
    太空封印站: {
        2: [attr('atkPct', 12), {
                title: '速度大于等于120提高攻击力12%',
                check: ({ calc, attr }) => calc(attr.speed) > 120,
                data: {
                    atkPct: 12
                }
            }]
    },
    不老者的仙舟: {
        2: [attr('hpPct', 12), {
                title: '速度大于等于120提高攻击力8%',
                check: ({ calc, attr }) => calc(attr.speed) > 120,
                data: {
                    atkPct: 8
                }
            }]
    },
    泛银河商业公司: {
        2: [attr('effPct', 10), {
                title: '基于效果命中提高攻击力[atkPct]%',
                data: {
                    atkPct: ({ calc, attr }) => Math.min(25, calc(attr.effPct) / 4 || 0)
                }
            }]
    },
    筑城者的贝洛伯格: {
        2: [attr('defPct', 15), {
                title: '效果命中大于50%时，提高防御力15%',
                check: ({ calc, attr }) => calc(attr.effPct) > 50,
                data: {
                    defPct: 15
                }
            }]
    },
    星体差分机: {
        2: attr('cdmg', 16)
    },
    停转的萨尔索图: {
        2: [attr('cpct', 8), {
                title: '终结技与追加攻击造成的伤害提高15%',
                check: ({ attr, calc }) => {
                    return calc(attr.cpct) > 50;
                },
                data: {
                    qDmg: 15,
                    tDmg: 15
                }
            }]
    },
    盗贼公国塔利亚: {
        2: [attr('stance', 16), {
                title: '速度大于145时，击破特攻提高20%',
                check: ({ calc, attr }) => calc(attr.speed) > 145,
                data: {
                    stance: 20
                }
            }]
    },
    生命的翁瓦克: {
        2: attr('recharge', 5)
    },
    繁星竞技场: {
        2: [attr('cpct', 8), {
                title: '暴击率大于等于70%时，普攻和战技造成的伤害提高20%',
                check: ({ calc, attr }) => calc(attr.cpct) >= 70,
                data: {
                    aDmg: 20,
                    eDmg: 20
                }
            }]
    },
    折断的龙骨: {
        2: [attr('effDef', 10), {
                title: '效果抵抗大于等于30%时，暴击伤害提高10%',
                check: ({ calc, attr }) => calc(attr.effDef) >= 30,
                data: {
                    cdmg: 10
                }
            }]
    },
    毁烬焚骨的大公: {
        2: {
            title: '追加攻击造成的伤害提高[tDmg]%',
            data: {
                tDmg: 20
            }
        },
        4: {
            title: '计算[_buffCount]层，提高攻击力[atkPct]%',
            data: {
                _buffCount: ({ params }) => params.tArtisBuffCount || 3,
                atkPct: ({ params }) => (params.tArtisBuffCount || 3) * 6
            }
        }
    },
    幽锁深牢的系囚: {
        2: attr('atkPct', 12),
        4: {
            title: '3层持续伤害下无视18%防御力',
            data: {
                ignore: 18
            }
        }
    },
    苍穹战线格拉默: {
        2: [attr('atkPct', 12), {
                title: '装备者速度大于[_speed]时，伤害提高[dmg]%',
                check: ({ attr }) => attr.speed >= 135,
                data: {
                    _speed: ({ attr }) => attr.speed < 160 ? 135 : 160,
                    dmg: ({ attr }) => attr.speed < 160 ? 12 : 18
                }
            }]
    },
    梦想之地匹诺康尼: {
        2: [attr('recharge', 5), {
                title: '同属性角色伤害提高10%',
                data: {
                    dmg: 10
                }
            }]
    },
    死水深潜的先驱: {
        2: {
            title: '对受负面状态影响的敌人造成伤害提高[dmg]%',
            data: {
                dmg: 12
            }
        },
        4: [attr('cpct', 4), {
                title: '对陷入不少于3个负面效果的敌方目标造成的暴击率额外提高[cpct]%，暴击伤害提高[cdmg]%',
                data: {
                    cpct: 4,
                    cdmg: 24
                }
            }]
    },
    机心戏梦的钟表匠: {
        2: attr('stance', 16),
        4: {
            title: '装备者对我方释放终结技时，我方全体击破特攻提高[stance]%',
            data: {
                stance: 30
            }
        }
    },
    无主荒星茨冈尼亚: {
        2: [attr('cpct', 4), {
                title: '敌方目标被消灭提高暴击伤害，至多提高[cdmg]%',
                data: {
                    cdmg: 40
                }
            }]
    },
    出云显世与高天神国: {
        2: [attr('atkPct', 12), {
                title: '存在一名与装备者命途相同的队友时，暴击率提高[cpct]%',
                data: {
                    cpct: 12
                }
            }]
    },
    荡除蠹灾的铁骑: {
        2: attr('stance', 16),
        4: [{
                title: '基于击破特攻，造成的击破伤害无视敌方[breakIgnore]%防御',
                sort: 9,
                data: {
                    breakIgnore: ({ attr }) => attr.stance >= 150 ? 10 : 0
                }
            }, {
                title: '基于击破特攻，造成的超击破伤害无视敌方[superBreakIgnore]%防御',
                sort: 9,
                data: {
                    superBreakIgnore: ({ attr }) => attr.stance >= 250 ? 15 : 0
                }
            }]
    },
    风举云飞的勇烈: {
        2: attr('atkPct', 12),
        4: [attr('cpct', 6), {
                title: '施放追加攻击使终结技伤害提高[qDmg]%',
                data: {
                    qDmg: 36
                }
            }]
    },
    奔狼的都蓝王朝: {
        2: {
            title: '5层Buff使得装备者造成的追击伤害提高[tDmg]%，装备者的暴击伤害提高[cdmg]%',
            data: {
                tDmg: 25,
                cdmg: 25
            }
        }
    },
    劫火莲灯铸炼宫: {
        2: [attr('speedPct', 6), {
                title: '击中火弱点的敌方时，击破特攻提高[stance]%',
                data: {
                    stance: 40
                }
            }]
    },
    奇想蕉乐园: {
        2: [attr('cdmg', 16), {
                title: '存在装备者召唤的目标时，暴击伤害额外提高[cdmg]%',
                data: {
                    cdmg: 32
                }
            }]
    },
    沉陆海域露莎卡: {
        2: attr('recharge', 5)
    },
    重循苦旅的司铎: {
        2: attr('speedPct', 6)
    },
    识海迷坠的学者: {
        2: attr('cpct', 8),
        4: {
            title: '终结技造成的伤害提高[qDmg]%，施放终结技后战技伤害提高[eDmg]%',
            data: {
                eDmg: 45,
                qDmg: 20
            }
        }
    },
    凯歌祝捷的英豪: {
        2: attr('atkPct', 12),
        4: {
            title: '装备者的忆灵在场时，装备者的速度提高[speedPct]%。装备者的忆灵攻击时，装备者和忆灵的暴击伤害提高[cdmg]%',
            data: {
                speedPct: 6,
                cdmg: 30
            }
        }
    },
    哀歌覆国的诗人: {
        2: attr('quantum', 10),
        4: [attr('speedPct', -8), {
                check: ({ attr, calc }) => calc(attr.staticAttr.speed) < 110,
                title: '进入战斗时，若装备者的速度小于[_speed]，使装备者的暴击率提高[cpct]%',
                data: {
                    _speed: ({ attr, calc }) => calc(attr.staticAttr.speed) < 95 ? 95 : 110,
                    cpct: ({ attr, calc }) => calc(attr.staticAttr.speed) < 95 ? 32 : 20
                }
            }]
    },
    谧宁拾骨地: {
        2: [attr('hpPct', 12), {
                title: '生命值大于等于5000时暴击伤害提升28%',
                check: ({ calc, attr }) => calc(attr.hp) >= 5000,
                data: {
                    cdmg: 28
                }
            }]
    },
    渊思寂虑的巨树: {
        2: [attr('speedPct', 6), {
                title: '速度大于等于[_speed]时，治疗量提高[heal]%',
                check: ({ attr }) => attr.speed >= 135,
                data: {
                    _speed: ({ attr }) => attr.speed >= 180 ? 180 : 135,
                    heal: ({ attr }) => attr.speed >= 180 ? 20 : 12
                }
            }]
    },
    烈阳惊雷的女武神: {
        2: attr('speedPct', 6),
        4: {
            title: '当装备者及其忆灵为装备者及其忆灵以外的我方目标提供治疗后，速度提高[speedPct]%，我方全体暴击伤害提高[cdmg]%',
            data: {
                speedPct: 6,
                cdmg: 15
            }
        }
    },
    恶海逐波的船长: {
        2: attr('cdmg', 16),
        4: {
            title: '消耗2层【助力】，使装备者攻击力提高[atkPct]%',
            data: {
                atkPct: 48
            }
        }
    },
    妖精织梦的乐园: {
        2: {
            title: '造成的伤害提高[dmg]%',
            data: {
                dmg: 36
            }
        }
    },
    沉欢醉饮的海隅: {
        2: [attr('atkPct', 12), {
                check: ({ calc, attr }) => calc(attr.atk) >= 2400,
                title: '装备者攻击力大于等于[_atk]时，持续伤害额外提高[dotDmg]%',
                data: {
                    _atk: ({ attr }) => attr.atk < 3600 ? 2400 : 3600,
                    dotDmg: ({ attr }) => attr.atk < 3600 ? 12 : 24
                }
            }]
    },
    自匿星芒的隐士: {
        2: attr('shield', 10),
        4: {
            title: '护盾量提高[shield]%',
            data: {
                shield: 12
            }
        }
    },
    再创天地的救世主: {
        2: attr('cpct', 8),
        4: {
            check: ({ params }) => params.Memosprite,
            title: '若装备者的忆灵在场，使装备者及其忆灵生命上限提高[hpPct]% ，我方全体造成的伤害提高[dmg]]%',
            data: {
                hpPct: 24,
                dmg: 15
            }
        }
    },
    永恒之地翁法罗斯: {
        2: [
            attr('cpct', 8), {
                check: ({ params }) => params.Memosprite,
                title: '装备者的忆灵在场时，我方全体速度提高[speedPct]%',
                data: {
                    speedPct: 8
                }
            }
        ]
    },
    '天国@直播间': {
        2: [attr('cdmg', 16), {
                title: '暴击伤害额外提高[cdmg]%',
                data: {
                    cdmg: 32
                }
            }]
    },
    闪耀功勋的魔法少女: {
        2: attr('cdmg', 16),
        4: [attr('ignore', 10), {
                title: '计算[_buffCount]层，造成的欢愉伤害额外无视目标[xeIgnore]%的防御力',
                data: {
                    _buffCount: ({ params }) => params.tArtisBuffCount || 10,
                    xeIgnore: ({ params }) => (params.tArtisBuffCount || 10) * 1
                }
            }]
    },
    应天涉远的卜者: {
        2: attr('speedPct', 6),
        4: {
            title: '进入战斗前，若装备者的速度大于等于[_speed]，使装备者的暴击率提高[cpct]%。装备者施放欢愉技时，使我方全体欢愉度提高[joy]%',
            data: {
                _speed: ({ attr, calc }) => calc(attr.staticAttr.speed) < 160 ? calc(attr.staticAttr.speed) < 120 ? 0 : 120 : 160,
                cpct: ({ attr, calc }) => calc(attr.staticAttr.speed) < 160 ? calc(attr.staticAttr.speed) < 120 ? 0 : 10 : 18,
                joy: 10
            }
        }
    },
    零号关卡朋克洛德: {
        2: [attr('joy', 8), {
                check: ({ attr }) => attr.joy.base >= 40,
                title: '战斗中欢愉度首次达到[joy]%时，使装备者暴击伤害提高[cdmg]%',
                data: {
                    joy: ({ attr, calc }) => {
                        let _joy = calc(attr.joy);
                        return _joy < 80 ? 40 : 80;
                    },
                    cdmg: ({ attr, calc }) => {
                        let joy = calc(attr.joy);
                        return joy < 80 ? 20 : 32;
                    }
                }
            }]
    },
    千星荟萃之城: {
        2: [{
                title: '装备者施放追加攻击时，攻击力提高[atkPct]%。当敌方目标被消灭时，我方全体在本场战斗中暴击伤害提高[cdmg]%',
                data: {
                    atkPct: 24,
                    cdmg: 12
                }
            }]
    },
    星如我见的领航员: {
        2: attr('atkPct', 12),
        4: {
            title: '3层Buff使战技造成的伤害提高[eDmg]%，终结技造成的伤害提高[qDmg]%',
            data: {
                eDmg: 54,
                qDmg: 54
            }
        }
    },
    叩问天工的名冶: {
        2: attr('hpPct', 12),
        4: {
            title: '装备者对防御力降低状态的敌方目标造成的暴击伤害提高[cdmg]% ，持有【助燃】的我方目标造成的伤害提高[dmg]%',
            data: {
                cdmg: 28,
                dmg: 15
            }
        }
    },
    寰宇生研院: {
        2: [{
                check: ({ attr }) => attr.sp >= 200,
                title: '装备者能量上限大于等于200点，使装备者造成的伤害提高[dmg]%。',
                data: {
                    dmg: ({ attr }) => { return (attr.sp - 200) * 0.2 > 32 ? 32 : (attr.sp - 200) * 0.2; }
                }
            }]
    },
    坠星启航地: {
        2: [attr('cpct', 8), {
                title: '使装备者的暴击伤害提高[cdmg]%',
                data: {
                    cdmg: 32
                }
            }]
    },
    戏梦点星的伶人: {
        2: attr('speedPct', 6),
        4: {
            check: ({ params }) => params.certifiedBanger >= 10,
            title: '若装备者持有的好活当赏大于等于10点，则使我方全体目标的暴击伤害提高[cpct]%',
            data: {
                cpct: 12
            }
        }
    },
    贪噬禁果的异端: {
        2: attr('cpct', 16),
        4: {
            title: '使装备者普攻造成的伤害提高[aDmg]%。当装备者施放普攻时，使装备者的攻击力提高[atkPct]%',
            data: {
                aDmg: 36,
                atkPct: 15
            }
        }
    }
};

return exports;})().default;
const weaponBuffs={};
const artiMeta=(()=>{const attrs=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.attrMap = exports.subAttr = exports.mainAttr = void 0;
exports.mainAttr = {
    3: 'atk,def,hp,cpct,cdmg,heal,effPct'.split(','),
    4: 'atk,def,hp,speed'.split(','),
    5: 'atk,def,hp,dmg'.split(','),
    6: 'atk,def,hp,recharge,stance'.split(',')
};
exports.subAttr = 'atk,atkPlus,def,defPlus,hp,hpPlus,speed,cpct,cdmg,effPct,effDef,stance'.split(',');
/**
 * 圣遗物词条配置
 * @[]value 副词条单次提升值（最大档位）
 * @[]valueMin 副词条单次提升最小值
 * @[]calc 伤害计算时变更的字段type
 * @[]type 词条的类型 normal:普通字段 plus:小词条
 * @[]base 词条类型为小词条时，对应的大词条
 * @[]text 展示文字
 */
const attrMap = {
    atk: { title: '大攻击', format: 'pct', calc: 'pct', value: 4.32 },
    atkPlus: { title: '小攻击', format: 'comma', value: 127 / 6 },
    def: { title: '大防御', format: 'pct', calc: 'pct', value: 5.4 },
    defPlus: { title: '小防御', format: 'comma', value: 127 / 6 },
    hp: { title: '大生命', format: 'pct', calc: 'pct', value: 4.32 },
    hpPlus: { title: '小生命', format: 'comma', value: 254 / 6 },
    speed: { title: '速度', format: 'comma', calc: 'plus', value: 2.6 },
    cpct: { title: '暴击率', format: 'pct', calc: 'plus', value: 3.24 },
    cdmg: { title: '暴击伤害', format: 'pct', calc: 'plus', value: 6.48 },
    recharge: { title: '充能效率', format: 'pct', calc: 'plus', value: 3.11 },
    dmg: { title: '伤害加成', format: 'pct', value: 6.22 },
    heal: { title: '治疗加成', format: 'pct', calc: 'pct', value: 5.53 },
    stance: { title: '击破特攻', format: 'pct', value: 6.48, calc: 'pct' },
    effPct: { title: '效果命中', format: 'pct', value: 4.32, calc: 'pct' },
    effDef: { title: '效果抵抗', format: 'pct', value: 4.32, calc: 'pct' }
};
exports.attrMap = attrMap;
lodash.forEach(attrMap, (attr, key) => {
    // 设置value
    if (!attr.value) {
        return true;
    }
    // 设置type
    attr.base = { hpPlus: 'hp', atkPlus: 'atk', defPlus: 'def' }[key];
    attr.type = attr.base ? 'plus' : 'normal';
    // 设置展示文字
    attr.text = Format[attr.format](attr.value, 2);
});

return exports;})();const alias=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.aliasCfg = exports.artiSetAbbr = exports.artiAbbr = void 0;
exports.artiAbbr = {
    快枪手的野穗毡帽: '快枪手的毡帽',
    快枪手的粗革手套: '快枪手的手套',
    快枪手的猎风披肩: '快枪手的披肩',
    快枪手的铆钉马靴: '快枪手的铆钉靴',
    废土客的呼吸面罩: '废土客的面罩',
    废土客的荒漠终端: '废土客的终端',
    废土客的修士长袍: '废土客的长袍',
    废土客的动力腿甲: '废土客的腿甲',
    '「黑塔」的空间站点': '黑塔的空间站点',
    '「黑塔」的漫历轨迹': '黑塔的漫历轨迹',
    罗浮仙舟的天外楼船: '罗浮仙舟的楼船',
    罗浮仙舟的建木枝蔓: '罗浮仙舟的建木',
    贝洛伯格的存护堡垒: '贝洛伯格的堡垒',
    贝洛伯格的铁卫防线: '贝洛伯格的防线',
    螺丝星的机械烈阳: '螺丝星的烈阳',
    螺丝星的环星孔带: '螺丝星的孔带',
    萨尔索图的移动城市: '萨尔索图的城市',
    萨尔索图的晨昏界线: '萨尔索图的界线',
    塔利亚的钉壳小镇: '塔利亚的小镇',
    塔利亚的裸皮电线: '塔利亚的电线',
    翁瓦克的诞生之岛: '翁瓦克的诞生岛',
    翁瓦克的环岛海岸: '翁瓦克的海岸',
    泰科铵的镭射球场: '泰科铵的球场',
    泰科铵的弧光赛道: '泰科铵的赛道',
    伊须磨洲的残船鲸落: '伊须磨洲的鲸落',
    伊须磨洲的坼裂缆索: '伊须磨洲的缆索',
    格拉默的铁骑兵团: '格拉默的兵团',
    格拉默的寂静坟碑: '格拉默的坟碑',
    匹诺康尼的堂皇酒店: '匹诺康尼的酒店',
    匹诺康尼的逐梦轨道: '匹诺康尼的轨道',
    钟表匠的极目透镜: '钟表匠的透镜',
    钟表匠的交运腕表: '钟表匠的腕表',
    钟表匠的空幻礼服: '钟表匠的礼服',
    钟表匠的隐梦革履: '钟表匠的革履',
    铸炼宫的莲华灯芯: '铸炼宫的灯芯',
    铸炼宫的焰轮天绸: '铸炼宫的天绸'
};
exports.artiSetAbbr = {
    盗匪荒漠的废土客: '虚数套',
    繁星璀璨的天才: '量子套',
    激奏雷电的乐队: '雷套',
    熔岩锻铸的火匠: '火套',
    晨昏交界的翔鹰: '风套',
    野穗伴行的快枪手: '快枪手',
    密林卧雪的猎人: '冰套',
    流星追迹的怪盗: '怪盗',
    街头出身的拳王: '拳王',
    云无留迹的过客: '过客',
    戍卫风雪的铁卫: '铁卫',
    净庭教宗的圣骑士: '圣骑士',
    筑城者的贝洛伯格: '筑城者',
    星体差分机: '差分机',
    停转的萨尔索图: '停转',
    不老者的仙舟: '仙舟',
    盗贼公国塔利亚: '盗贼',
    泛银河商业公司: '泛银河',
    太空封印站: '太空',
    生命的翁瓦克: '翁瓦克',
    繁星竞技场: '竞技场',
    折断的龙骨: '龙骨',
    宝命长存的莳者: '莳者',
    骇域漫游的信使: '信使',
    毁烬焚骨的大公: '大公',
    幽锁深牢的系囚: '系囚',
    苍穹战线格拉默: '苍穹',
    梦想之地匹诺康尼: '梦想',
    死水深潜的先驱: '死水',
    机心戏梦的钟表匠: '机心',
    无主荒星茨冈尼亚: '荒星',
    出云显世与高天神国: '出云',
    荡除蠹灾的铁骑: '铁骑',
    风举云飞的勇烈: '勇烈',
    奔狼的都蓝王朝: '奔狼',
    劫火莲灯铸炼宫: '劫火',
    奇想蕉乐园: '蕉乐园',
    沉陆海域露莎卡: '露莎卡',
    重循苦旅的司铎: '司铎',
    识海迷坠的学者: '学者',
    凯歌祝捷的英豪: '英豪',
    哀歌覆国的诗人: '诗人',
    谧宁拾骨地: '拾骨',
    渊思寂虑的巨树: '巨树',
    烈阳惊雷的女武神: '女武神',
    恶海逐波的船长: '船长',
    沉欢醉饮的海隅: '海隅',
    妖精织梦的乐园: '乐园',
    再创天地的救世主: '救世主',
    自匿星芒的隐士: '隐士',
    '天国@直播间': '天国',
    永恒之地翁法罗斯: '翁法罗斯',
    应天涉远的卜者: '卜者',
    闪耀功勋的魔法少女: '少女',
    零号关卡朋克洛德: '零号',
    千星荟萃之城: '荟萃之城',
    叩问天工的名冶: '名冶',
    星如我见的领航员: '领航员',
    坠星启航地: '坠星,启航',
    寰宇生研院: '寰宇,生研院'
};
exports.aliasCfg = {
    盗匪荒漠的废土客: '虚数套,废土',
    繁星璀璨的天才: '量子套',
    激奏雷电的乐队: '雷套,雷电',
    熔岩锻铸的火匠: '火套',
    晨昏交界的翔鹰: '风套',
    野穗伴行的快枪手: '快枪手,快枪',
    密林卧雪的猎人: '冰套',
    流星追迹的怪盗: '怪盗',
    街头出身的拳王: '拳王,物理套',
    云无留迹的过客: '过客,云无留迹',
    戍卫风雪的铁卫: '铁卫,风雪',
    净庭教宗的圣骑士: '圣骑士,圣骑士',
    筑城者的贝洛伯格: '筑城者,贝洛伯格,筑城',
    星体差分机: '差分机,星体',
    停转的萨尔索图: '停转,萨尔索图,萨尔',
    不老者的仙舟: '仙舟,不老者,不老',
    盗贼公国塔利亚: '盗贼,塔利亚',
    泛银河商业公司: '泛银河,银河,商业',
    太空封印站: '太空,封印站',
    生命的翁瓦克: '翁瓦克,生命',
    繁星竞技场: '繁星,竞技场',
    折断的龙骨: '龙骨',
    宝命长存的莳者: '莳者,时者',
    骇域漫游的信使: '信使,信使套',
    毁烬焚骨的大公: '毁烬,焚骨,毁烬焚骨,追加套,追加攻击套,追伤套',
    幽锁深牢的系囚: '幽锁,深牢,幽锁深牢,持续伤害套',
    苍穹战线格拉默: '苍穹,战线,苍穹战线,格拉默',
    梦想之地匹诺康尼: '梦想,梦想之地,匹诺康尼',
    死水深潜的先驱: '死水,深潜,死水深潜,死水套,深潜套',
    机心戏梦的钟表匠: '机心,钟表匠,机心套,钟表套',
    无主荒星茨冈尼亚: '荒星,无主荒星,茨冈尼亚',
    出云显世与高天神国: '出云,出云显世,高天神国',
    荡除蠹灾的铁骑: '铁骑,铁骑套,超击破套',
    风举云飞的勇烈: '风举云飞,勇烈,勇烈套,终追套',
    奔狼的都蓝王朝: '都蓝王朝,都蓝,都蓝套,奔狼套',
    劫火莲灯铸炼宫: '劫火莲灯,铸炼宫,劫火,劫火套',
    奇想蕉乐园: '奇想,蕉乐园,蕉蕉,乐园,奇想套,蕉乐园套,蕉蕉套,乐园套',
    沉陆海域露莎卡: '沉陆,海域,露莎卡,沉陆套,海域套,露莎卡套',
    重循苦旅的司铎: '司铎,司铎套',
    识海迷坠的学者: '学者,学者套',
    凯歌祝捷的英豪: '凯歌,英豪,凯歌套,英豪套',
    哀歌覆国的诗人: '诗人,诗人套',
    谧宁拾骨地: '拾骨,骨地,拾骨套,骨地套',
    渊思寂虑的巨树: '巨树,巨树套',
    烈阳惊雷的女武神: '女武神,女武神套',
    恶海逐波的船长: '船长,船长套',
    妖精织梦的乐园: '妖精,织梦,乐园,妖精乐园',
    沉欢醉饮的海隅: '沉欢醉饮的海偶,沉欢醉饮,海隅,海偶',
    再创天地的救世主: '救世主',
    自匿星芒的隐士: '隐士',
    '天国@直播间': '天国直播间,天国,直播间',
    永恒之地翁法罗斯: '翁法罗斯,永恒之地,永恒',
    应天涉远的卜者: '应天涉远,卜者',
    闪耀功勋的魔法少女: '闪耀功勋,魔法少女',
    零号关卡朋克洛德: '零号关卡,朋克洛德',
    千星荟萃之城: '千星,荟萃之城',
    叩问天工的名冶: '天工,名冶',
    星如我见的领航员: '领航员,我见',
    坠星启航地: '坠星,启航',
    寰宇生研院: '寰宇,生研院'
};

return exports;})();const mark=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.usefulAttr = void 0;
/**
 * 角色的默认评分规则
 * 如character/${name}/artis.js下有角色自定义规则优先使用自定义
 */
exports.usefulAttr = {
    真珠: { hp: 0, atk: 0, def: 100, speed: 75, cpct: 0, cdmg: 0, stance: 0, heal: 100, recharge: 100, effPct: 0, effDef: 100, dmg: 0 },
    '知更鸟•晴歌': { hp: 100, atk: 0, def: 0, speed: 100, cpct: 50, cdmg: 50, stance: 0, heal: 0, recharge: 100, effPct: 0, effDef: 0, dmg: 0 },
    '砂金•戏浪': { hp: 100, atk: 0, def: 0, speed: 100, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 0, effPct: 0, effDef: 0, dmg: 0 },
    远坂凛: { hp: 0, atk: 100, def: 0, speed: 75, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 100, effPct: 0, effDef: 0, dmg: 100 },
    吉尔伽美什: { hp: 0, atk: 100, def: 0, speed: 0, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 0, effPct: 0, effDef: 0, dmg: 100 },
    '姬子•启行': { hp: 0, atk: 100, def: 0, speed: 0, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 50, effPct: 0, effDef: 0, dmg: 100 },
    '千冶•刃': { hp: 100, atk: 0, def: 0, speed: 50, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 50, effPct: 0, effDef: 0, dmg: 50 },
    绯英: { hp: 0, atk: 100, def: 0, speed: 50, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 100, effPct: 0, effDef: 0, dmg: 100 },
    '银狼LV.999': { hp: 10, atk: 0, def: 10, speed: 100, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 0, effPct: 0, effDef: 0, dmg: 0 },
    不死途: { hp: 0, atk: 100, def: 0, speed: 50, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 0, effPct: 0, effDef: 0, dmg: 100 },
    火花: { hp: 0, atk: 75, def: 0, speed: 50, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 50, effPct: 0, effDef: 0, dmg: 0 },
    爻光: { hp: 75, atk: 0, def: 0, speed: 100, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 100, effPct: 0, effDef: 0, dmg: 0 },
    大丽花: { hp: 20, atk: 100, def: 20, speed: 100, cpct: 0, cdmg: 0, stance: 100, heal: 0, recharge: 30, effPct: 0, effDef: 30, dmg: 0 },
    昔涟: { hp: 100, atk: 0, def: 0, speed: 100, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 50, effPct: 0, effDef: 0, dmg: 100 },
    '丹恒•腾荒': { hp: 0, atk: 100, def: 0, speed: 100, cpct: 50, cdmg: 50, stance: 0, heal: 0, recharge: 100, effPct: 0, effDef: 0, dmg: 50 },
    长夜月: { hp: 100, atk: 0, def: 0, speed: 100, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 50, effPct: 0, effDef: 0, dmg: 100 },
    海瑟音: { hp: 0, atk: 100, def: 0, speed: 100, cpct: 0, cdmg: 0, stance: 0, heal: 0, recharge: 0, effPct: 100, effDef: 0, dmg: 100 },
    刻律德菈: { hp: 0, atk: 100, def: 0, speed: 100, cpct: 0, cdmg: 100, stance: 0, heal: 0, recharge: 100, effPct: 0, effDef: 0, dmg: 100 },
    白厄: { hp: 0, atk: 75, def: 0, speed: 0, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 0, effPct: 0, effDef: 0, dmg: 100 },
    赛飞儿: { hp: 0, atk: 75, def: 0, speed: 100, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 100, effPct: 75, effDef: 0, dmg: 100 },
    风堇: { hp: 100, atk: 0, def: 0, speed: 100, cpct: 0, cdmg: 75, stance: 0, heal: 100, recharge: 100, effPct: 0, effDef: 50, dmg: 0 },
    那刻夏: { hp: 0, atk: 75, def: 0, speed: 100, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 100, effPct: 0, effDef: 0, dmg: 100 },
    遐蝶: { hp: 100, atk: 0, def: 0, speed: 0, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 0, effPct: 0, effDef: 30, dmg: 100 },
    万敌: { hp: 100, atk: 0, def: 0, speed: 100, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 100, effPct: 0, effDef: 0, dmg: 100 },
    缇宝: { hp: 100, atk: 0, def: 0, speed: 0, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 100, effPct: 0, effDef: 0, dmg: 100 },
    阿格莱雅: { hp: 0, atk: 75, def: 0, speed: 100, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 0, effPct: 0, effDef: 0, dmg: 100 },
    大黑塔: { hp: 0, atk: 75, def: 0, speed: 100, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 0, effPct: 0, effDef: 0, dmg: 100 },
    忘归人: { hp: 50, atk: 0, def: 50, speed: 100, cpct: 0, cdmg: 0, stance: 100, heal: 0, recharge: 100, effPct: 100, effDef: 0, dmg: 0 },
    星期日: { hp: 75, atk: 0, def: 75, speed: 100, cpct: 0, cdmg: 100, stance: 0, heal: 0, recharge: 100, effPct: 0, effDef: 50, dmg: 0 },
    乱破: { hp: 30, atk: 100, def: 30, speed: 100, cpct: 0, cdmg: 0, stance: 100, heal: 0, recharge: 0, effPct: 0, effDef: 30, dmg: 0 },
    灵砂: { hp: 0, atk: 75, def: 0, speed: 100, cpct: 0, cdmg: 0, stance: 100, heal: 100, recharge: 100, effPct: 0, effDef: 0, dmg: 0 },
    飞霄: { hp: 0, atk: 75, def: 0, speed: 75, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 100, effPct: 0, effDef: 0, dmg: 100 },
    云璃: { hp: 0, atk: 75, def: 0, speed: 0, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 100, effPct: 0, effDef: 0, dmg: 100 },
    貊泽: { hp: 0, atk: 75, def: 0, speed: 50, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 100, effPct: 0, effDef: 0, dmg: 100 },
    椒丘: { hp: 75, atk: 50, def: 75, speed: 100, cpct: 0, cdmg: 0, stance: 0, heal: 0, recharge: 0, effPct: 100, effDef: 50, dmg: 0 },
    翡翠: { hp: 0, atk: 75, def: 0, speed: 75, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 50, effPct: 0, effDef: 0, dmg: 100 },
    流萤: { hp: 30, atk: 100, def: 30, speed: 100, cpct: 0, cdmg: 0, stance: 100, heal: 0, recharge: 0, effPct: 0, effDef: 30, dmg: 0 },
    流萤Pro: { hp: 10, atk: 100, def: 0, speed: 100, cpct: 0, cdmg: 0, stance: 100, heal: 0, recharge: 0, effPct: 0, effDef: 30, dmg: 0 },
    波提欧: { hp: 50, atk: 50, def: 50, speed: 100, cpct: 50, cdmg: 50, stance: 100, heal: 0, recharge: 50, effPct: 0, effDef: 0, dmg: 50 },
    知更鸟: { hp: 75, atk: 100, def: 75, speed: 100, cpct: 0, cdmg: 0, stance: 0, heal: 0, recharge: 100, effPct: 0, effDef: 50, dmg: 0 },
    砂金: { hp: 0, atk: 0, def: 100, speed: 100, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 75, effPct: 0, effDef: 50, dmg: 100 },
    加拉赫: { hp: 75, atk: 0, def: 75, speed: 100, cpct: 0, cdmg: 0, stance: 100, heal: 100, recharge: 75, effPct: 0, effDef: 50, dmg: 0 },
    黄泉: { hp: 0, atk: 100, def: 0, speed: 75, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 0, effPct: 0, effDef: 0, dmg: 100 },
    米沙: { hp: 0, atk: 75, def: 0, speed: 100, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 75, effPct: 0, effDef: 0, dmg: 100 },
    花火: { hp: 75, atk: 0, def: 75, speed: 100, cpct: 0, cdmg: 100, stance: 0, heal: 0, recharge: 100, effPct: 0, effDef: 50, dmg: 0 },
    花火Pro: { hp: 75, atk: 0, def: 75, speed: 100, cpct: 0, cdmg: 100, stance: 0, heal: 0, recharge: 100, effPct: 0, effDef: 50, dmg: 0 },
    黑天鹅: { hp: 0, atk: 100, def: 0, speed: 75, cpct: 0, cdmg: 0, stance: 0, heal: 0, recharge: 50, effPct: 100, effDef: 0, dmg: 100 },
    黑天鹅Pro: { hp: 0, atk: 100, def: 0, speed: 75, cpct: 0, cdmg: 0, stance: 0, heal: 0, recharge: 50, effPct: 100, effDef: 0, dmg: 100 },
    真理医生: { hp: 0, atk: 75, def: 0, speed: 100, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 50, effPct: 0, effDef: 0, dmg: 100 },
    '阮•梅': { hp: 75, atk: 0, def: 75, speed: 100, cpct: 0, cdmg: 0, stance: 100, heal: 0, recharge: 100, effPct: 0, effDef: 50, dmg: 0 },
    雪衣: { hp: 0, atk: 75, def: 0, speed: 100, cpct: 100, cdmg: 100, stance: 100, heal: 0, recharge: 50, effPct: 0, effDef: 0, dmg: 100 },
    寒鸦: { hp: 75, atk: 0, def: 75, speed: 100, cpct: 0, cdmg: 0, stance: 0, heal: 0, recharge: 100, effPct: 0, effDef: 50, dmg: 0 },
    银枝: { hp: 0, atk: 75, def: 0, speed: 75, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 75, effPct: 0, effDef: 0, dmg: 100 },
    藿藿: { hp: 100, atk: 0, def: 50, speed: 75, cpct: 0, cdmg: 0, stance: 0, heal: 100, recharge: 100, effPct: 0, effDef: 50, dmg: 0 },
    藿藿Pro: { hp: 100, atk: 0, def: 50, speed: 75, cpct: 0, cdmg: 0, stance: 0, heal: 100, recharge: 100, effPct: 0, effDef: 50, dmg: 0 },
    桂乃芬: { hp: 0, atk: 100, def: 0, speed: 100, cpct: 0, cdmg: 0, stance: 50, heal: 0, recharge: 50, effPct: 75, effDef: 0, dmg: 100 },
    '托帕&账账': { hp: 0, atk: 75, def: 0, speed: 100, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 50, effPct: 0, effDef: 0, dmg: 100 },
    镜流: { hp: 0, atk: 75, def: 0, speed: 100, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 50, effPct: 0, effDef: 0, dmg: 100 },
    镜流Pro: { hp: 100, atk: 0, def: 0, speed: 100, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 50, effPct: 0, effDef: 0, dmg: 100 },
    玲可: { hp: 100, atk: 0, def: 50, speed: 100, cpct: 0, cdmg: 0, stance: 0, heal: 100, recharge: 75, effPct: 0, effDef: 50, dmg: 0 },
    符玄: { hp: 100, atk: 0, def: 50, speed: 100, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 50, effPct: 0, effDef: 50, dmg: 100 },
    驭空: { hp: 50, atk: 75, def: 50, speed: 100, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 100, effPct: 0, effDef: 50, dmg: 100 },
    '丹恒•饮月': { hp: 0, atk: 75, def: 0, speed: 50, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 50, effPct: 0, effDef: 0, dmg: 100 },
    卢卡: { hp: 0, atk: 100, def: 0, speed: 100, cpct: 0, cdmg: 0, stance: 75, heal: 0, recharge: 50, effPct: 75, effDef: 0, dmg: 100 },
    卡芙卡: { hp: 0, atk: 100, def: 0, speed: 100, cpct: 0, cdmg: 0, stance: 50, heal: 0, recharge: 50, effPct: 75, effDef: 0, dmg: 100 },
    卡芙卡Pro: { hp: 0, atk: 100, def: 0, speed: 100, cpct: 0, cdmg: 0, stance: 50, heal: 0, recharge: 50, effPct: 75, effDef: 0, dmg: 100 },
    刃: { hp: 100, atk: 50, def: 0, speed: 100, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 50, effPct: 0, effDef: 0, dmg: 100 },
    刃Pro: { hp: 100, atk: 0, def: 0, speed: 100, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 50, effPct: 0, effDef: 0, dmg: 100 },
    罗刹: { hp: 0, atk: 100, def: 0, speed: 100, cpct: 100, cdmg: 100, stance: 0, heal: 100, recharge: 100, effPct: 0, effDef: 50, dmg: 100 },
    银狼: { hp: 0, atk: 75, def: 0, speed: 100, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 75, effPct: 100, effDef: 0, dmg: 100 },
    银狼Pro: { hp: 0, atk: 75, def: 0, speed: 100, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 75, effPct: 100, effDef: 0, dmg: 100 },
    景元: { hp: 0, atk: 75, def: 0, speed: 100, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 50, effPct: 0, effDef: 0, dmg: 100 },
    希儿: { hp: 0, atk: 75, def: 0, speed: 100, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 50, effPct: 0, effDef: 0, dmg: 100 },
    希儿Pro: { hp: 0, atk: 75, def: 0, speed: 100, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 50, effPct: 0, effDef: 0, dmg: 100 },
    布洛妮娅: { hp: 75, atk: 0, def: 75, speed: 100, cpct: 0, cdmg: 100, stance: 0, heal: 0, recharge: 100, effPct: 0, effDef: 50, dmg: 0 },
    杰帕德: { hp: 50, atk: 0, def: 100, speed: 100, cpct: 0, cdmg: 0, stance: 0, heal: 0, recharge: 100, effPct: 50, effDef: 50, dmg: 0 },
    姬子: { hp: 0, atk: 75, def: 0, speed: 75, cpct: 100, cdmg: 100, stance: 50, heal: 0, recharge: 50, effPct: 0, effDef: 0, dmg: 100 },
    瓦尔特: { hp: 0, atk: 75, def: 0, speed: 75, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 50, effPct: 100, effDef: 0, dmg: 100 },
    瓦尔特Pro: { hp: 0, atk: 75, def: 0, speed: 75, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 50, effPct: 100, effDef: 0, dmg: 100 },
    彦卿: { hp: 0, atk: 75, def: 0, speed: 75, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 75, effPct: 0, effDef: 0, dmg: 100 },
    白露: { hp: 100, atk: 0, def: 50, speed: 75, cpct: 0, cdmg: 0, stance: 0, heal: 100, recharge: 100, effPct: 0, effDef: 50, dmg: 0 },
    克拉拉: { hp: 0, atk: 75, def: 0, speed: 75, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 50, effPct: 0, effDef: 0, dmg: 100 },
    三月七: { hp: 50, atk: 0, def: 100, speed: 100, cpct: 0, cdmg: 0, stance: 0, heal: 0, recharge: 100, effPct: 50, effDef: 50, dmg: 0 },
    丹恒: { hp: 0, atk: 75, def: 0, speed: 75, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 50, effPct: 0, effDef: 0, dmg: 100 },
    阿兰: { hp: 0, atk: 75, def: 0, speed: 75, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 50, effPct: 0, effDef: 0, dmg: 100 },
    艾丝妲: { hp: 0, atk: 75, def: 0, speed: 100, cpct: 75, cdmg: 75, stance: 100, heal: 0, recharge: 100, effPct: 0, effDef: 0, dmg: 100 },
    黑塔: { hp: 0, atk: 75, def: 0, speed: 75, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 50, effPct: 0, effDef: 0, dmg: 100 },
    希露瓦: { hp: 0, atk: 75, def: 0, speed: 75, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 50, effPct: 0, effDef: 0, dmg: 100 },
    娜塔莎: { hp: 100, atk: 0, def: 50, speed: 75, cpct: 0, cdmg: 0, stance: 0, heal: 100, recharge: 75, effPct: 0, effDef: 50, dmg: 0 },
    佩拉: { hp: 75, atk: 0, def: 75, speed: 100, cpct: 0, cdmg: 0, stance: 0, heal: 0, recharge: 100, effPct: 100, effDef: 50, dmg: 0 },
    桑博: { hp: 0, atk: 75, def: 0, speed: 100, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 50, effPct: 100, effDef: 0, dmg: 100 },
    虎克: { hp: 0, atk: 75, def: 0, speed: 75, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 50, effPct: 0, effDef: 0, dmg: 100 },
    青雀: { hp: 0, atk: 75, def: 0, speed: 75, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 50, effPct: 0, effDef: 0, dmg: 100 },
    停云: { hp: 50, atk: 100, def: 0, speed: 100, cpct: 0, cdmg: 0, stance: 0, heal: 0, recharge: 100, effPct: 0, effDef: 0, dmg: 0 },
    素裳: { hp: 0, atk: 75, def: 0, speed: 75, cpct: 100, cdmg: 100, stance: 50, heal: 0, recharge: 50, effPct: 0, effDef: 0, dmg: 100 },
    '三月七·巡猎': { hp: 0, atk: 100, def: 0, speed: 100, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 0, effPct: 0, effDef: 0, dmg: 100 },
    // 主角
    '穹·同谐': { hp: 75, atk: 0, def: 75, speed: 100, cpct: 0, cdmg: 0, stance: 100, heal: 0, recharge: 75, effPct: 0, effDef: 0, dmg: 0 },
    '星·同谐': { hp: 75, atk: 0, def: 75, speed: 100, cpct: 0, cdmg: 0, stance: 100, heal: 0, recharge: 75, effPct: 0, effDef: 0, dmg: 0 },
    '穹·存护': { hp: 50, atk: 0, def: 100, speed: 100, cpct: 0, cdmg: 0, stance: 0, heal: 0, recharge: 0, effPct: 100, effDef: 50, dmg: 0 },
    '星·存护': { hp: 50, atk: 0, def: 100, speed: 100, cpct: 0, cdmg: 0, stance: 0, heal: 0, recharge: 0, effPct: 100, effDef: 50, dmg: 0 },
    '穹·毁灭': { hp: 0, atk: 75, def: 0, speed: 75, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 50, effPct: 0, effDef: 0, dmg: 100 },
    '星·毁灭': { hp: 0, atk: 75, def: 0, speed: 75, cpct: 100, cdmg: 100, stance: 0, heal: 0, recharge: 50, effPct: 0, effDef: 0, dmg: 100 },
    '穹·记忆': { hp: 0, atk: 100, def: 0, speed: 100, cpct: 0, cdmg: 100, stance: 0, heal: 0, recharge: 50, effPct: 0, effDef: 0, dmg: 0 },
    '星·记忆': { hp: 0, atk: 100, def: 0, speed: 100, cpct: 0, cdmg: 100, stance: 0, heal: 0, recharge: 50, effPct: 0, effDef: 0, dmg: 0 },
    '穹·欢愉': { hp: 0, atk: 100, def: 0, speed: 100, cpct: 75, cdmg: 75, stance: 0, heal: 0, recharge: 100, effPct: 0, effDef: 0, dmg: 0 },
    '星·欢愉': { hp: 0, atk: 100, def: 0, speed: 100, cpct: 75, cdmg: 75, stance: 0, heal: 0, recharge: 100, effPct: 0, effDef: 0, dmg: 0 }
};

return exports;})();return {mainAttr:attrs.mainAttr,subAttr:attrs.subAttr,attrMap:attrs.attrMap,usefulAttr:mark.usefulAttr,setAbbr:alias.artiSetAbbr}})();
const usefulAttr=artiMeta.usefulAttr;
Object.assign(weaponBuffs,loadWeaponDefinitions((()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1(staticIdx, keyIdx) {
    return {
        一场术后对话: [
            staticIdx(1, 'recharge'),
            keyIdx('释放终结技时的治疗量提高[qHeal]%', 'qHeal', 2)
        ],
        同一种心情: [
            staticIdx(1, 'heal')
        ],
        嘉果: [],
        时节不居: [
            staticIdx(1, 'hpPct'),
            staticIdx(2, 'heal')
        ],
        暖夜不会漫长: [
            staticIdx(1, 'hpPct')
        ],
        此时恰好: [
            staticIdx(1, 'effDef'),
            (tables) => {
                return {
                    title: '基于效果抵抗，提高治疗量[heal]%',
                    data: {
                        heal: ({ attr, calc }) => Math.min(tables[3], calc(attr.effDef) * tables[2] / 100)
                    }
                };
            }
        ],
        物穰: [
            keyIdx('释放战技与终结技时，治疗量提高[eHeal]%', { eHeal: 1, qHeal: 1 })
        ],
        等价交换: [],
        蕃息: [],
        棺的回响: [
            staticIdx(1, 'atkPct'),
            keyIdx('释放终结技后，速度提升[speed]', 'speed', 3)
        ],
        '嘿，我在这儿': [
            staticIdx(1, 'hpPct'),
            keyIdx('释放战技时，治疗量提高[heal]%', 'heal', 2)
        ],
        惊魂夜: [
            staticIdx(1, 'recharge'),
            (tables) => {
                return {
                    title: '提供治疗时，5层提供目标攻击力[atkPct]%',
                    data: {
                        atkPct: tables[3] * 5
                    }
                };
            }
        ],
        何物为真: [
            staticIdx(1, 'stance')
        ],
        唯有香如故: [
            staticIdx(1, 'stance'),
            (tables) => {
                return {
                    title: '终结技攻击敌方目标后，敌方目标受到的伤害提高[enemydmg]%',
                    sort: 9,
                    data: {
                        enemydmg: ({ attr, calc }) => calc(attr.stance) >= 150 ? (tables[2] + tables[3]) : tables[2]
                    }
                };
            }
        ],
        梦的蒙太奇: [
            staticIdx(1, 'speedPct')
        ],
        直到明天的明天: [
            staticIdx(1, 'heal'),
            keyIdx('造成的伤害提高[dmg]%', 'dmg', 2)
        ]
    };
}

return exports;})().default,"sr"));
Object.assign(weaponBuffs,loadWeaponDefinitions((()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1(staticIdx, keyIdx) {
    return {
        与行星相会: [],
        但战斗还未结束: [
            staticIdx(1, 'recharge')
        ],
        '舞！舞！舞！': [],
        记忆中的模样: [
            staticIdx(1, 'stance')
        ],
        调和: [
            keyIdx('进入战斗提高速度[speed]', 'speed', 1)
        ],
        轮契: [],
        过往未来: [],
        镂月裁云之意: [
            keyIdx('攻击Buff下提高攻击力[atkPct]%', 'atkPct', 1)
        ],
        齐颂: [
            keyIdx('进入战斗后，攻击力提高[atkPct]%', 'atkPct', 1)
        ],
        镜中故我: [
            staticIdx(1, 'stance'),
            keyIdx('装备者释放终结技后，我方全体造成伤害提高[dmg]%', 'dmg', 2)
        ],
        游戏尘寰: [
            staticIdx(1, 'cdmg')
        ],
        美梦小镇大冒险: [
            (tables) => {
                return {
                    title: '普攻、战技、终结技造成的伤害提高[_dmg]%',
                    data: {
                        _dmg: tables[1],
                        aDmg: tables[1],
                        eDmg: tables[1],
                        qDmg: tables[1]
                    }
                };
            }
        ],
        为了明日的旅途: [
            staticIdx(1, 'atkPct'),
            keyIdx('装备者释放终结技后，造成的伤害提高[dmg]%', 'dmg', 2)
        ],
        夜色流光溢彩: [
            (tables) => {
                return {
                    title: '5层【歌咏】使装备者能量恢复效率提高[recharge]%，释放终结技后，装备者攻击力提高[atkPct]%，我方全体造成伤害提高[dmg]%',
                    data: {
                        recharge: tables[1] * 5,
                        atkPct: tables[2],
                        dmg: tables[3]
                    }
                };
            }
        ],
        芳华待灼: [
            staticIdx(1, 'atkPct'),
            keyIdx('有两名及以上相同命途我方角色，暴击伤害提高[cdmg]%', 'cdmg', 2)
        ],
        如果时间是一朵花: [
            staticIdx(1, 'cdmg'),
            keyIdx('当装备者持有【谕示】时，我方全体目标的暴击伤害提高[cdmg]%', 'cdmg', 2)
        ],
        永远的迷境饭: [
            staticIdx(1, 'atkPct'),
            (tables) => {
                return {
                    title: '3层buff使装备者攻击力提高[atkPct]%',
                    data: {
                        atkPct: tables[2] * 3
                    }
                };
            }
        ],
        金血铭刻的时代: [staticIdx(1, 'atkPct')]
    };
}

return exports;})().default,"sr"));
Object.assign(weaponBuffs,loadWeaponDefinitions((()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1(staticIdx, keyIdx) {
    return {
        余生的第一天: [
            staticIdx(1, 'defPct')
        ],
        制胜的瞬间: [
            staticIdx(1, 'defPct'),
            staticIdx(2, 'effPct'),
            keyIdx('受到攻击时，防御力提高[defPct]%', 'defPct', 3)
        ],
        宇宙市场趋势: [
            staticIdx(1, 'defPct')
        ],
        开疆: [],
        戍御: [],
        我们是地火: [],
        朗道的选择: [],
        琥珀: [
            staticIdx(1, 'defPct'),
            keyIdx('生命值小于50%时，防御力提高[defPct]%', 'defPct', 2)
        ],
        记忆的质料: [
            staticIdx(1, 'effDef')
        ],
        '这就是我啦！': [
            staticIdx(1, 'defPct'),
            (tables) => {
                return {
                    title: '基于防御力提高终结技伤害值[qInc]',
                    data: {
                        qInc: ({ calc, attr }) => calc(attr.def) * tables[2] / 100
                    }
                };
            }
        ],
        她已闭上双眼: [
            staticIdx(1, 'hpPct'),
            staticIdx(2, 'recharge'),
            keyIdx('装备者生命降低时，使我方全体造成的伤害提高[dmg]%', 'dmg', 3)
        ],
        织造命运之线: [
            staticIdx(1, 'effDef'),
            (tables) => {
                return {
                    title: '基于防御力提高造成的伤害[dmg]%',
                    data: {
                        dmg: ({ calc, attr }) => Math.min(calc(attr.def) * tables[2] / 100, 32)
                    }
                };
            }
        ],
        命运从未公平: [
            staticIdx(1, 'defPct'),
            (tables) => {
                return {
                    title: '装备者为我方提供护盾时，暴击伤害提高[cdmg]%。追击命中敌方时，使敌方目标受到伤害提高[enemydmg]%',
                    data: {
                        cdmg: tables[2],
                        enemydmg: tables[4]
                    }
                };
            }
        ],
        两个人的演唱会: [
            staticIdx(1, 'defPct'),
            (tables) => {
                return {
                    title: '基于场上持有护盾的角色数目，提高装备者造成的伤害[dmg]%',
                    data: {
                        dmg: tables[2] * 4
                    }
                };
            }
        ],
        愿旅途永远坦然: [
            keyIdx('我方目标持有护盾时，造成的伤害提高[dmg]%', 'dmg', 2)
        ],
        纵然山河万程: [
            staticIdx(1, 'atkPct'),
            (tables) => {
                return {
                    title: '终结技后【卫戍】状态：造成的伤害提高[dmg]%',
                    data: {
                        dmg: tables[4]
                    }
                };
            }
        ]
    };
}

return exports;})().default,"sr"));
Object.assign(weaponBuffs,loadWeaponDefinitions((()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1(staticIdx, keyIdx) {
    return {
        于夜色中: [
            staticIdx(1, 'cpct'),
            (tables) => {
                return {
                    title: '[_count]层Buff提高普攻与战技伤害[aDmg]%，终结技的暴击伤害[qCdmg]%',
                    data: {
                        _count: ({ attr, calc }) => Math.min(6, Math.floor((calc(attr.speed) - 100) / 10)) + '',
                        aDmg: ({ attr, calc }) => Math.min(6, Math.floor((calc(attr.speed) - 100) / 10)) * tables[2],
                        eDmg: ({ attr, calc }) => Math.min(6, Math.floor((calc(attr.speed) - 100) / 10)) * tables[2],
                        qCdmg: ({ attr, calc }) => Math.min(6, Math.floor((calc(attr.speed) - 100) / 10)) * tables[3]
                    }
                };
            }
        ],
        唯有沉默: [
            staticIdx(1, 'atkPct'),
            keyIdx('场上目标小于等于2时，暴击率提高[cpct]%', 'cpct', 2)
        ],
        如泥酣眠: [
            staticIdx(1, 'cdmg')
        ],
        星海巡航: [
            staticIdx(1, 'cpct'),
            keyIdx('对生命值小于50%的敌人暴击率提高[cpct]%，消灭敌方目标后，攻击力提高[atkPct]%', { cpct: 2, atkPct: 3 })
        ],
        春水初生: [
            keyIdx('进入战斗提高速度[speedPct]%，伤害[dmg]%', { speedPct: 1, dmg: 2 })
        ],
        '点个关注吧！': [(tables) => {
                return {
                    title: '满能量提高普攻和战技伤害[aDmg]%',
                    data: {
                        aDmg: tables[1] * 2,
                        eDmg: tables[1] * 2
                    }
                };
            }],
        相抗: [
            keyIdx('消灭敌方目标后，速度提高[speedPct]%', 'speedPct', 1)
        ],
        离弦: [
            keyIdx('消灭敌方目标后，攻击力[atkPct]%', 'atkPct', 1)
        ],
        论剑: [(tables) => {
                return {
                    title: '连续击中5次同一个目标，伤害提高[dmg]%',
                    data: {
                        dmg: tables[1] * 5
                    }
                };
            }],
        重返幽冥: [
            staticIdx(1, 'cpct')
        ],
        锋镝: [
            keyIdx('战斗开始时暴击率提高[cpct]%', 'cpct', 1)
        ],
        '烦恼着，幸福着': [
            staticIdx(1, 'cpct'),
            keyIdx('追加攻击造成的伤害提高[tDmg]%', 'tDmg', 2),
            (tables) => {
                return {
                    title: '2层Buff时，造成的暴击伤害提高[cdmg]%',
                    data: {
                        cdmg: tables[3] * 2
                    }
                };
            }
        ],
        纯粹思维的洗礼: [
            staticIdx(1, 'cdmg'),
            (tables) => {
                return {
                    title: '3层Buff提升装备者暴击伤害[cdmg]%',
                    data: {
                        cdmg: tables[2] * 3
                    }
                };
            },
            keyIdx('释放终结技后，造成伤害提高[dmg]%，追加攻击无视目标防御力[tIgnore]%', { dmg: 3, tIgnore: 4 })
        ],
        最后的赢家: [
            staticIdx(1, 'atkPct'),
            (tables) => {
                return {
                    title: '4层Buff提升装备者暴击伤害[cdmg]%',
                    data: {
                        cdmg: tables[2] * 4
                    }
                };
            }
        ],
        驶向第二次生命: [
            staticIdx(1, 'stance'),
            keyIdx('造成的击破伤害无视目标[breakIgnore]%的防御力', 'breakIgnore', 2),
            (tables) => {
                return {
                    title: '速度提高[speedPct]%',
                    check: ({ attr }) => attr.stance >= 150,
                    data: {
                        speedPct: tables[3]
                    }
                };
            }
        ],
        黑夜如影随行: [
            staticIdx(1, 'stance'),
            (tables) => {
                return {
                    title: '进入战斗或造成击破伤害后，速度提高[speedPct]%',
                    data: {
                        speedPct: tables[2]
                    }
                };
            }
        ],
        '我将，巡征追猎': [
            staticIdx(1, 'cpct'),
            (tables) => {
                return {
                    title: '2层Buff使装备者造成的终结技伤害无视目标[qIgnore]%防御力',
                    data: {
                        qIgnore: tables[2] * 2
                    }
                };
            }
        ],
        理想燃烧的地狱: [
            staticIdx(1, 'cpct'),
            (tables) => {
                return {
                    title: '攻击力提高[atkPct]%',
                    data: {
                        atkPct: tables[2] + tables[3] * 4
                    }
                };
            }
        ],
        于那终点再见: [
            staticIdx(1, "cdmg"),
            keyIdx("装备者战技和追加攻击造成的伤害提高[eDmg]%", { eDmg: 2, tDmg: 2 })
        ],
        一场谎言的终幕: [
            staticIdx(1, "cpct"),
            keyIdx("战斗开始时或施放4.0次追加攻击后攻击力提高[atkPct]%,敌方受到伤害提高[enemydmg]%", { atkPct: 2, enemydmg: 3 })
        ],
        向着地平线的终点: [
            staticIdx(1, "atkPct"),
            (tables) => {
                return {
                    title: '使装备者的暴击伤害提高[cdmg]%',
                    data: {
                        cdmg: tables[2] * 10
                    }
                };
            }
        ]
    };
}

return exports;})().default,"sr"));
Object.assign(weaponBuffs,loadWeaponDefinitions((()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1(staticIdx, keyIdx) {
    return {
        '「我」的诞生': [
            (tables) => {
                return {
                    title: '敌方目标当前生命值百分比小于等于50%追加伤害提高[tDmg]%',
                    data: {
                        tDmg: tables[1] + tables[2]
                    }
                };
            }
        ],
        今日亦是和平的一日: (tables) => {
            return {
                title: '根据能量上限提高数据',
                data: {
                    dmg: ({ attr }) => tables[1] * Math.min(attr.sp, 160)
                }
            };
        },
        别让世界静下来: [
            staticIdx(1, 'recharge'),
            keyIdx('终结技造成的伤害提高[qDmg]%', 'qDmg', 2)
        ],
        天才们的休憩: [
            staticIdx(1, 'atkPct'),
            keyIdx('消灭敌方目标后爆伤提高[cdmg]%', 'cdmg', 2)
        ],
        拂晓之前: [
            staticIdx(1, 'cdmg'),
            keyIdx('战技和终结技造成的伤害提高[eDmg]%，追加攻击造成的伤害提高[tDmg]%', { eDmg: 2, qDmg: 2, tDmg: 3 })
        ],
        早餐的仪式感: [
            staticIdx(1, 'dmg'),
            (tables) => {
                return {
                    title: '3层Buff下提高攻击力[atkPct]%',
                    data: {
                        atkPct: tables[2] * 3
                    }
                };
            }
        ],
        智库: [
            keyIdx('使装备者终结技造成的伤害提高[qDmg]%', 'qDmg', 1)
        ],
        灵钥: [],
        睿见: [
            keyIdx('施放终结技时攻击力提高[atkPct]%', 'atkPct', 1)
        ],
        银河铁道之夜: [
            (tables) => {
                return {
                    title: '3个敌方目标提高攻击力[atkPct]%，弱点击破时伤害提高[dmg]%',
                    data: {
                        atkPct: tables[1] * 3,
                        dmg: tables[2]
                    }
                };
            }
        ],
        '片刻，留在眼底': [
            staticIdx(1, 'cdmg'),
            (tables) => {
                return {
                    title: '根据装备者的能量上限提高伤害[qDmg]%',
                    data: {
                        qDmg: ({ attr }) => tables[2] * Math.min(attr.sp, 180)
                    }
                };
            }
        ],
        银河沦陷日: [
            staticIdx(1, 'atkPct'),
            keyIdx('提高暴击伤害[cdmg]%', 'cdmg', 2)
        ],
        谐乐静默之后: [
            staticIdx(1, 'stance'),
            keyIdx('施放终结技后速度提高[speedPct]%', 'speedPct', 2)
        ],
        不息的演算: [
            staticIdx(1, 'atkPct'),
            (tables) => {
                return {
                    title: '施放攻击击中五名敌方目标，使攻击力额外提高[atkPct]%，速度提升[speedPct]%',
                    data: {
                        atkPct: tables[2] * 5,
                        speedPct: tables[3]
                    }
                };
            }
        ],
        偏偏希望无价: [
            staticIdx(1, 'cpct'),
            (tables) => {
                return {
                    check: ({ attr }) => attr.cdmg > 120,
                    title: '基于暴击伤害，追加攻击伤害提高[tDmg]%，终结技或追加攻击伤害无视[qIgnore]%防御',
                    sort: 9,
                    data: {
                        tDmg: ({ attr }) => Math.min(Math.floor((attr.cdmg - 120) / 20), 4) * tables[2],
                        qIgnore: tables[3],
                        tIgnore: tables[3]
                    }
                };
            }
        ],
        '忍法帖•缭乱破魔': [
            staticIdx(1, 'stance')
        ],
        向着不可追问处: [
            staticIdx(1, 'cpct'),
            keyIdx('战技和终结技造成的伤害提高[eDmg]%', { eDmg: 2, qDmg: 2 })
        ],
        宇宙大生意: [
            staticIdx(1, 'atkPct'),
            (tables) => {
                return {
                    title: '造成的伤害提高[dmg]%',
                    data: {
                        dmg: tables[2] * 7
                    }
                };
            }
        ],
        生命当付之一炬: [
            keyIdx('装备者对其造成的伤害提高[dmg]%，装备者使其防御力降低[enemyDef]%', { dmg: 1, enemyDef: 2 })
        ],
        氤氲麦香的梦: [
            staticIdx(1, "cpct"),
            keyIdx("装备者终结技造成的伤害提高[qDmg]%,追加攻击造成的伤害提高[tDmg]%", { qDmg: 2, tDmg: 2 })
        ],
        当一颗星照亮夜空: [
            keyIdx("无视目标[ignore]%的防御力", { ignore: 2 }),
            (tables) => {
                return {
                    title: '3层助战技伤害提高[tDmg]%，终结技伤害提高[qDmg]%',
                    data: {
                        tDmg: tables[2] * 3,
                        qDmg: tables[3] * 3
                    }
                };
            }
        ],
        星火悄然闪耀: [
            staticIdx(1, "cpct"),
            keyIdx("无视目标[ignore]%的防御力，战技伤害提高[eDmg]%", { ignore: 2, eDmg: 2 })
        ]
    };
}

return exports;})().default,"sr"));
Object.assign(weaponBuffs,loadWeaponDefinitions((()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1(staticIdx, keyIdx) {
    return {
        花花世界迷人眼: [
            staticIdx(1, 'cdmg'),
            (tables) => {
                return {
                    title: '无视对方防御力[ignore]%，欢愉度提升[joy]%',
                    data: {
                        ignore: tables[2] * 4,
                        joy: tables[3]
                    }
                };
            }
        ],
        当她决定看见: [
            staticIdx(1, 'speedPct'),
            keyIdx('【上上签】增加[cpct]%暴击率,增加[cdmg]%暴击伤害', {
                cpct: 2, cdmg: 3
            })
        ],
        今日好手气: [
            staticIdx(1, 'cpct')
        ],
        菇菇嘎嘎历险记: [
            staticIdx(1, 'joy'),
            keyIdx('敌方全体受到的欢愉伤害提高[xeEnemydmg]%', 'xeEnemydmg', 2)
        ],
        '未来，有我们一起': [
            staticIdx(1, 'cdmg'),
            keyIdx('施放终结技后，使我方全体欢愉度提高[joy]%', 'joy', 2)
        ],
        欢愉满溢祝福: [
            staticIdx(1, 'atkPct')
        ],
        欢迎来到银河城: [
            staticIdx(1, 'speedPct'),
            keyIdx('造成的欢愉伤害无视目标[xeIgnore]%的防御力。装备者对自身单体施放终结技时，获得[punchline]点【笑点】', {
                xeIgnore: 2, punchline: 3
            })
        ],
        邂逅于下一个花季: [
            staticIdx(1, 'cdmg'),
            (tables) => {
                return {
                    title: '能量恢复效率提高[recharge]%',
                    data: {
                        recharge: ({ attr }) => tables[2] + (Math.min(attr.sp, 480) - 120) / 10 * 0.3
                    }
                };
            },
            keyIdx('施放欢愉技时，使敌方目标受到的伤害提高[enemydmg]%', 'enemydmg', 3)
        ],
        放个短假: [
            staticIdx(1, 'joyPct'),
            keyIdx('施放欢愉技期间无视目标[xeIgnore]%的防御力', 'xeIgnore', 2)
        ],
        向浪花掷下盛夏: [
            staticIdx(1, 'cpct'),
            keyIdx('装备者的速度提高[speedPct]%，欢愉度提高[joyPct]%', {
                speedPct: 2, joyPct: 3
            })
        ],
        献给明日的色彩: [
            staticIdx(1, 'defPct'),
            keyIdx('敌方全体受到的伤害提高[enemydmg]%', {
                enemydmg: 2
            })
        ]
    };
}

return exports;})().default,"sr"));
Object.assign(weaponBuffs,loadWeaponDefinitions((()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1(staticIdx, keyIdx) {
    return {
        乐圮: [
            keyIdx('对生命值大于50%的伤害提高[dmg]%', 'dmg', 1)
        ],
        俱殁: [
            keyIdx('生命值大于80%时提高暴击率[cpct]%', 'cpct', 1)
        ],
        在蓝天下: [
            staticIdx(1, 'atkPct'),
            keyIdx('消灭敌方目标后暴击率提高[cpct]%', 'cpct', 2)
        ],
        天倾: [
            keyIdx('普攻和战技伤害提高[aDmg]%', { aDmg: 1, eDmg: 1 })
        ],
        无可取代的东西: [
            staticIdx(1, 'atkPct'),
            keyIdx('受到攻击时提高造成伤害[dmg]%', 'dmg', 3)
        ],
        无处可逃: [
            staticIdx(1, 'atkPct')
        ],
        '汪！散步时间！': [
            staticIdx(1, 'atkPct'),
            keyIdx('对烧灼或裂伤状态的敌人伤害提高[dmg]%', 'dmg', 2)
        ],
        秘密誓心: [
            staticIdx(1, 'dmg'),
            keyIdx('对生命百分比高于角色的敌人的伤害提高[dmg]%', 'dmg', 2)
        ],
        记一位星神的陨落: [(tables) => {
                return {
                    title: '4层Buff提高攻击力[atkPct]%，击破弱点后造成的伤害提高[dmg]%',
                    data: {
                        atkPct: tables[1] * 4,
                        dmg: tables[2]
                    }
                };
            }],
        鼹鼠党欢迎你: [(tables) => {
                return {
                    title: '3层Buff提高攻击力[atkPct]%',
                    data: {
                        atkPct: tables[1] * 3
                    }
                };
            }],
        到不了的彼岸: [
            staticIdx(1, 'cpct'),
            staticIdx(2, 'hpPct'),
            keyIdx('受到攻击或装备者消耗自身生命值，伤害提高[dmg]%', 'dmg', 3)
        ],
        比阳光更明亮的: [
            staticIdx(1, 'cpct'),
            (tables) => {
                return {
                    title: '2层Buff提高攻击力[atkPct]%，能量恢复效率提高[recharge]%',
                    data: {
                        atkPct: tables[2] * 2,
                        recharge: tables[3] * 2
                    }
                };
            }
        ],
        此身为剑: [
            staticIdx(1, 'cdmg'),
            (tables) => {
                return {
                    title: '3层Buff使装备者下一次攻击造成的伤害提高[dmg]%，使该次攻击无视目标[ignore]%的防御力',
                    data: {
                        dmg: tables[2] * 3,
                        ignore: tables[3]
                    }
                };
            }
        ],
        在火的远处: [
            keyIdx('装备者造成的伤害提高[dmg]%', 'dmg', 1)
        ],
        铭记于心的约定: [
            staticIdx(1, 'stance'),
            keyIdx('释放终结技时，暴击率提高[cpct]%', 'cpct', 2)
        ],
        梦应归于何处: [
            staticIdx(1, 'stance'),
            keyIdx('敌方受到装备者造成的击破伤害提高[breakEnemydmg]%', 'breakEnemydmg', 2)
        ],
        落日时起舞: [
            staticIdx(1, 'cdmg'),
            (tables) => {
                return {
                    title: '2层Buff使追加攻击造成的伤害提高[tDmg]%',
                    data: {
                        tDmg: tables[2] * 2
                    }
                };
            }
        ],
        '忍事录•音律狩猎': [
            staticIdx(1, 'hpPct'),
            keyIdx('损失或回复自身生命值，暴击伤害提高[cdmg]%', 'cdmg', 2)
        ],
        '血火啊，燃烧前路': [
            staticIdx(1, 'hpPct'),
            (tables) => {
                return {
                    title: '施放战技或终结技时，使本次攻击造成的伤害提高[eDmg]%',
                    data: {
                        eDmg: ({ attr, calc }) => {
                            return calc(attr.hp) * tables[3] / 100 > 500 ? tables[5] + tables[4] : tables[4];
                        },
                        qDmg: ({ attr, calc }) => {
                            return calc(attr.hp) * tables[3] / 100 > 500 ? tables[5] + tables[4] : tables[4];
                        }
                    }
                };
            }
        ],
        黎明恰如此燃烧: [
            staticIdx(1, 'speed'),
            keyIdx('造成伤害时无视目标[ignore]%的防御力', 'ignore', 2),
            (tables) => {
                return {
                    title: '装备者施放终结技后，持有【烈阳】时，装备者造成的伤害提高[dmg]%',
                    data: {
                        dmg: tables[3]
                    }
                };
            }
        ],
        没有回报的加冕: [
            staticIdx(1, 'cdmg'),
            (tables) => {
                return {
                    title: '攻击力提高[atkPct]%',
                    data: {
                        atkPct: tables[2] + tables[3]
                    }
                };
            }
        ],
        一行往日的血: [
            keyIdx('暴击率提高[cpct]%,战技伤害提高[eDmg]%,终结技伤害提高[qDmg]%', {
                cpct: 1, eDmg: 2, qDmg: 2
            })
        ],
        所见即我: [
            keyIdx('装备者的攻击力提高[atkPct]%，能量恢复效率提高[recharge]%', {
                atkPct: 1, recharge: 2
            }),
            (tables) => {
                return {
                    title: '终结技伤害提高[qDmg]%，【王之娱乐】我方全体暴击伤害提高[cdmg]%',
                    data: {
                        qDmg: ({ attr }) => {
                            return attr.sp >= 360 ? tables[4] : tables[3] * attr.sp;
                        },
                        cdmg: tables[5]
                    }
                };
            }
        ]
    };
}

return exports;})().default,"sr"));
Object.assign(weaponBuffs,loadWeaponDefinitions((()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1(staticIdx, keyIdx) {
    return {
        以世界之名: [
            keyIdx('对陷入负面效果的敌人伤害提高[dmg]%，释放战技的功力提高[atkPct]%', { dmg: 1, atkPct: 3 })
        ],
        决心如汗珠般闪耀: [
            keyIdx('攻陷状态敌方防御力降低[enemyDef]%', 'enemyDef', 2)
        ],
        匿影: [],
        后会有期: [],
        幽邃: [
            keyIdx('使装备者的效果命中提高[effPct]%', 'effPct', 1)
        ],
        延长记号: [
            staticIdx(1, 'stance'),
            keyIdx('对触电或风化状态的敌方目标伤害提高[dmg]%', 'dmg', 2)
        ],
        晚安与睡颜: [
            (tables) => {
                return {
                    title: '3层Buff提高伤害[dmg]%',
                    data: {
                        dmg: tables[1] * 3
                    }
                };
            }
        ],
        渊环: [
            keyIdx('对减速状态的目标伤害提高[dmg]%', 'dmg', 1)
        ],
        猎物的视线: [
            staticIdx(1, 'effPct')
        ],
        新手任务开始前: [
            staticIdx(1, 'effPct')
        ],
        雨一直下: [
            staticIdx(1, 'effPct'),
            keyIdx('对大于3个负面效果的敌人提升暴击率[cpct]%', 'cpct', 2),
            keyIdx('以太编码提升伤害[dmg]%', 'dmg', 3)
        ],
        只需等待: [
            staticIdx(1, 'dmg'),
            (tables) => {
                return {
                    title: '释放攻击3层Buff提高速度[speedPct]%',
                    data: {
                        speedPct: tables[2] * 3
                    }
                };
            }
        ],
        孤独的疗愈: [
            staticIdx(1, 'stance'),
            keyIdx('装备者造成的持续伤害提高[dotDmg]%', 'dotDmg', 2)
        ],
        重塑时光之忆: [
            staticIdx(1, 'effPct'),
            (tables) => {
                return {
                    title: '4层Buff提高攻击力[atkPct]%，造成的持续伤害无视目标[dotIgnore]%的防御力',
                    data: {
                        atkPct: tables[2] * 4,
                        dotIgnore: tables[3] * 4
                    }
                };
            }
        ],
        好戏开演: [
            (tables) => {
                return {
                    title: '3层Buff提高造成的伤害[dmg]%',
                    data: {
                        dmg: tables[1] * 3
                    }
                };
            },
            (tables) => {
                return {
                    check: ({ attr, calc }) => calc(attr.effPct) >= 80,
                    title: '攻击力提高[atkPct]%',
                    data: {
                        atkPct: tables[2]
                    }
                };
            }
        ],
        行于流逝的岸: [
            staticIdx(1, 'cdmg'),
            (tables) => {
                return {
                    title: '造成的伤害提高[dmg]%，终结技伤害额外提高[qDmg]%',
                    data: {
                        dmg: tables[2],
                        qDmg: tables[3]
                    }
                };
            }
        ],
        无边曼舞: [
            staticIdx(1, 'cpct'),
            keyIdx('对处于防御降低或减速状态下的敌人暴击伤害提高[cdmg]%', 'cdmg', 2)
        ],
        那无数个春天: [
            staticIdx(1, 'effPct'),
            (tables) => {
                return {
                    title: '【穷寇】状态下，敌方目标受到的伤害提高[enemydmg]%',
                    data: {
                        enemydmg: tables[2] + tables[3]
                    }
                };
            }
        ],
        长路终有归途: [
            staticIdx(1, 'stance'),
            keyIdx('【焚灼】状态下，敌方目标受到的击破伤害提高[breakEnemydmg]%', 'breakEnemydmg', 2)
        ],
        谎言在风中飘扬: [
            staticIdx(1, 'speedPct'),
            keyIdx('装备者施放攻击后，有120%的基础概率使敌方每个单体目标的防御力降低[enemyDef]%', 'enemyDef', 2),
            (tables) => {
                return {
                    title: '若装备者的速度大于等于170，有120%的基础概率使敌方每个单体目标的防御力降低[enemyDef]%',
                    check: ({ attr, calc }) => calc(attr.speed) >= 170,
                    data: {
                        enemyDef: tables[3]
                    }
                };
            }
        ],
        海洋为何而歌: [
            staticIdx(1, 'effPct'),
            (tables) => {
                return {
                    title: '【魂迷】状态下，有6个装备者施加的负面效果，受到的持续伤害提高[dotEnemydmg]%，受到我方目标攻击时，使攻击者速度提高[speedPct]%，持续3回合',
                    data: {
                        dotEnemydmg: tables[2] * 6,
                        speedPct: tables[3]
                    }
                };
            }
        ],
        假日浴场大冒险: [
            staticIdx(1, 'dmg'),
            keyIdx('使受到攻击的目标陷入易伤状态，受到的伤害提高[enemydmg]%', 'enemydmg', 2)
        ],
        勿忘她的火焰: [
            staticIdx(1, 'stance'),
            keyIdx('装备者造成的击破伤害提高[breakEnemydmg]%', 'breakEnemydmg', 2)
        ],
        灼尽炼狱的新骸: [
            staticIdx(1, 'hpPct'),
            (tables) => {
                return {
                    title: '【炼狱】状态下目标受到的暴击伤害提高[cdmgName]%，受到来自装备者的暴击伤害额外提高[cdmgName]%',
                    data: {
                        cdmgName: tables[2],
                        cdmg: tables[2] * 2
                    }
                };
            }
        ]
    };
}

return exports;})().default,"sr"));
Object.assign(weaponBuffs,loadWeaponDefinitions((()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1(staticIdx, keyIdx) {
    return {
        溯忆: [(tables) => {
                return {
                    title: '造成的伤害提高[dmg]%',
                    data: {
                        dmg: tables[1] * 4
                    }
                };
            }],
        焚影: [],
        '多流汗，少流泪': [
            staticIdx(1, 'cpct'),
            keyIdx('造成的伤害提高[dmg]%', 'dmg', 2)
        ],
        天才们的问候: [
            staticIdx(1, 'atkPct'),
            keyIdx('普攻伤害提高[aDmg]%', 'aDmg', 2)
        ],
        胜利只在朝夕间: [
            staticIdx(1, 'cdmg'),
            keyIdx('造成的伤害提高[dmg]%', 'dmg', 2)
        ],
        将光阴织成黄金: [
            staticIdx(1, 'speed'),
            (tables) => {
                return {
                    title: '暴击伤害提高[cdmg]%，普攻伤害提高[aDmg]%',
                    data: {
                        cdmg: tables[2] * 6,
                        aDmg: tables[3] * 6
                    }
                };
            }
        ],
        记忆永不落幕: [
            staticIdx(1, 'speedPct'),
            keyIdx('装备者施放战技后，使我方全体造成的伤害提高[dmg]%', 'dmg', 2)
        ],
        '让告别，更美一些': [
            staticIdx(1, 'hpPct'),
            keyIdx('无视目标[ignore]%的防御力', 'ignore', 2)
        ],
        愿虹光永驻天空: [
            staticIdx(1, 'speedPct'),
            keyIdx('装备者的忆灵施放忆灵技时，使敌方全体受到的伤害提高[enemydmg]%', 'enemydmg', 4)
        ],
        故事的下一页: [
            staticIdx(1, "hpPct"),
            (tables) => {
                return {
                    check: ({ params }) => params.Memosprite,
                    title: "装备者的忆灵攻击后，使装备者与忆灵的治疗量提高[heal]%",
                    data: {
                        heal: tables[2]
                    }
                };
            }
        ],
        花儿不会忘记: [
            staticIdx(1, "cdmg"),
            (tables) => {
                return {
                    check: ({ params }) => params.Memosprite,
                    title: "装备者忆灵造成的暴击伤害额外提高[meCdmg]%",
                    data: {
                        meCdmg: tables[2]
                    }
                };
            }
        ],
        致长夜的星光: [
            staticIdx(1, "hpPct"),
            (tables) => {
                return {
                    check: ({ params }) => params.Memosprite,
                    title: "【夜色】状态：忆灵无视[meIgnore]%防御，自身与忆灵增伤[dmg]%",
                    data: {
                        meIgnore: tables[2],
                        mtIgnore: tables[2],
                        dmg: tables[3]
                    }
                };
            }
        ],
        爱如此刻永恒: [
            staticIdx(1, "speedPct"),
            (tables) => {
                return {
                    check: ({ params }) => params.Memosprite,
                    title: "敌方全体受到的伤害提高[enemydmg]%，我方全体的暴击伤害提高[cdmg]%",
                    data: {
                        enemydmg: tables[3] * (100 + tables[4]) / 100,
                        cdmg: tables[2] * (100 + tables[4]) / 100
                    }
                };
            }
        ],
        飞向粉色的明天: [
            staticIdx(1, "cdmg"),
            (tables) => {
                return {
                    check: ({ charId }) => [8007, 8008].includes(charId * 1),
                    title: "开拓者•记忆装备时：我方全体增伤[dmg]%，强化普攻增伤[a2Dmg]%",
                    data: {
                        dmg: tables[2],
                        a2Dmg: tables[3]
                    }
                };
            }
        ],
        你将起身歌唱: [
            staticIdx(1, "hpPct")
        ]
    };
}

return exports;})().default,"sr"));
