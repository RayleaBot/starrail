// The parts of miao-plugin's runtime the pinned calculation modules use:
// numbers and formatting only. Logging is silenced so combat values never
// reach the host log.
const lodash = _;
let Format;
const Data = { eachStr: (text, fn) => String(text).split(',').forEach(fn) };
const Cfg = { get: (_key, defaultValue) => defaultValue };

// Base answers unknown properties from _get or meta, as upstream's Base does.
// Like upstream, a class not given a game counts as Genshin; the pinned
// modules pass Star Rail's.
class Base {
  constructor() {
    this.game = 'gs';
    return new Proxy(this, {
      get(target, key, receiver) {
        if (key in target) return Reflect.get(target, key, receiver);
        if (target._get) return target._get.call(receiver, key);
        return target.meta?.[key];
      },
    });
  }
  get isGs() { return this.game === 'gs'; }
  get isSr() { return this.game === 'sr'; }
}

// Upstream getMeta(game, type, key) returns one entry when key is given.
const Meta = {
  getMeta(_game, kind, key) {
    const meta = kind === 'weapon' ? { weaponBuffs } : artiMeta;
    return key ? meta[key] : meta;
  },
};
const console = { log() {} };
const logger = { info() {}, warn() {}, error() {}, debug() {} };

// currentGame is the game upstream's modules are called with; weaponCatalog
// holds the light cones of the running calculation.
let currentGame, weaponCatalog;

const ArtifactSet = {
  getArtisSetBuff(name, count) {
    const data = artifactBuffs[name]?.[count] || artifactBuffs[name + count];
    if (!data) return [];
    return Array.isArray(data) ? data : [data];
  },
};

// loadWeaponDefinitions calls a light cone family's calc.js with upstream's
// static and conditional helpers; plain definition objects are returned as
// they are.
function loadWeaponDefinitions(definitions) {
  if (typeof definitions !== 'function') return definitions;
  const staticEffect = (idx, key) => ({ isStatic: true, idx, key });
  const effect = (title, key, idx) => {
    if (typeof key === 'object') {
      return tables => ({ title, data: Object.fromEntries(Object.entries(key).map(([k, n]) => [k, tables[n]])) });
    }
    return { title, idx, key };
  };
  return definitions(staticEffect, effect);
}

class ReferenceWeapon {
  constructor(record) {
    this.name = record.name;
    this.type = record.type;
    this.id = record.id;
    this.detail = record.data;
    this.game = currentGame;
  }

  // calcAttr is the light cone's base attributes at a level and ascension.
  calcAttr(level, promote) {
    const values = this.detail.attr[promote];
    if (!values) throw Error('weapon.promote');
    return Object.fromEntries(Object.entries(values.attrs).map(([k, v]) => [k, v * 1 + (this.detail.growAttr?.[k] || 0) * (level - 1)]));
  }

  // getWeaponAffixBuffs returns the permanent or the combat effects at a
  // superimposition, as upstream's Weapon.getWeaponAffixBuffs.
  getWeaponAffixBuffs(affix, isStatic = true) {
    let definitions = lodash.cloneDeep(weaponBuffs[this.id] || weaponBuffs[this.name] || []);
    if (!Array.isArray(definitions)) definitions = [definitions];
    const tables = Object.fromEntries(Object.entries(this.detail.skill?.tables || {}).map(([k, v]) => [k, v[affix - 1]]));
    const result = [];
    for (let data of definitions) {
      if (typeof data === 'function') data = data(tables);
      if (!data || !!data.isStatic !== !!isStatic) continue;
      if (isStatic) {
        const values = {};
        if (data.idx && data.key) values[data.key] = tables[data.idx];
        for (const [k, v] of Object.entries(data.refine || {})) values[k] = v[affix - 1] * (data.buffCount || 1);
        if (Object.keys(values).length) result.push({ isStatic: true, data: values });
        continue;
      }
      data.title = /：/.test(data.title) ? data.title : this.name + '：' + (data.title || '额外效果');
      data.data = data.data || {};
      if (data.idx && data.key) {
        data.data[data.key] = tables[data.idx];
      } else {
        for (const [k, v] of Object.entries(data.refine || {})) data.data[k] = ({ refine }) => v[refine] * (data.buffCount || 1);
      }
      result.push(data);
    }
    return result;
  }
}

const Weapon = {
  get(name) {
    const record = weaponCatalog.find(w => w.name === name || w.id === String(name));
    return record ? new ReferenceWeapon(record) : false;
  },
};
