package images

import (
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/RayleaBot/starrail/internal/app"
	"github.com/RayleaBot/starrail/internal/gacha"
)

// miaoGachaArtwork are the fonts and backgrounds of miao's gacha-stat page,
// 抽卡统计.
var miaoGachaArtwork = [][2]string{
	{"Number", "resources/common/font/tttgbnumber.woff"},
	{"NZBZ", "resources/common/font/NZBZ.woff"},
	{"YS", "resources/common/font/HYWH-65W.woff"},
	{"common-bg-bg-sr", "resources/common/bg/bg-sr.webp"},
	{"common-cont-card-bg", "resources/common/cont/card-bg.png"},
	{"common-item-bg1", "resources/common/item/bg1.png"},
	{"common-item-bg2", "resources/common/item/bg2.png"},
	{"common-item-bg3", "resources/common/item/bg3.png"},
	{"common-item-bg4", "resources/common/item/bg4.png"},
	{"common-item-bg5", "resources/common/item/bg5.png"},
	{"gacha-imgs-date-icon", "resources/gacha/imgs/date-icon.webp"},
}

// miaoVersion is a period of miao's GachaData: a warp half with its rate-up
// five-stars. The 新版本 after the last known period has no end.
type miaoVersion struct {
	version, half, from, to string
	char5, weapon5          []string
}

// before is whether a pull is older than the period, so the next period has
// to be looked up. Upstream never looks again once a pull falls in no
// period, which puts every older pull in it; a period without a start is
// looked up again each pull.
func (v miaoVersion) before(at string) bool { return v.from == "" || at < v.from }

// miaoPeriods are miao's poolVersionSr: the banner data's warps, then 新版本.
func miaoPeriods(context app.ImageContext) []miaoVersion {
	versions := []miaoVersion{}
	for _, pool := range context.Game.Data.Resources.Pools {
		if pool.Kind == "event" {
			versions = append(versions, miaoVersion{pool.Version, pool.Half, pool.From, pool.To, pool.Characters5, pool.Weapons5})
		}
	}
	sort.SliceStable(versions, func(i, j int) bool { return versions[i].from < versions[j].from })
	// Upstream ends 新版本 on a fixed date its data has since passed.
	if len(versions) > 0 {
		versions = append(versions, miaoVersion{version: "新版本", half: "?", from: versions[len(versions)-1].to})
	}
	return versions
}

// miaoPeriod is miao's getVersion: the period a pull falls in, else 未知, or
// 全部 where periods are not told apart.
func miaoPeriod(periods []miaoVersion, at string, versioned bool) miaoVersion {
	if !versioned {
		return miaoVersion{version: "全部"}
	}
	for _, version := range periods {
		if at > version.from && (version.to == "" || at < version.to) {
			return version
		}
	}
	return miaoVersion{version: "未知"}
}

// miaoItem is an item of miao's readJSON: the character or light cone pulled.
type miaoItem struct {
	id, name, abbr, img string
	weapon              bool
	star                int
}

// miaoPull is a pull of an item at a time.
type miaoPull struct {
	item *miaoItem
	at   string
}

// miaoPulls are miao's readJSON over the pools: each record once, newest
// first, with its item. Names the catalog does not know are 未知 as upstream's
// 403 and 404, kept apart by the record's star, which upstream fixes at four
// for a character and three for a light cone.
func miaoPulls(context app.ImageContext, resources *app.ImageResources, archive gacha.Archive, pools ...string) []miaoPull {
	records := []gacha.Record{}
	seen := map[string]bool{}
	for _, record := range archive.Records {
		if slices.Contains(pools, record.GachaType) && !seen[record.ID] {
			seen[record.ID] = true
			records = append(records, record)
		}
	}
	// Newest first, as upstream sorts the pools it reads together; record
	// IDs order the pulls of one time.
	sort.Slice(records, func(i, j int) bool {
		if records[i].Time != records[j].Time {
			return records[i].Time > records[j].Time
		}
		if len(records[i].ID) != len(records[j].ID) {
			return len(records[i].ID) > len(records[j].ID)
		}
		return records[i].ID > records[j].ID
	})
	paths := lightConePaths(context)
	items := map[string]*miaoItem{}
	pulls := []miaoPull{}
	for _, record := range records {
		weapon := record.ItemType == "光锥" || record.ItemType == "武器"
		if !weapon && record.ItemType != "角色" {
			continue
		}
		kind := "character"
		if weapon {
			kind = "weapon"
		}
		item := items[kind+"/"+record.Name]
		if item == nil {
			item = &miaoItem{name: "未知", abbr: "未知", weapon: weapon}
			// Character.get and Weapon.get read a name or alias, never part
			// of one.
			if entry, ok := context.Catalog.Resolve(record.Name, kind, nil); ok && (entry.Name == record.Name || slices.Contains(entry.Aliases, record.Name)) {
				item.id, item.name, item.abbr, item.star = entry.ID, entry.Name, entry.Name, entry.Rarity
				// A light cone keeps a name of up to four characters.
				if entry.Abbr != "" && !(weapon && utf8.RuneCountInString(entry.Name) <= 4) {
					item.abbr = entry.Abbr
				}
				item.img = resources.Artwork(kind+"-"+entry.ID, "miao-plugin", poolPicture(entry, paths[entry.ID]))
			} else {
				item.star, _ = strconv.Atoi(record.Rank)
			}
			items[kind+"/"+record.Name] = item
		}
		pulls = append(pulls, miaoPull{item, record.Time})
	}
	return pulls
}

