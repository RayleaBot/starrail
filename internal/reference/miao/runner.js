// Entry points the plugin calls: runBuild (伤害计算), runChange (面板换装) and
// runScore (遗器评分), each on the pinned miao-plugin modules.

// miaoKey maps the panel's attribute keys to miao's, which names two elements
// differently.
function miaoKey(key) {
  return { physical: 'phy', lightning: 'elec' }[key] || key;
}

// characterObject stands in for upstream's Character for one catalog record.
function characterObject(record) {
  const data = record.data;
  return {
    id: Number(record.id),
    name: record.name,
    game: record.game,
    detail: data,
    elem: record.element,
    weapon: data.weapon,
    weaponTypeName: data.weapon,
    sp: data.sp,
    isElem: elem => Format.sameElem(elem, record.element),
    getCalcRule: () => characterRule,
    getLvAttr(level, promote) {
      const item = data.attr[promote];
      if (!item) throw Error('character.promote');
      return Object.fromEntries(Object.entries(item.attrs).map(([key, value]) => [key, value * 1 + (item.grow?.[key] || 0) * (level - 1)]));
    },
  };
}

// artifactObject stands in for upstream's ArtisSet: the set bonuses reached,
// forEach over the relics and eachArtisSet over the bonuses.
function artifactObject(gear) {
  const adapt = stat => ({ ...stat, key: miaoKey(stat.key) });
  const items = gear.map(g => ({ main: adapt(g.main), attrs: g.sub.map(adapt) }));
  const counts = {};
  for (const piece of gear) {
    if (piece.set_name) counts[piece.set_name] = (counts[piece.set_name] || 0) + 1;
  }
  const sets = {};
  for (const [name, count] of Object.entries(counts)) {
    if (count >= 4) sets[name] = 4;
    else if (count >= 2) sets[name] = 2;
  }
  const result = { ...sets };
  Object.defineProperties(result, {
    forEach: { value: fn => items.forEach(fn) },
    eachArtisSet: {
      value: fn => {
        for (const [name, n] of Object.entries(sets)) {
          fn({ name }, 2);
          if (n >= 4) fn({ name }, 4);
        }
      },
    },
  });
  return result;
}

// levels reads each skill's tables at the profile's skill levels. Numbered
// keys (a2, e1…) share the level of their base skill; tables marked isSame
// keep their first value.
const skillKeys = ['a', 'a2', 'e', 'e1', 'e2', 'q', 'q2', 'q3', 't', 't2', 'xe', 'xe2', 'me', 'me2', 'mt', 'mt1', 'mt2'];
function levels(profile) {
  const out = { talentLevel: profile.talent };
  for (const key of skillKeys) {
    const level = profile.talent[key.replace(/[123]$/, '')];
    out[key] = {};
    for (const table of Object.values(profile.char.detail.talent?.[key]?.tables || {})) {
      if (!Number.isInteger(level) || level < 1) throw Error('talent.' + key);
      out[key][table.name] = table.isSame ? table.values[0] : table.values[level - 1];
    }
  }
  return out;
}

