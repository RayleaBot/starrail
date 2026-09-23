package images

import (
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/RayleaBot/plugin-starrail/internal/app"
	"github.com/RayleaBot/plugin-starrail/internal/gacha"
)

// countWords are stripped before picking the pool, as LogCount.getPool does.
var countWords = regexp.MustCompile(`#|抽卡|记录|祈愿|分析|池|原神|星铁|崩坏星穹铁道|铁道|统计`)

// countStandard are the standard five-stars LogCount ranks one lower.
var countStandard = []string{"姬子", "杰帕德", "彦卿", "白露", "瓦尔特", "克拉拉", "布洛妮娅", "银河铁道之夜", "无可取代的东西", "但战斗还未结束", "以世界之名", "制胜的瞬间", "如泥酣眠", "时节不居"}

// countArtwork are the fonts and images of Yunzai's log-count page; its
// genshinLayout header shows the banner miao gives every Star Rail character.
var countArtwork = [][3]string{
	{"tttgbnumber", "yunzai-genshin", "resources/font/tttgbnumber.ttf"},
	{"HYWenHei-55W", "yunzai-genshin", "resources/font/HYWenHei-55W.ttf"},
	{"img-other-bg4", "yunzai-genshin", "resources/img/other/bg4.png"},
	{"img-other-bg5", "yunzai-genshin", "resources/img/other/bg5.png"},
	{"logo", "yunzai-genshin", "resources/StarRail/img/other/logo.png"},
	{"head-img", "miao-plugin", "resources/meta-sr/character/common/imgs/banner.webp"},
}

// countPool is one of LogCount's pools: its period and its five-stars.
type countPool struct {
	from, to string
	five     []string
}

// countPools are the pools of a type as Yunzai's defSet/pool lists them: the
// limited warps from the banner data (Yunzai's pinned lists stop in 2025),
// the others one pool for all time as upstream has them.
func countPools(context app.ImageContext, pool string) []countPool {
	switch pool {
	case "1":
		return []countPool{{"2020-04-26 08:00:00", "2050-09-15 17:59:59", []string{"常驻池"}}}
	case "2":
		return []countPool{{"2023-04-26 08:00:00", "2050-09-15 17:59:59", []string{"新手池"}}}
	}
	pools := []countPool{}
	for _, banner := range context.Game.Data.Resources.Pools {
		five := banner.Characters5
		if pool == "12" {
			five = banner.Weapons5
		}
		if banner.Kind == "event" && len(five) > 0 {
			pools = append(pools, countPool{banner.From, banner.To, five})
		}
	}
	sort.SliceStable(pools, func(i, j int) bool { return pools[i].from < pools[j].from })
	return pools
}

// GachaCount draws 抽卡统计 the way Yunzai's LogCount and html/gacha/log-count
// do for Star Rail: newest pool first, each with its five-stars, pulls and
// dates and every four- and five-star it gave with how often, five-star
// characters first; in a group only about twelve rows. Yunzai's short names
// are Genshin's only, so names print in full.
func GachaCount(context app.ImageContext, image app.GachaImage) (app.Image, bool) {
	pool, typeName := "11", "角色"
	switch countWords.ReplaceAllString(image.Word, "") {
	case "常驻":
		pool, typeName = "1", "常驻"
	case "武器", "光锥":
		pool, typeName = "12", "光锥"
	case "新手":
		pool, typeName = "2", "新手"
	}
	records := []gacha.Record{}
	for _, record := range image.Archive.Records {
		if record.GachaType == pool {
			records = append(records, record)
		}
	}
	// Oldest first; record IDs grow with time.
	sort.Slice(records, func(i, j int) bool {
		if len(records[i].ID) != len(records[j].ID) {
			return len(records[i].ID) < len(records[j].ID)
		}
		return records[i].ID < records[j].ID
	})
	type counted struct {
		name, itemType string
		rank, count    int
	}
	type period struct {
		pool  countPool
		count int
		items []*counted
	}
	remaining := countPools(context, pool)
	periods := map[string]*period{}
	order := []string{}
	for _, record := range records {
		// LogCount drops every pool before the one a pull falls in, so a
		// pull outside every pool ends the counting.
		for len(remaining) > 0 && !(record.Time >= remaining[0].from && record.Time <= remaining[0].to) {
			remaining = remaining[1:]
		}
		if len(remaining) == 0 {
			break
		}
		current := remaining[0]
		entry := periods[current.from]
		if entry == nil {
			entry = &period{pool: current}
			periods[current.from] = entry
			order = append(order, current.from)
		}
		entry.count++
		rank, _ := strconv.Atoi(record.Rank)
		if rank < 4 || rank == 5 && record.Name == "未知" {
			continue
		}
		index := slices.IndexFunc(entry.items, func(item *counted) bool { return item.name == record.Name })
		if index < 0 {
			entry.items = append(entry.items, &counted{record.Name, record.ItemType, rank, 0})
			index = len(entry.items) - 1
		}
		entry.items[index].count++
	}
	if len(order) == 0 {
		return app.Image{}, false
	}
	resources := &app.ImageResources{Context: context}
	for _, item := range countArtwork {
		resources.Artwork(item[0], item[1], item[2])
	}
	paths := lightConePaths(context)
	icons := map[string]string{}
	pools := []any{}
	lines := 0
	for index := len(order) - 1; index >= 0; index-- {
		entry := periods[order[index]]
		lines++
		sorted := slices.Clone(entry.items)
		key := func(item *counted) int {
			key := (item.rank-3)*1000 + item.count
			if slices.Contains(countStandard, item.name) {
				key--
			}
			if item.itemType == "角色" && item.rank == 5 {
				key += 1000
			}
			return key
		}
		sort.SliceStable(sorted, func(i, j int) bool { return key(sorted[i]) > key(sorted[j]) })
		items := []any{}
		for _, item := range sorted {
			icon, seen := icons[item.name]
			if !seen {
				// GachaLog.getIcon: a character's face, a light cone's icon.
				kind := "character"
				if item.itemType == "光锥" {
					kind = "weapon"
				}
				if found, ok := context.Catalog.Resolve(item.name, kind, nil); ok {
					icon = resources.Artwork(kind+"-"+found.ID, "miao-plugin", poolPicture(found, paths[found.ID]))
				}
				icons[item.name] = icon
			}
			items = append(items, map[string]any{"rank": item.rank, "count": item.count, "life5": item.count >= 5 && item.rank == 5, "icon": icon})
		}
		pools = append(pools, map[string]any{"five": strings.Join(entry.pool.five, "、"), "count": entry.count, "start": entry.pool.from[:10], "end": entry.pool.to[:10], "items": items})
		lines += (len(items) + 5) / 6
		if image.Group && lines >= 12 {
			break
		}
	}
	return app.Image{Template: "gacha-count", Data: map[string]any{"uid": image.UID, "type_name": typeName, "dated": typeName != "常驻" && typeName != "新手", "group": image.Group, "pools": pools}, Resources: resources.List}, true
}
