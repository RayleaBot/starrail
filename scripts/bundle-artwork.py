"""Copies the upstream images, fonts and data files the templates read from
the pinned reference snapshots into assets/<source>/. The plugin build ships
the top-level assets/ directory, and the artwork store serves these files until
an administrator downloads a newer copy of the source. assets/<source>.json
records the upstream commit and the totals the status shows.

Usage: python scripts/bundle-artwork.py --references <参考项目/2026-09-15>

Run it before building a package. Official images are fetched file by file at
render time, and the Atlas is downloaded on request, so neither is bundled.
"""
import argparse
import fnmatch
import json
import pathlib
import shutil

# Source id: the reference snapshot and the files the Go code reads from it.
# A pattern's * also matches "/".
BUNDLES = {
    "miao-plugin": ("miao-plugin", [
        "resources/common/font/tttgbnumber.woff",
        "resources/common/font/NZBZ.woff",
        "resources/common/font/HYWH-65W.woff",
        "resources/common/bg/bg-*.webp",
        "resources/common/bg/talent-*.webp",
        "resources/common/cont/card-bg.png",
        "resources/common/item/*",
        "resources/character/imgs/*",
        "resources/wiki/imgs/*",
        "resources/gacha/imgs/*",
        "resources/meta-sr/public/icons/*",
        # Panels, rankings, the roster and the wiki draw every character's
        # splash, face, trace and eidolon icons.
        "resources/meta-sr/character/*/data.json",
        "resources/meta-sr/character/*/imgs/splash.webp",
        "resources/meta-sr/character/*/imgs/face.webp",
        "resources/meta-sr/character/*/imgs/talent-*.webp",
        "resources/meta-sr/character/*/imgs/cons-*.webp",
        "resources/meta-sr/character/*/imgs/tree-*.webp",
        "resources/meta-sr/character/common/imgs/*",
        "resources/meta-sr/weapon/*/splash.webp",
        "resources/meta-sr/weapon/*/icon.webp",
        "resources/meta-sr/weapon/*/icon-s.webp",
        "resources/meta-sr/artifact/*/arti-[1-6].webp",
    ]),
    "starrail-plugin": ("StarRail-plugin", [
        "resources/common/font/SR.ttf",
        "resources/common/font/tttgbnumber.ttf",
        "resources/common/font/YS.ttf",
        "resources/common/font/汉仪润圆-75W.ttf",
        "resources/common/cont/*",
        "resources/common/element/*",
        "resources/common/rogue/*",
        "resources/challenge/images/*",
        "resources/rogue/ball.png",
        "resources/rogue/bg.png",
        "resources/rogue/oblivious_bg@2x.7b5d7b38.png",
        "resources/rogue/tourn/simulated-s3-bg-top.e5a5b60a.webp",
        "resources/rogue/locust/*",
        "resources/gridFight/*",
        "resources/card/images/logo.png",
        "resources/images/rank4bg.webp",
        "resources/images/rank5bg.webp",
        "resources/help/icon.png",
        "resources/help/theme/default/*",
        "resources/month/*",
        "resources/charge/*",
        "resources/online/jeremy-thomas-E0AHdsENmDg-unsplash.jpg",
        "resources/note/resources/background.png",
        "resources/gachasimulation/resources/images/*",
        "resources/gatcha/images/char/*",
        "resources/panel/resources/weapon/*",
        # The panel service's character list and panel: the pictures, skill,
        # relic and stat icons, the skill tree backgrounds and the data
        # naming the paths, relic icons and tree layouts.
        "resources/panel/resources/avatar/*.png",
        "resources/panel/resources/char_image/*",
        "resources/profile/normal-character/*",
        "resources/panel/resources/dividing-line.png",
        "resources/panel/resources/icon/value/*",
        "resources/panel/resources/star/*",
        "resources/panel/resources/skill/*",
        "resources/panel/resources/relic/*",
        "resources/panel/resources/skill_tree/*",
        "resources/panel/data/*.json",
        "resources/baseData/characters.json",
        "resources/baseData/light_cones.json",
        # Strategy pictures sent by the static picture commands.
        "resources/srsr/*",
    ]),
    # The ledger pages, 抽卡记录, 抽卡统计, the UID list and the news pages.
    "yunzai-genshin": ("Yunzai-genshin", [
        "resources/StarRail/img/note/*",
        "resources/StarRail/img/other/*",
        "resources/img/icon/check.webp",
        "resources/img/other/bg4.png",
        "resources/img/other/bg5.png",
        "resources/html/mysNews/iconfont.fb3712d.woff2",
        "resources/html/mysNews/mys.png",
        "resources/html/mysNews-list/蒙德.png",
        "resources/font/tttgbnumber.ttf",
        "resources/font/HYWenHei-55W.ttf",
    ]),
    "ark-plugin": ("ark-plugin", ["resources/graph/background.png"]),
}

parser = argparse.ArgumentParser()
parser.add_argument("--references", type=pathlib.Path, required=True)
args = parser.parse_args()
refs = args.references.resolve()
root = pathlib.Path(__file__).resolve().parents[1]
game = json.loads((root / "internal/assets/game.json").read_text(encoding="utf-8"))
extensions = {source["id"]: tuple(source.get("extensions", [])) for source in game["artwork"]}

for source_id, (snapshot, patterns) in BUNDLES.items():
    base = refs / snapshot
    pinned = json.loads((refs / f"{snapshot}.source.json").read_text(encoding="utf-8"))
    names = [file.relative_to(base).as_posix() for file in base.rglob("*") if file.is_file() and ".git" not in file.relative_to(base).parts]
    # The download keeps the same extensions, so an update never drops a
    # bundled file type.
    names = [name for name in names if name.lower().endswith(extensions[source_id])]
    chosen = set()
    for pattern in patterns:
        matched = [name for name in names if fnmatch.fnmatchcase(name, pattern)]
        if not matched:
            raise SystemExit(f"{source_id}: {pattern} matches no file in {snapshot}")
        chosen.update(matched)
    target = root / "assets" / source_id
    shutil.rmtree(target, ignore_errors=True)
    size = 0
    for name in sorted(chosen):
        file = target / name
        file.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(base / name, file)
        size += file.stat().st_size
    record = {"commit": pinned["commit"], "archive": pinned["archive_url"], "files": len(chosen), "bytes": size}
    (root / "assets" / f"{source_id}.json").write_text(json.dumps(record, indent=2) + "\n", encoding="utf-8")
    print(f"assets/{source_id}: {len(chosen)} files, {size / 1e6:.1f} MB at {pinned['commit'][:7]}")