// combatBuffs lists the character, light cone, set and user bonuses in
// upstream's order.
function combatBuffs(profile, conditions) {
  const list = conditions?.disable_character ? [] : lodash.cloneDeep(characterRule.buffs || []);
  const weapon = Weapon.get(profile.weapon.id);
  // Miao's light cone and relic adapters append only combat effects. Permanent
  // effects are already part of the static panel and must not be counted twice.
  if (weapon && !conditions?.disable_weapon) list.push(...weapon.getWeaponAffixBuffs(profile.weapon.affix, false));
  if (!conditions?.disable_equipment) {
    profile.artis.eachArtisSet((set, count) => {
      for (const buff of ArtifactSet.getArtisSetBuff(set.name, count)) {
        if (buff && !buff.isStatic) list.push({ ...lodash.cloneDeep(buff), title: set.name + count + '：' + buff.title });
      }
    });
  }
  if (conditions) {
    for (const source of [{ name: '用户填写', bonuses: conditions.bonuses }, ...(conditions.team || [])]) {
      for (const [key, value] of Object.entries(source.bonuses || {})) {
        if (!value) continue;
        const target = { enemyDmg: 'enemydmg', resistance: 'kx' }[key] || key;
        list.push({ title: `自定义·${source.name}：${key} +${value}`, sort: 1, data: { [target]: value } });
      }
    }
    // Upstream counts enemies at no resistance.
    if (conditions.enemy_resistance != null) {
      list.push({ title: `自定义敌人抗性 ${conditions.enemy_resistance}%`, sort: -1, data: { kx: -conditions.enemy_resistance } });
    }
  }
  const reactions = { vaporize: '蒸发', melt: '融化', swirl: '扩散', aggravate: '超激化', spread: '蔓激化' };
  return list.map((buff, index) => {
    if (typeof buff === 'string') {
      const key = buff;
      const extra = ['aggravate', 'spread'].includes(key) ? `，伤害值提升[_${key}num]` : '';
      buff = { title: `元素精通：${reactions[key] || key}伤害提高[_${key}]%` + extra, mastery: key, sort: 9 };
    }
    return { ...buff, sort: buff.sort ?? 1, _index: index };
  }).sort((a, b) => a.sort - b.sort || a._index - b._index);
}

// legalPromotions lists the ascensions a level allows; boundary levels allow
// two.
const promotionSteps = [1, 20, 30, 40, 50, 60, 70, 80];
function legalPromotions(level) {
  return promotionSteps.slice(0, -1).flatMap((left, index) => level >= left && level <= promotionSteps[index + 1] ? [index] : []);
}

// inferPromotions picks the ascensions whose base stats match the official
// panel, as the official answer does not name them.
function inferPromotions(profile, observed) {
  const characterPromotions = profile.promote == null ? legalPromotions(profile.level) : [profile.promote];
  const weaponPromotions = !profile.weapon.id ? [0] : (profile.weapon.promote == null ? legalPromotions(profile.weapon.level) : [profile.weapon.promote]);
  let best;
  for (const promote of characterPromotions) {
    for (const weaponPromote of weaponPromotions) {
      try {
        const test = { ...profile, promote, weapon: { ...profile.weapon, promote: weaponPromote } };
        const attrs = new Attr(test).calc();
        let score = 0;
        let count = 0;
        for (const key of ['hpBase', 'atkBase', 'defBase']) {
          if (!Number.isFinite(observed[key])) continue;
          score += Math.abs(attrs[key] - observed[key]) / Math.max(1, observed[key]);
          count++;
        }
        if (!count) throw Error('attributes.base_missing');
        if (!best || score < best.score) best = { score, promote, weaponPromote, attrs };
      } catch (e) {
        // Another legitimate promotion may fit the same boundary level.
      }
    }
  }
  if (!best || !Number.isFinite(best.score) || best.score > 0.03) throw Error('attributes.reference_mismatch');
  profile.promote = best.promote;
  profile.weapon.promote = best.weaponPromote;
  return best.attrs;
}

// ruleValue reads a rule setting that may be a function of the meta.
function ruleValue(value, meta, fallback) {
  return typeof value === 'function' ? value(meta) : value || fallback;
}