// miaoPlayer is miao's getFace with the name kept for the UID. The face is
// that of the player's head icon, but Star Rail head icons are no character
// IDs, so upstream always falls back to 卡芙卡 (1005) and her banner, which is
// miao's common Star Rail banner.
func miaoPlayer(context app.ImageContext, resources *app.ImageResources, image app.GachaImage) map[string]any {
	name := image.Nickname
	if name == "" {
		name = "旅行者"
	}
	face := ""
	if entry, ok := context.Catalog.Get("1005"); ok {
		face = resources.Artwork("player-face", "miao-plugin", poolPicture(entry, ""))
	}
	return map[string]any{"elem": "sr", "uid": image.UID, "name": name, "face": face,
		"banner": resources.Artwork("player-banner", "miao-plugin", "resources/meta-sr/character/common/imgs/banner.webp"), "mark": context.Game.Prefix}
}

// miaoStat is a figure in miao's header or a period's summary.
func miaoStat(stats []any, value any, title string) []any {
	return append(stats, map[string]any{"value": value, "title": title})
}

// statLabels are the warps that do not split by period: standard and
// departure have no periods, and the banner data has no collaboration warps.
var statLabels = map[string]string{"1": "常驻池", "2": "新手池", "21": "角色联动池", "22": "光锥联动池", "21,22": "联动池"}

// GachaStat draws 抽卡统计 the way miao's GachaData.stat and gacha/gacha-stat
// do for Star Rail: the totals over the warps the word names, then each
// period, newest first, with its rate-ups, dates and counts and every four-
// and five-star it gave, rate-ups marked; standard, departure, collaboration
// warps and 全部 each form one block.
const statPeriods = 10

var statUID = regexp.MustCompile(`[0-9]+$`)

