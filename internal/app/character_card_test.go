package app

import "testing"

func TestCardQueryFollowsMiaoCheck(t *testing.T) {
	for text, want := range map[string][2]string{
		"希儿":            {"希儿", ""},
		"希儿卡片100000001": {"希儿", "100000001"},
		"老婆卡芙卡":         {"卡芙卡", ""},
		"希儿 187654321":  {"希儿", "187654321"},
		"希儿1876543210":  {"希儿", "1876543210"},
	} {
		if name, uid := cardQuery(text); name != want[0] || uid != want[1] {
			t.Errorf("cardQuery(%q) = %q, %q; want %q, %q", text, name, uid, want[0], want[1])
		}
	}
}

// 照片 and the card draw from what miao's CharImg.getCardImg reads: the
// character's photos, uploaded or imported as its photos, and the downloaded
// character-img directory; other images of the library are not read.
func TestCardImagesReadTheCharacterPhotos(t *testing.T) {
	a := &App{Media: &MediaStore{Directory: t.TempDir()}}
	defer a.Close()
	for _, entry := range []MediaEntry{
		{Title: "希儿照片1", Category: "photo", CatalogID: "1102", License: "聊天上传"},
		{Title: "希儿图鉴", Category: "character", CatalogID: "1102", License: "本人有权使用"},
		{Title: "卡芙卡照片1", Category: "photo", CatalogID: "1005", License: "聊天上传"},
	} {
		if _, err := a.Media.add(entry, pngFixture(t)); err != nil {
			t.Fatal(err)
		}
	}
	media, photos := a.cardImages(Entry{ID: "1102", Name: "希儿"})
	if len(media) != 1 || media[0].Title != "希儿照片1" || len(photos) != 0 {
		t.Fatalf("media = %+v, photos = %+v", media, photos)
	}
}