// calculateProfile runs the character's damage details as upstream's
// ProfileDmg.calcData, with the static panel anchored to the official one.
function calculateProfile(profile, observed, staticOriginal, enemyLevel, conditions) {
  const staticNow = new Attr(profile).calc();
  const anchored = { ...staticNow, staticAttr: staticNow.staticAttr };
  for (const [key, value] of Object.entries(observed)) {
    if (Number.isFinite(value) && Number.isFinite(staticOriginal[key]) && Number.isFinite(staticNow[key])) {
      anchored[key] = value + staticNow[key] - staticOriginal[key];
    }
  }
  const talent = levels(profile);
  // Rules name traces by the last three digits of their point ID.
  const trees = {};
  for (const id of profile.trees) {
    const match = /1?\d{4}(\d{3})/.exec(id);
    if (match) trees[match[1]] = true;
  }
  const meta = { characterName: profile.char.name, level: profile.level, cons: profile.cons, talent, trees, weapon: profile.weapon };
  const originalAttr = DmgAttr.getAttr({ attr: anchored, weapon: profile.weapon, char: profile.char, game: currentGame });
  const buffs = combatBuffs(profile, conditions);
  const defParams = ruleValue(characterRule.defParams, meta, {});
  // miao's single-damage mode (group ranks) takes the first detail computed
  // among those named by defDmgKey, else the one at defDmgIdx, else the first.
  const defKey = ruleValue(characterRule.defDmgKey, meta, '');
  const defIdx = ruleValue(characterRule.defDmgIdx, meta, -1);
  // Custom defence bonuses stay within upstream's 0–100%.
  const clampDefence = conditions && [conditions.bonuses, ...(conditions.team || []).map(t => t.bonuses)].some(b => b && (b.ignore || b.enemyDef));
  const results = [];
  for (const [index, configured] of (characterRule.details || []).entries()) {
    const isDefault = defKey ? configured.dmgKey === defKey : index === (defIdx > -1 ? defIdx : 0);
    let detail = configured;
    if (typeof detail === 'function') {
      const { attr } = DmgAttr.calcAttr({ originalAttr, buffs, artis: profile.artis, meta, game: currentGame });
      detail = detail({ ...DmgAttr.getDs(attr, meta), talent, attr, profile });
    }
    if (!detail || detail.isStatic || detail.cons && meta.cons < detail.cons) continue;
    const params = lodash.merge({}, defParams, ruleValue(detail.params, meta, {}));
    const { attr, msg } = DmgAttr.calcAttr({ originalAttr, buffs, artis: profile.artis, meta, params, talent: detail.talent || '', game: currentGame });
    if (clampDefence) {
      attr.enemy.def = Math.max(0, Math.min(100, attr.enemy.def));
      attr.enemy.ignore = Math.max(0, Math.min(100, attr.enemy.ignore));
    }
    const ds = lodash.merge({ talent }, DmgAttr.getDs(attr, meta, params));
    ds.artis = profile.artis;
    if (detail.check && !detail.check(ds)) continue;
    if (!detail.dmg) continue;
    const fn = DmgCalc.getDmgFn({ ds, attr, level: profile.level, enemyLv: enemyLevel, game: currentGame });
    const calculated = detail.dmg(ds, fn);
    const text = calculated?.type === 'text';
    if (text && /(NaN|Infinity|undefined)/.test(String(calculated.avg))) throw Error('rule.nonfinite_text.' + index);
    if (!calculated || (!text && !Number.isFinite(calculated.avg)) || calculated.dmg != null && !Number.isFinite(calculated.dmg)) throw Error('rule.nonfinite.' + index);
    const title = typeof detail.title === 'function' ? detail.title(ds) : detail.title;
    results.push({
      id: String(index),
      title: String(title),
      expected: text ? null : calculated.avg,
      text: text ? String(calculated.avg) : '',
      critical: calculated.dmg ?? null,
      buffs: msg,
      kind: calculated.type || 'damage',
      default: isDefault && !results.some(r => r.default),
    });
  }
  if (!results.length) throw Error('rule.empty');
  const attributes = {};
  for (const key of ['hp', 'atk', 'def', 'hpBase', 'atkBase', 'defBase', 'speed', 'cpct', 'cdmg', 'recharge', 'dmg', 'phy', 'stance']) {
    if (Number.isFinite(anchored[key])) attributes[key] = anchored[key];
  }
  return { weapon: profile.weapon, promote: profile.promote, attributes, results };
}