func GachaStat(context app.ImageContext, image app.GachaImage) (app.Image, bool) {
	word := strings.TrimSpace(statUID.ReplaceAllString(strings.NewReplacer("喵喵", "", "统计", "", "分析", "", "池", "").Replace(image.Word), ""))
	paged := strings.Contains(word, "全部") && word != "全部"
	if paged {
		word = strings.ReplaceAll(word, "全部", "")
	}
	pools, all := gachaSelection(word)
	poolName := "抽卡"
	if slices.Equal(pools, []string{"11"}) {
		poolName = "角色"
	}
	if slices.Equal(pools, []string{"12"}) {
		poolName = "光锥"
	}
	label := statLabels[strings.Join(pools, ",")]
	switch {
	case all:
		// 全部 is every warp but the departure warp, in one block.
		pools, label = []string{"11", "12", "21", "22", "1"}, "全部统计"
	case pools == nil:
		// As miao, no warp named is both limited warps by period.
		pools = []string{"11", "12"}
	}
	resources := &app.ImageResources{Context: context}
	for _, item := range miaoGachaArtwork {
		resources.Artwork(item[0], "miao-plugin", item[1])
	}
	pulls := miaoPulls(context, resources, image.Archive, pools...)
	if len(pulls) == 0 {
		return app.Image{}, false
	}
	versioned := label == ""
	periods := miaoPeriods(context)
	type block struct {
		version miaoVersion
		counts  map[*miaoItem]int
	}
	blocks := []*block{}
	for _, pull := range pulls {
		if len(blocks) == 0 || versioned && blocks[len(blocks)-1].version.before(pull.at) {
			version := miaoPeriod(periods, pull.at, versioned)
			if !versioned {
				version.version = label
			}
			// A pull in no period stays with the unknown block before it.
			if len(blocks) == 0 || !(version.from == "" && blocks[len(blocks)-1].version.version == version.version) {
				blocks = append(blocks, &block{version, map[*miaoItem]int{}})
			}
		}
		blocks[len(blocks)-1].counts[pull.item]++
	}
	keys := []string{"totalNum", "star5Num", "c5UpNum", "w5UpNum", "star4Num", "upNum", "c4Num", "w4Num"}
	total := map[string]int{}
	versions := []any{}
	for _, entry := range blocks {
		version := entry.version
		names := []string{}
		for _, name := range version.char5 {
			if found, ok := context.Catalog.Resolve(name, "character", nil); ok && found.Abbr != "" {
				name = found.Abbr
			}
			names = append(names, name)
		}
		up := map[string]bool{}
		for _, name := range append(slices.Clone(version.char5), version.weapon5...) {
			up[name] = true
		}
		counts := map[string]int{}
		type counted struct {
			item *miaoItem
			num  int
			up   bool
		}
		list := []counted{}
		for item, num := range entry.counts {
			isUp := up[item.name]
			list = append(list, counted{item, num, isUp})
			switch {
			case !item.weapon && item.star == 5:
				counts["c5Num"] += num
				if isUp {
					counts["c5UpNum"] += num
				}
			case !item.weapon:
				counts["c4Num"] += num
			case item.star == 5:
				counts["w5Num"] += num
				if isUp {
					counts["w5UpNum"] += num
				}
			case item.star == 4:
				counts["w4Num"] += num
			default:
				counts["w3Num"] += num
			}
		}
		counts["upNum"] = counts["w5UpNum"] + counts["c5UpNum"]
		counts["star5Num"] = counts["w5Num"] + counts["c5Num"]
		counts["star4Num"] = counts["w4Num"] + counts["c4Num"]
		counts["totalNum"] = counts["star5Num"] + counts["star4Num"] + counts["w3Num"]
		for _, key := range keys {
			total[key] += counts[key]
		}
		// Upstream's sortBy star, count and rate-up, reversed; the rest by
		// ID, as object keys enumerate.
		sort.Slice(list, func(i, j int) bool {
			a, b := list[i], list[j]
			if a.item.star != b.item.star {
				return a.item.star > b.item.star
			}
			if a.num != b.num {
				return a.num > b.num
			}
			if a.up != b.up {
				return a.up
			}
			x, _ := strconv.Atoi(a.item.id)
			y, _ := strconv.Atoi(b.item.id)
			return x > y
		})
		items := []any{}
		for _, item := range list {
			if item.item.star < 4 {
				continue
			}
			name := item.item.name
			if utf8.RuneCountInString(name) > 4 {
				name = item.item.abbr
			}
			items = append(items, map[string]any{"up": item.up, "star": item.item.star, "img": item.item.img, "num": item.num, "name": name})
		}
		stats := []any{}
		for _, key := range []string{"totalNum", "star5Num", "upNum", "c4Num", "w4Num"} {
			if counts[key] > 0 {
				stats = miaoStat(stats, counts[key], map[string]string{"totalNum": "总抽卡", "star5Num": "金卡", "upNum": "UP金卡", "c4Num": "紫角色", "w4Num": "紫武器"}[key])
			}
		}
		// Upstream prints Invalid date for a pull in no period; it shows as
		// the unversioned blocks do here.
		from, to := "", ""
		if versioned && version.from != "" {
			from = version.from[2:10]
			if version.to != "" {
				to = version.to[2:10]
			}
		}
		versions = append(versions, map[string]any{"version": version.version, "half": version.half, "from": from, "to": to, "name": strings.Join(names, " / "), "stats": stats, "items": items})
	}
	stats := []any{}
	for _, key := range []string{"totalNum", "star5Num", "c5UpNum", "w5UpNum", "star4Num"} {
		if total[key] > 0 {
			stats = miaoStat(stats, total[key], map[string]string{"totalNum": "抽卡总数", "star5Num": "金卡", "c5UpNum": "UP角色", "w5UpNum": "UP武器", "star4Num": "紫卡"}[key])
		}
	}
	if total["upNum"] > 0 {
		stats = miaoStat(stats, app.JSFixed(float64(total["totalNum"])/float64(total["upNum"]), 1), "平均UP抽")
	}
	data := miaoPlayer(context, resources, image)
	data["pool_name"] = poolName
	if paged && len(versions) > statPeriods {
		count := len(versions)
		pages := (count + statPeriods - 1) / statPeriods
		page := min(max(image.Page, 1), pages)
		from, to := (page-1)*statPeriods, min(page*statPeriods, count)
		data["page"] = map[string]any{"page": page, "total": pages, "from": from + 1, "to": to, "count": count}
		versions = versions[from:to]
	} else if len(versions) > statPeriods {
		data["shown"], data["hidden"] = statPeriods, len(versions)-statPeriods
		versions = versions[:statPeriods]
	}
	data["stats"], data["card_width"], data["versions"] = stats, 90, versions
	return app.Image{Template: "gacha-stat", Data: data, Resources: resources.List}, true
}
