"""Convert pinned local miao-plugin JSON into internal/assets/catalog.json.

Usage: python scripts/import-reference-data.py --references <参考项目/2026-09-15>

This reads data only. It never evaluates reference JavaScript or installs its
dependencies. Generated catalogs carry their source version and attribution.
Aliases, short names and the equipment-set tables that import-aliases.py adds
are kept from the existing catalog, so the two scripts can run in any order.

Skill texts carry miao's placeholders ($1[i]%: table 1, integer, percent).
Characters get the values of one reference level (普攻 6, others 10, the
levels the game shows without eidolons); light cones get the range from
superimposition 1 to 5.
"""
import argparse
import html
import json
import pathlib
import re
import shutil

parser = argparse.ArgumentParser()
parser.add_argument("--references", type=pathlib.Path, required=True)
args = parser.parse_args()
refs = args.references.resolve()
root = pathlib.Path(__file__).resolve().parents[1]
target = root / "internal/assets/catalog.json"

PLACEHOLDER = re.compile(r"\$(\d+)\[(i|f1|f2)\](%?)")
REFERENCE_LEVELS = {"普攻": 6, "战技": 10, "终结技": 10, "天赋": 10}


def clean(value):
    if not isinstance(value, str):
        return ""
    value = re.sub(r"<br\s*/?>", "\n", value, flags=re.I)
    value = html.unescape(re.sub(r"<[^>]*>", "", value))
    return value.encode("utf-8", errors="replace").decode("utf-8").strip()


def name_of(value):
    # miao wraps some skill names in bars; the game shows them bare.
    return clean(value).strip("|").strip()


def number(value, fmt, pct, scale):
    if not isinstance(value, (int, float)):
        return str(value)
    if pct:
        value *= scale
        decimals = 2 if fmt == "f2" else 1
    else:
        decimals = 2
    text = f"{value:.{decimals}f}"
    if "." in text:
        text = text.rstrip("0").rstrip(".")
    return text + ("%" if pct else "")


def values_of(table):
    if isinstance(table, dict):
        table = table.get("values")
    return table if isinstance(table, list) and table else None


def render(text, tables, level, scale=100):
    """Fills the placeholders of one description: with the value at level
    (1-based, clamped to the table) or, when level is None, the range from the
    first to the last value. Character tables hold fractions (scale 100 for a
    percent); light cone tables already hold percentages (scale 1)."""
    used = False

    def fill(match):
        nonlocal used
        index, fmt, pct = match.groups()
        values = values_of((tables or {}).get(index))
        if not values:
            return match.group(0)
        used = True
        if level is None:
            first, last = number(values[0], fmt, pct, scale), number(values[-1], fmt, pct, scale)
            return first if first == last else f"{first}–{last}"
        return number(values[min(level, len(values)) - 1], fmt, pct, scale)

    return PLACEHOLDER.sub(fill, text), used


def lines_of(value):
    if isinstance(value, str):
        value = [value]
    return [clean(line) for line in (value or []) if clean(line)]


def load(file):
    return json.loads(file.read_text(encoding="utf-8"))


def dump(file, value):
    file.parent.mkdir(parents=True, exist_ok=True)
    file.write_bytes((json.dumps(value, ensure_ascii=False, indent=2) + "\n").encode("utf-8"))


def stats_of(raw):
    """The attributes at the highest ascension, with the level they hold at."""
    best = None
    phases = raw.get("attr") or {}
    for phase in (phases.values() if isinstance(phases, dict) else phases):
        if isinstance(phase, dict) and isinstance(phase.get("attrs"), dict) and (best is None or phase.get("maxLevel", 0) > best.get("maxLevel", 0)):
            best = phase
    if best is None:
        return {}
    stats = {key: round(value, 1) if isinstance(value, float) else value for key, value in best["attrs"].items() if isinstance(value, (int, float))}
    stats["level"] = best.get("maxLevel", 0)
    if isinstance(raw.get("sp"), (int, float)):
        stats["sp"] = raw["sp"]
    return stats


def character_talents(raw):
    talents = []
    for talent in (raw.get("talent") or {}).values():
        if not isinstance(talent, dict):
            continue
        kind = clean(talent.get("type"))
        level = REFERENCE_LEVELS.get(kind, 1)
        description, used = [], False
        for line in lines_of(talent.get("desc")):
            text, filled = render(line, talent.get("tables"), level)
            used = used or filled
            description.append(text)
        talents.append({"name": name_of(talent.get("name")) or "技能", "type": kind, "tag": clean(talent.get("tag")), "level": level if used else 0, "description": description})
    return talents


def weapon_talents(raw):
    skill = raw.get("skill")
    if not isinstance(skill, dict) or not skill.get("desc"):
        return []
    description = [render(line, skill.get("tables"), None, 1)[0] for line in lines_of(skill.get("desc"))]
    return [{"name": name_of(skill.get("name")) or "光锥技能", "type": "光锥技能", "tag": "", "level": 0, "description": description}]


def cons_of(raw):
    cons = []
    for key in sorted((raw.get("cons") or {}).keys(), key=lambda k: int(k) if str(k).isdigit() else 0):
        item = raw["cons"][key]
        if isinstance(item, dict) and item.get("name"):
            cons.append({"name": name_of(item.get("name")), "description": lines_of(item.get("desc"))})
    return cons


existing = load(target) if target.exists() else {}
kept = {entry["id"]: entry for entry in existing.get("entries", []) if isinstance(entry, dict) and "id" in entry}
entries = {}
base = refs / "miao-plugin/resources/meta-sr"
for kind in ("character", "weapon"):
    for file in sorted((base / kind).rglob("data.json")):
        raw = load(file)
        if not isinstance(raw, dict) or not raw.get("name") or not raw.get("id"):
            continue
        item_id = str(raw["id"])
        if item_id in entries:
            continue
        previous = kept.get(item_id, {})
        entry = {
            "id": item_id,
            "name": clean(raw["name"]),
            "aliases": previous.get("aliases") or ([clean(raw["abbr"])] if raw.get("abbr") and raw["abbr"] != raw["name"] else []),
            "kind": kind,
            "rarity": raw.get("star", 0),
            # Elements and paths are already named in Chinese; a light cone's
            # path is its type.
            "element": raw.get("elem", "") if kind == "character" else "",
            "weapon": raw.get("weapon", "") if kind == "character" else clean(raw.get("type")),
            "description": clean(raw.get("desc")),
            "materials": {k: clean(v) for k, v in (raw.get("materials") or {}).items() if isinstance(v, str)},
            "stats": stats_of(raw),
            "talents": character_talents(raw) if kind == "character" else weapon_talents(raw),
        }
        if previous.get("abbr"):
            entry["abbr"] = previous["abbr"]
        if kind == "character":
            entry["cons"] = cons_of(raw)
        entries[item_id] = entry
source = "https://github.com/yoimiya-kokomi/miao-plugin/tree/7f6f1c84c89102bc6b1c58c8e1b06c61f4642161/resources/meta-sr"
catalog = {"version": "miao-7f6f1c84c891", "source": source, "entries": sorted(entries.values(), key=lambda item: item["id"])}
catalog.update({key: value for key, value in existing.items() if key not in catalog})
dump(target, catalog)
shutil.copyfile(refs / "miao-plugin/LICENSE", root / "LICENSES/miao-plugin-MIT.txt")
print(json.dumps({"entries": len(entries)}, ensure_ascii=True))