// 面板换装 follows miao's ProfileChange: the changed panel's properties are
// calculated from the character, light cone, relics and traces with Attr, as
// Avatar.calcAttr does.
function runChange(record, weapons, input) {
  currentGame = record.game;
  weaponCatalog = weapons;
  if (typeof characterRule === 'undefined') globalThis.characterRule = {};
  const char = characterObject(record);
  const weapon = Weapon.get(input.weapon.id);
  if (input.weapon.id && !weapon) throw Error('weapon.missing');
  const promote = input.promote ?? Attr.calcPromote(input.level, currentGame);
  const weaponPromote = input.weapon.promote ?? (weapon ? Attr.calcPromote(input.weapon.level, currentGame) : 0);
  const profile = {
    game: currentGame, char, id: char.id, elem: char.elem, level: input.level, promote, cons: input.rank, talent: input.talents, trees: input.trees,
    weapon: { ...input.weapon, promote: weaponPromote, name: weapon?.name || '', affix: input.weapon.refinement },
    artis: artifactObject(input.equipment),
  };
  const attributes = {};
  for (const [key, value] of Object.entries(new Attr(profile).calc())) {
    if (Number.isFinite(value)) attributes[key] = value;
  }
  return { promote, weapon_promote: weaponPromote, attributes, weapon_attrs: weapon ? weapon.calcAttr(profile.weapon.level, weaponPromote) : null };
}

function runBuild(record, weapons, input) {
  currentGame = record.game;
  weaponCatalog = weapons;
  const char = characterObject(record);
  const weapon = Weapon.get(input.weapon.id);
  if (input.weapon.id && !weapon) throw Error('weapon.missing');
  const profile = {
    game: currentGame, char, id: char.id, elem: char.elem, level: input.level, promote: input.promote, cons: input.rank, talent: input.talents, trees: input.trees,
    weapon: { ...input.weapon, name: weapon?.name || '', affix: input.weapon.refinement },
    artis: artifactObject(input.equipment),
  };
  const original = inferPromotions(profile, input.attributes);
  if (input.resolve_identity === true) return { promote: profile.promote, weapon_promote: profile.weapon.promote };
  const baseline = calculateProfile(profile, input.attributes, original, input.enemy_level);
  let candidate = null;
  if (input.candidate_weapon || input.candidate_equipment || input.conditions) {
    const proposed = input.candidate_weapon || profile.weapon;
    const next = Weapon.get(proposed.id);
    if (proposed.id && (!next || next.type !== char.weapon)) throw Error('weapon.type');
    const candidateProfile = {
      ...profile,
      weapon: { ...proposed, name: next?.name || '', affix: proposed.refinement },
      artis: input.candidate_equipment ? artifactObject(input.candidate_equipment) : profile.artis,
    };
    if (next && !legalPromotions(candidateProfile.weapon.level).includes(candidateProfile.weapon.promote)) throw Error('weapon.promote');
    candidate = calculateProfile(candidateProfile, input.attributes, original, input.enemy_level, input.conditions);
  }
  return { source: 'simulation', version: 'miao-7f6f1c84-reference-v1', character_id: record.id, character: record.name, enemy_level: input.enemy_level, baseline, candidate };
}

