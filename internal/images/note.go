// Package images draws this plugin's replies with its own templates, following
// the StarRail-plugin images for the same commands.
package images

import (
	"fmt"
	"strconv"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	gamekit "github.com/RayleaBot/game-plugin-kit"
)

// Builders lists the image builders by the operation they draw.
func Builders() map[string]gamekit.ImageBuilder {
	return map[string]gamekit.ImageBuilder{"starrail.note": Note}
}

var weekdays = [...]string{"星期日", "星期一", "星期二", "星期三", "星期四", "星期五", "星期六"}

// Note draws the real-time note the way StarRail-plugin's new_note does:
// the requester's avatar and today's check-in, trailblaze power with its
// recovery time, daily training and simulated universe scores, and each
// assignment with its remaining time.
func Note(context gamekit.ImageContext, result gamekit.QueryResult) (gamekit.Image, bool) {
	data := result.Data
	if _, ok := data["current_stamina"]; !ok {
		return gamekit.Image{}, false
	}
	now := context.Now.In(time.FixedZone("UTC+8", 8*3600))
	current, maximum := gamekit.Int(data["current_stamina"]), gamekit.Int(data["max_stamina"])
	stamina := map[string]any{"current": current, "max": maximum, "full": current == maximum, "progress": percent(current, maximum)}
	if current != maximum {
		seconds := gamekit.Int(data["stamina_recover_time"])
		end := now.Add(time.Duration(seconds) * time.Second)
		day := "今日"
		if end.Day() != now.Day() {
			day = "明日"
		}
		stamina["wait"] = duration(seconds, "%02d小时%02d分钟") + " |"
		stamina["day"] = "[" + day + "]"
		stamina["clock"] = end.Format("15:04") + "完全恢复"
	}

	resources := []rayleabot.RenderImageResource{}
	for _, background := range [][3]string{{"container-bg", "resources/common/cont/bkg2.jpg"}, {"article-bg", "resources/note/resources/background.png"}} {
		if resource, ok := context.ArtworkResource(background[0], "starrail-plugin", background[1]); ok {
			resources = append(resources, resource)
		}
	}
	expeditions := []any{}
	for index, raw := range asList(data["expeditions"]) {
		item, _ := raw.(map[string]any)
		remaining := gamekit.Int(item["remaining_time"])
		avatars := []any{}
		for slot, avatar := range asList(item["avatars"]) {
			id := fmt.Sprintf("expedition-%d-%d", index, slot)
			if url := gamekit.Text(avatar); url != "" {
				resources = append(resources, rayleabot.RenderImageResource{ID: id, URL: url})
				avatars = append(avatars, id)
			}
		}
		expeditions = append(expeditions, map[string]any{
			"name":      gamekit.Text(item["name"]),
			"remaining": duration(remaining, "%02d时%02d分"),
			"finished":  remaining == 0,
			"avatars":   avatars,
			"progress":  percent(72000-remaining, 72000),
		})
	}

	accepted := data["accepted_epedition_num"]
	if accepted == nil {
		accepted = data["accepted_expedition_num"]
	}
	signed := false
	if context.Query != nil {
		if sign, err := context.Query("starrail.sign_info", nil); err == nil {
			signed, _ = sign.Data["is_sign"].(bool)
		}
	}
	return gamekit.Image{
		Template: "note",
		Data: map[string]any{
			"nickname":    result.Role.Nickname,
			"uid":         result.Role.UID,
			"level":       result.Role.Level,
			"signed":      signed,
			"time":        now.Format("2006-01-02 15:04:05") + " " + weekdays[now.Weekday()],
			"stamina":     stamina,
			"train":       gamekit.Text(data["current_train_score"]) + "/" + gamekit.Text(data["max_train_score"]),
			"rogue":       gamekit.Text(data["current_rogue_score"]) + "/" + gamekit.Text(data["max_rogue_score"]),
			"accepted":    gamekit.Text(accepted),
			"total":       gamekit.Text(data["total_expedition_num"]),
			"expeditions": expeditions,
		},
		Resources: resources,
	}, true
}

// duration formats a countdown as upstream's moment.utc(...).format, which
// wraps at a day; zero reads as finished.
func duration(seconds int, format string) string {
	if seconds == 0 {
		return "已完成"
	}
	return fmt.Sprintf(format, seconds/3600%24, seconds/60%60)
}

func percent(value, total int) string {
	if total <= 0 {
		return "0%"
	}
	return strconv.FormatFloat(float64(value)*100/float64(total), 'f', 2, 64) + "%"
}

func asList(value any) []any { list, _ := value.([]any); return list }
