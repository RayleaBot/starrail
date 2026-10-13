package images

import (
	"regexp"
	"strings"

	"github.com/RayleaBot/starrail/internal/app"
)

// helpArtwork maps the font and the default theme's artwork the help
// stylesheet names to their StarRail-plugin paths.
var helpArtwork = [][2]string{
	{"YS", "resources/common/font/YS.ttf"},
	{"help-theme-main", "resources/help/theme/default/main.png"},
}

// helpLegend explains the notation every written form uses.
const helpLegend = "< > 换成实际内容 · [ ] 可省略 · | 任选其一"

// helpArgument matches a written form's placeholders: <...> for what to fill
// in, [...] for what may be left out.
var helpArgument = regexp.MustCompile(`<[^<>]*>|\[[^\[\]]*\]`)

// Help draws the help menu: StarRail-plugin's default-theme artwork as the
// header, then each group's commands with their written forms, placeholders
// set apart from the words to type, and description.
func Help(context app.ImageContext, help app.HelpImage) (app.Image, bool) {
	if len(help.Groups) == 0 {
		return app.Image{}, false
	}
	resources := &app.ImageResources{Context: context}
	for _, item := range helpArtwork {
		resources.Artwork(item[0], "starrail-plugin", item[1])
	}
	groups := []any{}
	for _, group := range help.Groups {
		commands := []any{}
		for _, command := range group.Commands {
			commands = append(commands, map[string]any{"forms": helpForms(command.Usage), "desc": command.Description})
		}
		groups = append(groups, map[string]any{"title": group.Title, "commands": commands})
	}
	data := map[string]any{"title": help.Title, "legend": helpLegend, "groups": groups}
	if help.Note != "" {
		data["note"] = help.Note
	}
	return app.Image{Template: "help", Data: data, Resources: resources.List}, true
}

// helpForms splits a usage into its written forms, each a run of text and
// placeholder parts.
func helpForms(usage string) []any {
	forms := []any{}
	for _, form := range strings.Split(usage, " / ") {
		parts, last := []any{}, 0
		for _, match := range helpArgument.FindAllStringIndex(form, -1) {
			if match[0] > last {
				parts = append(parts, map[string]any{"text": form[last:match[0]]})
			}
			parts = append(parts, map[string]any{"text": form[match[0]:match[1]], "arg": true})
			last = match[1]
		}
		if last < len(form) {
			parts = append(parts, map[string]any{"text": form[last:]})
		}
		forms = append(forms, parts)
	}
	return forms
}
