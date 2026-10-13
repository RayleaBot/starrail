package app

import (
	"slices"
	"strings"
)

func overseasGameRegion(region string) bool {
	return slices.Contains([]string{"prod_official_usa", "prod_official_eur", "prod_official_asia", "prod_official_cht"}, region)
}
func syncRegionAllowed(region string) bool {
	return overseasGameRegion(region)
}

// uidRegion is the region a UID's first digits name, as miao's getServer.
func uidRegion(uid string) string {
	if len(uid) > 8 {
		switch uid[:len(uid)-8] {
		case "5":
			return "prod_qd_cn"
		case "6":
			return "prod_official_usa"
		case "7":
			return "prod_official_eur"
		case "8", "18":
			return "prod_official_asia"
		case "9":
			return "prod_official_cht"
		}
	}
	return "prod_gf_cn"
}

// publicRegions are the region names 公开信息 takes besides the codes.
var publicRegions = map[string]string{"官服": "prod_gf_cn", "国服": "prod_gf_cn", "b服": "prod_qd_cn", "渠道服": "prod_qd_cn", "prod_gf_cn": "prod_gf_cn", "prod_qd_cn": "prod_qd_cn"}

// publicRegion uses the named region, or the one the UID's first digits name.
func publicRegion(uid, name string) (string, bool) {
	if name == "" {
		return uidRegion(uid), true
	}
	region, ok := publicRegions[strings.ToLower(name)]
	return region, ok
}

// regionTimezone is the UTC offset a region's records are written in.
func regionTimezone(region string) int {
	switch region {
	case "prod_official_usa":
		return -5
	case "prod_official_eur":
		return 1
	}
	return 8
}
