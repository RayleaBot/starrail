package images

import (
	"strconv"

	gamekit "github.com/RayleaBot/game-plugin-kit"
)

// artifactListArtwork maps the images named in the converted stylesheets
// (miao's common, character/profile-detail and character/artis-list) that the
// list shows to their paths.
var artifactListArtwork = [][2]string{
	{"Number", "resources/common/font/tttgbnumber.woff"},
	{"NZBZ", "resources/common/font/NZBZ.woff"},
	{"YS", "resources/common/font/HYWH-65W.woff"},
	{"character-imgs-up-num-icon0", "resources/character/imgs/up-num-icon0.png"},
	{"character-imgs-up-num-icon1", "resources/character/imgs/up-num-icon1.png"},
	{"character-imgs-up-num-icon2", "resources/character/imgs/up-num-icon2.png"},
	{"common-bg-bg-hydro", "resources/common/bg/bg-hydro.webp"},
}

// ArtifactList draws 遗器列表 the way miao's character/artis-list does:
// each piece with its icon, name, score and
// grade, the main stat and the substats marked by the wearer's weights.
func ArtifactList(context gamekit.ImageContext, list gamekit.ArtifactListImage) (gamekit.Image, bool) {
	resources := &gamekit.ImageResources{Context: context}
	for _, item := range artifactListArtwork {
		resources.Artwork(item[0], "miao-plugin", item[1])
	}
	artis := []any{}
	for index, listed := range list.Pieces {
		detail := listed.Panel.ScoreDetail
		class := func(key string) string {
			switch w := detail.Weights[key]; {
			case w > 79.9:
				return "great"
			case w > 0:
				return "useful"
			}
			return "nouse"
		}
		attrs := []any{}
		for _, attr := range listed.Scored.Attrs {
			attrs = append(attrs, map[string]any{"class": class(attr.Key), "eff": attr.Eff, "up": attr.UpNum, "title": detail.Titles[attr.Key], "value": attr.Value})
		}
		piece := strconv.Itoa(index)
		artis = append(artis, map[string]any{
			"icon": resources.Artwork("piece-"+piece, "miao-plugin", "resources/meta-sr/artifact/"+listed.Equipment.SetName+"/arti-"+strconv.Itoa(listed.Equipment.Slot)+".webp"),
			"name": listed.Equipment.Name, "mark": listed.Scored.Mark, "grade": listed.Scored.Grade,
			"main":  map[string]any{"title": detail.Titles[listed.Scored.Main.Key], "value": listed.Scored.Main.Value},
			"attrs": attrs,
		})
	}
	// miao has no side portrait for Star Rail characters, so the list shows
	// none.
	return gamekit.Image{Template: "artifact-list", Data: map[string]any{"uid": list.UID, "game": "sr", "artis": artis}, Resources: resources.List}, true
}
