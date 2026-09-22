package app

import (
	"context"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/RayleaBot/plugin-starrail/internal/localdata"
)

type GuideSource struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Collections []string `json:"collections"`
}

var guideSources = []GuideSource{{"1", "初始镜像", []string{"1996095"}}, {"2", "小橙子阿", []string{"1998643"}}, {"3", "星穹中心", []string{"2029394", "2009142", "2038092"}}}

type GuideConfig struct {
	Revision uint64 `json:"revision"`
	Default  string `json:"default_source"`
}
type GuideSettings struct {
	mu   sync.Mutex
	Path string
}

func (s *GuideSettings) Manage(action string, input map[string]any) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := GuideConfig{Default: "1"}
	if err := localdata.Read(s.Path, &v); err != nil {
		return nil, err
	}
	if action == "guides.configure" {
		var q GuideConfig
		if decodeObject(input, &q) != nil || !slices.ContainsFunc(guideSources, func(s GuideSource) bool { return s.ID == q.Default }) {
			return nil, gameError("input_invalid", "攻略来源无效。")
		}
		if q.Revision != v.Revision {
			return nil, gameError("settings_changed", "攻略设置已变化，请刷新。")
		}
		v.Default = q.Default
		v.Revision++
		if err := localdata.Write(s.Path, v); err != nil {
			return nil, err
		}
	}
	return map[string]any{"sources": guideSources, "settings": v}, nil
}
func (c PublicContentClient) guides(ctx context.Context, q ContentQuery) (map[string]any, error) {
	sources := guideSources
	index := slices.IndexFunc(sources, func(v GuideSource) bool { return v.ID == q.Source })
	if index < 0 || q.CollectionOffset < 0 || q.CollectionOffset >= len(sources[index].Collections) || len([]rune(q.Query)) > 100 || q.Offset < 0 {
		return nil, gameError("input_invalid", "攻略来源或筛选无效。")
	}
	source := sources[index]
	gid := bbsGID
	params := url.Values{"gids": {gid}, "order_type": {"2"}, "collection_id": {source.Collections[q.CollectionOffset]}}
	data, err := c.get(ctx, "https://bbs-api.mihoyo.com/post/wapi/getPostFullInCollection?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	raw, ok := data["posts"].([]any)
	if !ok || len(raw) > 2000 {
		return nil, gameError("public_invalid", "攻略合集格式暂不兼容。")
	}
	matches := []any{}
	for _, v := range raw {
		p := asObject(asObject(v)["post"])
		if p == nil {
			p = asObject(v)
		}
		if q.Query == "" || strings.Contains(publicText(asText(p["subject"])), q.Query) {
			matches = append(matches, v)
		}
	}
	start, end := min(q.Offset, len(matches)), min(q.Offset+10, len(matches))
	items := []PublicPost{}
	for _, v := range matches[start:end] {
		p, err := normalizePublicPost(v, true)
		if err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	next := -1
	if end < len(matches) {
		next = end
	}
	return map[string]any{"items": items, "total": len(matches), "source": source, "collection_offset": q.CollectionOffset, "next_offset": next, "fetched_at_ms": time.Now().UnixMilli()}, nil
}
