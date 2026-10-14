package app

import (
	"context"
	"fmt"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// gachaStatForward renders all pages before sending; a render failure leaves
// the normal single-image/text reply available to the caller.
func (a *App) gachaStatForward(ctx context.Context, event *rayleabot.EventContext, image GachaImage, first Image) (bool, error) {
	page := asObject(first.Data["page"])
	total := Int(page["total"])
	if total <= 1 {
		return false, nil
	}
	title := fmt.Sprintf("UID:%s %s统计，共%d期卡池，分%d张图", image.UID, asText(first.Data["pool_name"]), Int(page["count"]), total)
	parts := [][]rayleabot.Segment{{rayleabot.Text(title)}}
	for number := 1; number <= total; number++ {
		drawn := first
		if number > 1 {
			image.Page = number
			var ok bool
			drawn, ok = a.gacha(a.imageContext(ctx), image)
			if !ok {
				return false, nil
			}
		}
		result, err := event.Actions().RenderImage(ctx, rayleabot.RenderImageRequest{Template: drawn.Template, Output: "jpeg", FallbackText: title, Data: drawn.Data, Resources: drawn.Resources})
		path := asText(result["image_path"])
		if err != nil || path == "" {
			return false, nil
		}
		parts = append(parts, []rayleabot.Segment{rayleabot.Image(path)})
	}
	return true, a.sendForward(ctx, event, parts)
}
