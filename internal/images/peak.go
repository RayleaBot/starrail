package images

import (
	gamekit "github.com/RayleaBot/game-plugin-kit"
)

// peakArtwork maps the fixed images of the Anomaly Arbitration pages to their
// StarRail-plugin paths.
var peakArtwork = [][2]string{
	{"SR", "resources/common/font/SR.ttf"},
	{"tttgbnumber", "resources/common/font/tttgbnumber.ttf"},
	{"rogue-star", "resources/common/cont/rogue-star.png"},
	{"boss-star", "resources/challenge/images/boss_star.ae0258cf.png"},
	{"character-r4", "resources/challenge/images/character_r_4.24f329b7.png"},
	{"character-r5", "resources/challenge/images/character_r_5.99d42eb7.png"},
	{"block-bg", "resources/challenge/images/block_bg.b669cdc3.png"},
	{"normal-boss-bg", "resources/challenge/images/normal_boss_bg.047e5b19.png"},
	{"hell-boss-bg", "resources/challenge/images/hell_boss_bg.ca675173.png"},
	{"mob-bg", "resources/challenge/images/xsg_bg.962ba3e3.png"},
}

// Peak draws Anomaly Arbitration the way StarRail-plugin does: index_peak for
// the current or last run, and peak_recent with the three recent runs side by
// side for 往期. Each run shows its rank and theme, the King's and Knights'
// stars, then the King and every Knight with rounds, stars, team and buff.
func Peak(context gamekit.ImageContext, result gamekit.QueryResult) (gamekit.Image, bool) {
	records, _ := result.Data["challenge_peak_records"].([]any)
	period := gamekit.Int(context.Input["schedule_type"])
	switch period {
	case 2:
		// The last run is the second of the recent runs.
		if len(records) < 2 {
			return gamekit.Image{}, false
		}
		records = records[1:2]
	case 3:
		records = records[:min(len(records), 3)]
	default:
		records = records[:min(len(records), 1)]
	}
	if len(records) == 0 {
		return gamekit.Image{}, false
	}

	resources := newChallengeResources(context)
	for _, item := range peakArtwork {
		resources.local(item[0], item[1])
	}
	if period == 3 {
		resources.local("recent-bg", "resources/challenge/images/pc_bg.46428ec0.png")
	}
	stars := func(id string, count int) []any {
		list := []any{}
		for range max(count, 0) {
			list = append(list, id)
		}
		return list
	}

	// Upstream shows the best-record brief on every run it draws.
	brief, _ := result.Data["challenge_peak_best_record_brief"].(map[string]any)
	runs := []any{}
	for _, raw := range records {
		record, _ := raw.(map[string]any)
		group, _ := record["group"].(map[string]any)
		from, _ := challengeTime(group["begin_time"])
		to, _ := challengeTime(group["end_time"])
		bossRecord, _ := record["boss_record"].(map[string]any)
		rank := gamekit.Text(group["theme_pic_path"])
		battles := ""
		if brief != nil {
			rank, battles = gamekit.Text(brief["challenge_peak_rank_icon"]), gamekit.Text(brief["total_battle_num"])
		} else if bossRecord != nil {
			rank = gamekit.Text(bossRecord["challenge_peak_rank_icon"])
		}

		info, _ := record["boss_info"].(map[string]any)
		cleared, _ := bossRecord["has_challenge_record"].(bool)
		hard, _ := bossRecord["hard_mode"].(bool)
		boss := map[string]any{"hell": cleared && hard, "cleared": cleared, "name": gamekit.Text(info["name_mi18n"]), "icon": resources.official(info["icon"])}
		if cleared {
			boss["rounds"], boss["stars"], boss["time"] = gamekit.Text(bossRecord["round_num"]), stars("boss-star", gamekit.Int(bossRecord["star_num"])), challengeClock(bossRecord)
			boss["avatars"], boss["buff"] = resources.team(bossRecord), resources.buff(bossRecord)
		}

		mobs := []any{}
		infos, _ := record["mob_infos"].([]any)
		mobRecords, _ := record["mob_records"].([]any)
		for index, rawInfo := range infos {
			info, _ := rawInfo.(map[string]any)
			var mobRecord map[string]any
			if index < len(mobRecords) {
				mobRecord, _ = mobRecords[index].(map[string]any)
			}
			fast, _ := mobRecord["is_fast"].(bool)
			cleared, _ := mobRecord["has_challenge_record"].(bool)
			mob := map[string]any{"title": gamekit.Text(info["name"]) + ": " + gamekit.Text(info["monster_name"]), "icon": resources.official(info["monster_icon"]),
				"fast": fast, "cleared": cleared}
			if cleared && !fast {
				mob["rounds"], mob["stars"], mob["time"], mob["avatars"] = gamekit.Text(mobRecord["round_num"]), stars("rogue-star", gamekit.Int(mobRecord["star_num"])), challengeClock(mobRecord), resources.team(mobRecord)
			}
			mobs = append(mobs, mob)
		}

		runs = append(runs, map[string]any{"begin": from.Format("2006.01.02"), "end": to.Format("2006.01.02"),
			"rank_icon": resources.official(rank), "theme": resources.official(group["theme_pic_path"]),
			"boss_stars": gamekit.Text(record["boss_stars"]), "mob_stars": gamekit.Text(record["mob_stars"]),
			"version": gamekit.Text(group["game_version"]), "name": gamekit.Text(group["name_mi18n"]), "battles": battles,
			"boss": boss, "mobs": mobs})
	}

	template, tag := "peak", "本期"
	switch period {
	case 2:
		tag = "上期"
	case 3:
		template, tag = "peak-recent", "最近三期"
	}
	return gamekit.Image{Template: template, Data: map[string]any{"uid": result.Role.UID, "tag": tag, "records": runs}, Resources: resources.List}, true
}
