package app

import "testing"

func TestPublicProfileRegionFollowsUIDAndNames(t *testing.T) {
	for _, tc := range []struct{ uid, name, want string }{
		{"100000001", "", "prod_gf_cn"},
		{"500000001", "", "prod_qd_cn"},
		{"100000001", "B服", "prod_qd_cn"},
		{"500000001", "官服", "prod_gf_cn"},
		{"100000001", "渠道服", "prod_qd_cn"},
		{"500000001", "国服", "prod_gf_cn"},
		{"100000001", "prod_qd_cn", "prod_qd_cn"},
		{"100000001", "美服", ""},
	} {
		region, ok := publicRegion(tc.uid, tc.name)
		if region != tc.want || ok != (tc.want != "") {
			t.Errorf("%+v: got %q, %v", tc, region, ok)
		}
	}
}
