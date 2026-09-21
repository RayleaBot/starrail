package images

import (
	"encoding/json"
	"regexp"
	"sort"
	"strconv"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	gamekit "github.com/RayleaBot/game-plugin-kit"
	"github.com/RayleaBot/game-plugin-kit/gacha"
)

// gachaWords are stripped from the command word before picking the pool, as
// StarRail-plugin's gatcha command does.
var gachaWords = regexp.MustCompile(`#|\*|＊|星铁|星轨|崩铁|星穹铁道|穹批|跃迁|抽卡|记录|分析`)

// gachaPools are the warp pools upstream reports, in its order.
var gachaPools = []struct{ id, name string }{{"1", "群星跃迁"}, {"2", "新手跃迁"}, {"11", "限定跃迁"}, {"12", "光锥跃迁"}}

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
// overview (all pulls, this banner's pulls, pity on the limited pools and,
// for the overview, holdings) and either every pool's five-stars or one
// pool's four- and five-stars grouped by banner.
func Gacha(context gamekit.ImageContext, image gamekit.GachaImage) (gamekit.Image, bool) {
	kind := 0
	switch gachaWords.ReplaceAllString(image.Word, "") {
	case "角色":
		kind = 11
	case "武器", "光锥":
		kind = 12
	case "常驻":
		kind = 1
	case "新手":
		kind = 2
	}
	now := context.Now.In(chinaTime)
	var banners []gamekit.PoolInfo
	if context.Game.Data != nil {
		banners = context.Game.Data.Resources.Pools
	}
	within := func(banner gamekit.PoolInfo, at time.Time) bool {
		from, errFrom := time.ParseInLocation(timeLayout, banner.From, chinaTime)
		to, errTo := time.ParseInLocation(timeLayout, banner.To, chinaTime)
		return errFrom == nil && errTo == nil && at.After(from) && at.Before(to)
	}
	current := []gamekit.PoolInfo{}
	for _, banner := range banners {
		if within(banner, now) {
			current = append(current, banner)
		}
	}
	firstTime, lastTime := "2077-06-06 00:00:00", "2007-06-06 00:00:00"
	if len(current) > 0 {
		firstTime, lastTime = current[0].To, current[0].From
	}
	characterTotals, coneTotals := holdingTotals(context)
	allTotal := characterTotals[0] + characterTotals[1] + coneTotals[0] + coneTotals[1]
	hold := [2]int{}
	holdRate := [2]int{}
	pity := map[string][2]int{"1": {}, "2": {}, "11": {}, "12": {}}
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
	if kind != 0 {
		filterRank = 4
	}
	lists := []any{}
	counts := map[string]int{}
	for _, pool := range gachaPools {
		wanted := (kind == 0 && pool.id != "2") || strconv.Itoa(kind) == pool.id
		if !wanted {
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
			for _, banner := range banners {
				if !within(banner, at) {
					continue
				}
				if group == "" {
					group = banner.Version + banner.Half
					label = map[string]any{"version": banner.Version, "half": banner.Half, "from": banner.From, "to": banner.To}
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
		return gamekit.Image{}, false
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
	return gamekit.Image{Template: "gacha", Data: map[string]any{
		"uid": image.UID, "overview": kind == 0,
		"total":         counts["1"] + counts["11"] + counts["12"],
		"current_total": currentTotal,
		"pity_role":     pity["11"][1], "pity_cone": pity["12"][1],
		"hold": hold, "hold_rate": holdRate,
		"first_time": firstTime, "last_time": lastTime,
		"lists": lists,
	}, Resources: resources}, true
}

// holdingTotals counts the four- and five-star characters (without the
// Trailblazer) and light cones in StarRail-plugin's base data, the totals its
// holding rates divide by.
func holdingTotals(context gamekit.ImageContext) ([2]int, [2]int) {
	count := func(name string, skip []string) [2]int {
		var result [2]int
		if context.Artwork == nil {
			return result
		}
		raw, err := context.Artwork.Open("starrail-plugin", "resources/baseData/"+name+".json")
		if err != nil {
			return result
		}
		var entries map[string]struct {
			Rarity any `json:"rarity"`
		}
		if json.Unmarshal(raw, &entries) != nil {
			return result
		}
		for id, entry := range entries {
			skipped := false
			for _, value := range skip {
				skipped = skipped || id == value
			}
			if skipped {
				continue
			}
			switch gamekit.Text(entry.Rarity) {
			case "4":
				result[0]++
			case "5":
				result[1]++
			}
		}
		return result
	}
	return count("characters", []string{"8001", "8002", "8003", "8004"}), count("light_cones", nil)
}
