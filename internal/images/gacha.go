package images

import (
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/starrail/internal/app"
	"github.com/RayleaBot/starrail/internal/gacha"
)

// gachaPools are the six warp pools upstream reports, in its order.
var gachaPools = []struct{ id, name string }{{"1", "群星跃迁"}, {"2", "新手跃迁"}, {"11", "限定跃迁"}, {"12", "光锥跃迁"}, {"21", "角色联动跃迁"}, {"22", "光锥联动跃迁"}}

// gachaSelection reads the warps a record word names: nil when it names
// none, as 跃迁记录, 抽卡统计 or up记录, and all when it says 全部. A
// collaboration word is read before the character or light cone warp its
// name holds; 联动 alone is both collaboration warps. 抽卡, 抽奖, 集录,
// 喵喵 and a glued UID name no warp.
func gachaSelection(word string) (pools []string, all bool) {
	has := func(names ...string) bool {
		return slices.ContainsFunc(names, func(name string) bool { return strings.Contains(word, name) })
	}
	switch {
	case has("全部"):
		return nil, true
	case has("角色联动"):
		return []string{"21"}, false
	case has("光锥联动", "武器联动"):
		return []string{"22"}, false
	case has("联动"):
		return []string{"21", "22"}, false
	case has("新手"):
		return []string{"2"}, false
	case has("常驻"):
		return []string{"1"}, false
	case has("角色"):
		return []string{"11"}, false
	case has("光锥", "武器"):
		return []string{"12"}, false
	}
	return nil, false
}

const timeLayout = "2006-01-02 15:04:05"

// gachaArtwork maps the images named in the gacha stylesheet, converted from
// StarRail-plugin's common and gatcha/new stylesheets, to their paths.
var gachaArtwork = [][2]string{
	{"common-cont-bkg", "resources/common/cont/bkg.png"},
	{"common-cont-logo", "resources/common/cont/logo.png"},
	{"images-rank4bg", "resources/images/rank4bg.webp"},
	{"images-rank5bg", "resources/images/rank5bg.webp"},
}

