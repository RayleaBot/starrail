package app

import "slices"

func overseasGameRegion(region string) bool {
	return slices.Contains([]string{"prod_official_usa", "prod_official_eur", "prod_official_asia", "prod_official_cht"}, region)
}
func syncRegionAllowed(region string) bool {
	return overseasGameRegion(region)
}