// scoringArtifacts stands in for upstream's Artis for ArtisMarkCfg: set
// abbreviations and main-stat checks follow miao's Artis.is, isAttr and
// ArtisSet.getSetData.
function scoringArtifacts(gear) {
  const pieces = {};
  const counts = {};
  for (const g of gear) {
    pieces[g.slot] = { set: g.set_name, main: { key: miaoKey(g.main.key), value: g.main.value }, attrs: g.sub.map(s => ({ key: miaoKey(s.key), value: s.value })) };
    counts[g.set_name] = (counts[g.set_name] || 0) + 1;
  }
  const names = [];
  const abbrs = [];
  const full = [];
  for (const [name, count] of Object.entries(counts)) {
    if (count < 2) continue;
    const n = count >= 4 ? 4 : 2;
    names.push(name);
    abbrs.push((artiMeta.setAbbr[name] || name) + n);
    full.push(name + n);
  }
  const sets = [...abbrs, ...full];
  return {
    names,
    pieces,
    // is checks set bonuses, or with pos the main stats of those slots; the
    // planar sphere (slot 5) also matches dmg with any elemental bonus.
    is(check, pos = '') {
      if (!pos) return String(check).split(',').some(s => sets.includes(s));
      const attrs = String(check).split(',');
      return String(pos).split(',').every(p => {
        const main = pieces[p]?.main.key || '';
        return attrs.includes(main) || p === '5' && attrs.includes('dmg') && Format.isElem(main);
      });
    },
  };
}

function runScore(record, weapons, input) {
  currentGame = record.game;
  weaponCatalog = weapons;
  const data = record.data;
  const weapon = input.weapon.id ? Weapon.get(input.weapon.id) : false;
  const char = { name: record.name, abbr: data.abbr || record.name, game: currentGame, isGs: false, baseAttr: data.baseAttr, getArtisCfg: () => typeof scoreRule === 'undefined' ? false : scoreRule.default };
  const artis = scoringArtifacts(input.equipment);
  const profile = {
    game: currentGame, char, id: Number(record.id), elem: record.element, attr: input.attributes, cons: input.rank, artis,
    weapon: { name: weapon?.name || input.weapon.name || '', affix: input.weapon.refinement, bonusKey: weapon?.detail?.attr?.bonusKey },
  };
  const cfg = ArtisMarkCfg.getCfg(profile);
  // Substat detail as upstream's getMarkDetail shows it: rolls are the
  // official upgrade count plus the initial roll, efficiency is the value in
  // maximum single rolls.
  const attrMap = Meta.getMeta(currentGame, 'arti').attrMap;
  const rolled = s => {
    const key = miaoKey(s.key);
    const max = attrMap[key]?.value;
    return { key, value: s.value, upNum: (s.times || 0) + 1, eff: max ? s.value / max : 0 };
  };
  const pieces = [];
  const all = {};
  let total = 0;
  for (const g of input.equipment) {
    const arti = artis.pieces[g.slot];
    const mark = ArtisMark.getMark({ charCfg: cfg, idx: g.slot, arti, elem: profile.elem, game: currentGame, id: profile.id });
    if (!Number.isFinite(mark)) throw Error('score.nonfinite');
    const attrs = g.sub.map(rolled);
    for (const a of attrs) {
      const t = all[a.key] || (all[a.key] = { key: a.key, value: 0, upNum: 0, eff: 0 });
      t.value += a.value;
      t.upNum += a.upNum;
      t.eff += a.eff;
    }
    total += mark;
    pieces.push({
      slot: g.slot,
      score: mark,
      grade: ArtisMark.getMarkClass(mark) || 'MAX',
      mark: Format.comma(mark, 1),
      main: ArtisMark.formatArti(arti.main, cfg.attrs, true, currentGame),
      attrs: ArtisMark.formatArtiAttrs(attrs, cfg.attrs, currentGame),
    });
  }
  const allAttrs = ArtisMark.formatArti(lodash.sortBy(Object.values(all), ['eff']).reverse(), false, false, currentGame);
  return {
    title: cfg.classTitle,
    weights: lodash.mapValues(cfg.attrs, a => a.weight),
    pieces,
    total,
    mark: Format.comma(total, 1),
    grade: ArtisMark.getMarkClass(total / 6) || 'MAX',
    all_attrs: Array.isArray(allAttrs) ? allAttrs : [],
    titles: ArtisMark.getKeyTitleMap(currentGame),
  };
}
