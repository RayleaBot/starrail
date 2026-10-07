// Package panelapi runs what StarRail-plugin's own panel service (apps/panel.js)
// runs on, from its pinned snapshot: the request signing script its getSign
// evaluates, and the damage module and eidolon table its panel page draws on.
// scripts/bundle-panel-api.mjs copies and bundles the files.
package panelapi

import (
	"context"
	"crypto/md5"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync"

	"github.com/RayleaBot/starrail/internal/app"
	"github.com/RayleaBot/starrail/internal/reference"
)

// signScript is resources/fuck/fuck: the signing function in base64.
//
//go:embed sign.b64
var signScript string

// damageScript is utils/damage bundled into an expression whose value is
// damage(charinfo).
//
//go:embed damage.js
var damageScript string

// rankSkillUp is utils/damage/json/avatarRankSkillUp.json.
//
//go:embed rank-skill-up.json
var rankSkillUp []byte

var (
	signer = sync.OnceValues(func() (*reference.Function, error) {
		source, err := base64.StdEncoding.DecodeString(signScript)
		if err != nil {
			return nil, err
		}
		return reference.CompileFunction("sign", string(source))
	})
	calculator = sync.OnceValues(func() (*reference.Function, error) {
		return reference.CompileFunction("damage.js", damageScript)
	})
)

// Scripts are the pinned scripts and data the plugin's panel service
// commands run on.
func Scripts() app.PanelAPIScripts {
	var raises map[string][]app.RankSkillUp
	if err := json.Unmarshal(rankSkillUp, &raises); err != nil {
		panic(err) // the embedded table is fixed at build time
	}
	return app.PanelAPIScripts{Sign: sign, Damage: damage, RankSkillUp: raises}
}

// sign is getSign: the x-request-sr header of a UID, which the pinned script
// computes from the UID and the time. The script sees only md5, the hash
// function its module imports.
func sign(ctx context.Context, uid string) (string, error) {
	function, err := signer()
	if err != nil {
		return "", err
	}
	raw, err := function.Call(ctx, map[string]any{"md5": md5Hex}, uid)
	if err != nil {
		return "", err
	}
	var header string
	if err = json.Unmarshal(raw, &header); err != nil || header == "" {
		return "", errors.New("the signing script returned no header")
	}
	return header, nil
}

// md5Hex is the md5 package's md5: the hex digest of a string's UTF-8 bytes.
func md5Hex(text string) string {
	sum := md5.Sum([]byte(text))
	return hex.EncodeToString(sum[:])
}

// damage runs the damage module on a character as the panel page passes it,
// with its first skills raised by the eidolons and its traces as skillTree.
// The module returns nothing for characters it does not calculate.
func damage(ctx context.Context, character any) ([]app.APIDamage, error) {
	function, err := calculator()
	if err != nil {
		return nil, err
	}
	raw, err := function.Call(ctx, nil, character)
	if err != nil {
		return nil, err
	}
	var result *struct {
		Damage []app.APIDamage `json:"damage"`
	}
	if err = json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	if result == nil || result.Damage == nil {
		return nil, errors.New("the damage module does not calculate this character")
	}
	return result.Damage, nil
}