// Gacha draws warp records the way StarRail-plugin's gatcha/new does: the
// overview includes collaboration warps and shows five-stars and holdings;
// named warps show four- and five-stars grouped by banner. GachaStat draws
// gacha-stat's per-version statistics.
func Gacha(context app.ImageContext, image app.GachaImage) (app.Image, bool) {
	if image.Command == "gacha-stat" {
		return GachaStat(context, image)
	}
	pools, _ := gachaSelection(image.Word)
	// StarRail-plugin uses 联动 for the character collaboration warp.
	if slices.Equal(pools, []string{"21", "22"}) {
		pools = []string{"21"}
		if strings.Contains(image.Word, "光锥") || strings.Contains(image.Word, "武器") {
			pools = []string{"22"}
		}
	}
	overview := pools == nil
	now := context.Now.In(chinaTime)
	var banners, collabs []app.PoolInfo
	if context.Game.Data != nil {
		banners = context.Game.Data.Resources.Pools
		collabs = context.Game.Data.Resources.CollabPools
	}
	within := func(banner app.PoolInfo, at time.Time) bool {
		from, errFrom := time.ParseInLocation(timeLayout, banner.From, chinaTime)
		to, errTo := time.ParseInLocation(timeLayout, banner.To, chinaTime)
		return errFrom == nil && errTo == nil && at.After(from) && at.Before(to)
	}
	current := []app.PoolInfo{}
	for _, banner := range banners {
		if within(banner, now) {
			current = append(current, banner)
		}
	}
	firstTime, lastTime := "2077-06-06 00:00:00", "2007-06-06 00:00:00"
	if len(current) > 0 {
		firstTime, lastTime = current[0].To, current[0].From
	}
	allTotal := holdingTotal(context.Catalog)
	hold := [2]int{}
	holdRate := [2]int{}
	pity := map[string][2]int{"1": {}, "2": {}, "11": {}, "12": {}, "21": {}, "22": {}}
	currentTotal := 0
	resources := []rayleabot.RenderImageResource{}
	icons := map[string]string{}
	icon := func(record gacha.Record) string {
		if id, seen := icons[record.ItemID]; seen {
			return id
		}
		path := "resources/gatcha/images/char/" + record.ItemID + ".png"
		if record.ItemType == "光锥" {
			path = "resources/panel/resources/weapon/" + record.ItemID + ".png"
		}
		id := "item-" + record.ItemID
		if resource, ok := context.ArtworkResource(id, "starrail-plugin", path); ok {
			resources = append(resources, resource)
		} else {
			id = ""
		}
		icons[record.ItemID] = id
		return id
	}
	filterRank := 5
	if !overview {
		filterRank = 4
	}
	lists := []any{}
	counts := map[string]int{}
	for _, pool := range gachaPools {
		// The overview reads every warp but the departure warp.
		if overview && pool.id == "2" || !overview && !slices.Contains(pools, pool.id) {
			continue
		}
		records := []gacha.Record{}
		for _, record := range image.Archive.Records {
			if record.GachaType == pool.id {
				records = append(records, record)
			}
		}
		if len(records) == 0 {
			continue
		}
		counts[pool.id] = len(records)
		sort.Slice(records, func(i, j int) bool {
			if len(records[i].ID) != len(records[j].ID) {
				return len(records[i].ID) < len(records[j].ID)
			}
			return records[i].ID < records[j].ID
		})
		type entry struct {
			item  map[string]any
			group string
			label map[string]any
		}
		entries := []entry{}
		limits := map[int]int{4: 0, 5: 0}
		for _, record := range records {
			rank, _ := strconv.Atoi(record.Rank)
			at, _ := time.ParseInLocation(timeLayout, record.Time, chinaTime)
			if record.Time < firstTime {
				firstTime = record.Time
			}
			if record.Time > lastTime {
				lastTime = record.Time
			}
			// Holdings count every four- and five-star; upstream's rate uses
			// the count before this pull.
			if rank == 4 || rank == 5 {
				index := rank - 4
				if allTotal > 0 {
					holdRate[index] = int(float64(hold[index])/float64(allTotal)*100 + 0.5)
				}
				hold[index]++
			}
			limits[4]++
			limits[5]++
			until := limits[rank]
			if rank == 4 || rank == 5 {
				limits[rank] = 0
			}
			inCurrent := false
			for _, banner := range current {
				inCurrent = inCurrent || within(banner, at)
			}
			if inCurrent {
				currentTotal++
			}
			up := false
			group, label := "", map[string]any{}
			matched := []app.PoolInfo{}
			if pool.id == "21" || pool.id == "22" {
				for _, banner := range collabs {
					if banner.GachaType == pool.id && within(banner, at) && (len(matched) == 0 || banner.From > matched[0].From) {
						matched = []app.PoolInfo{banner}
					}
				}
			} else {
				var latest *app.PoolInfo
				for _, banner := range banners {
					if within(banner, at) {
						matched = append(matched, banner)
					}
					if banner.From < record.Time && (latest == nil || banner.From > latest.From) {
						latest = &banner
					}
				}
				if len(matched) == 0 && latest != nil {
					to, err := time.ParseInLocation(timeLayout, latest.To, chinaTime)
					if err == nil && at.Sub(to) <= 24*time.Hour {
						matched = append(matched, *latest)
					}
				}
			}
			for _, banner := range matched {
				if group == "" {
					group = banner.Version + banner.Half + banner.Name
					label = map[string]any{"version": banner.Version, "half": banner.Half, "name": banner.Name, "from": banner.From, "to": banner.To}
				}
				for _, name := range append(append([]string{}, banner.Characters5...), banner.Weapons5...) {
					up = up || rank == 5 && name == record.Name
				}
			}
			class := "char rank" + record.Rank
			if record.ItemType == "光锥" {
				class = "cones rank" + record.Rank
			}
			item := map[string]any{"class": class, "up": up, "until": until, "name": record.Name, "show": rank >= filterRank}
			if rank >= filterRank {
				item["icon"] = icon(record)
			}
			entries = append(entries, entry{item: item, group: group, label: label})
		}
		pity[pool.id] = [2]int{limits[4], limits[5]}
		// Upstream lists the newest first and groups banners in that order.
		display := []any{}
		groups := []any{}
		groupIndex := map[string]int{}
		for index := len(entries) - 1; index >= 0; index-- {
			entry := entries[index]
			display = append(display, entry.item)
			position, seen := groupIndex[entry.group]
			if !seen {
				position = len(groups)
				groupIndex[entry.group] = position
				groups = append(groups, map[string]any{"pool": entry.label, "total": 0, "records": []any{}})
			}
			group := groups[position].(map[string]any)
			group["total"] = group["total"].(int) + 1
			group["records"] = append(group["records"].([]any), entry.item)
		}
		lists = append(lists, map[string]any{"type": pool.id, "name": pool.name, "records": display, "groups": groups})
	}
	if len(lists) == 0 {
		return app.Image{}, false
	}
	for _, font := range [][2]string{{"SR", "resources/common/font/SR.ttf"}} {
		if resource, ok := context.ArtworkResource(font[0], "starrail-plugin", font[1]); ok {
			resources = append(resources, resource)
		}
	}
	for _, item := range gachaArtwork {
		if resource, ok := context.ArtworkResource(item[0], "starrail-plugin", item[1]); ok {
			resources = append(resources, resource)
		}
	}
	total := 0
	for _, count := range counts {
		total += count
	}
	pities := []any{map[string]any{"count": pity["11"][1], "label": "角色未出五星"}, map[string]any{"count": pity["12"][1], "label": "光锥未出五星"}}
	if slices.Equal(pools, []string{"21"}) {
		pities = []any{map[string]any{"count": pity["21"][1], "label": "联动角色未出五星"}}
	}
	if slices.Equal(pools, []string{"22"}) {
		pities = []any{map[string]any{"count": pity["22"][1], "label": "联动光锥未出五星"}}
	}
	return app.Image{Template: "gacha", Data: map[string]any{
		"uid": image.UID, "overview": overview,
		"total":         total,
		"current_total": currentTotal,
		"pities":        pities,
		"hold":          hold, "hold_rate": holdRate,
		"first_time": firstTime, "last_time": lastTime,
		"lists": lists,
	}, Resources: resources}, true
}

// holdingTotal counts the catalog's four- and five-star characters and light
// cones, the total StarRail-plugin's holding rates divide by. As upstream it
// leaves out the Trailblazer (IDs from 8000); an enhanced character, its ID
// with 2 for the leading 1 (see build.go), is the same character.
func holdingTotal(catalog app.Catalog) int {
	total := 0
	for _, entry := range catalog.Entries {
		if entry.Rarity != 4 && entry.Rarity != 5 {
			continue
		}
		if entry.Kind == "weapon" || entry.Kind == "character" && !strings.HasPrefix(entry.ID, "2") && !strings.HasPrefix(entry.ID, "8") {
			total++
		}
	}
	return total
}
