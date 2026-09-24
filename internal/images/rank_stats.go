package images

import (
	"github.com/RayleaBot/plugin-starrail/internal/app"
)

// rankStatsArtwork maps the miao images the rank statistics stylesheet names
// (the panel page's frame and card) to their repository paths.
var rankStatsArtwork = [][2]string{
	{"common-cont-card-bg", "resources/common/cont/card-bg.png"},
	{"Number", "resources/common/font/tttgbnumber.woff"},
	{"NZBZ", "resources/common/font/NZBZ.woff"},
	{"YS", "resources/common/font/HYWH-65W.woff"},
}

// RankStats draws <角色>排名统计 with the 排名统计 block of the panel page
// (ark-plugin's profile-detail) on the character's element background: its
// damage and relic distributions side by side with no panel placed on them,
// the calculation the damage ranks, and under each chart the UIDs ark
// counted, as its graph/stats writes them.
func RankStats(context app.ImageContext, image app.RankStatsImage) (app.Image, bool) {
	if image.Damage == nil && image.Artis == nil {
		return app.Image{}, false
	}
	resources := &app.ImageResources{Context: context}
	for _, item := range rankStatsArtwork {
		resources.Artwork(item[0], "miao-plugin", item[1])
	}
	elem := elementClasses[image.Character.Element]
	resources.Artwork("common-bg-bg-"+elem, "miao-plugin", "resources/common/bg/bg-"+elem+".webp")
	return app.Image{Template: "rank-stats", Data: map[string]any{"elem": elem, "name": image.Character.Name, "title": image.DamageTitle,
		"charts": rankCharts(image.Damage, image.Artis), "totals": image.Totals[:]}, Resources: resources.List}, true
}
