// Package panelapi runs what StarRail-plugin's own panel service (apps/panel.js)
// runs on, from its pinned snapshot: the request signing script its getSign
// evaluates. scripts/bundle-panel-api.mjs copies the files.
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

	"github.com/RayleaBot/plugin-starrail/internal/app"
	"github.com/RayleaBot/plugin-starrail/internal/reference"
)

// signScript is resources/fuck/fuck: the signing function in base64.
//
//go:embed sign.b64
var signScript string

var signer = sync.OnceValues(func() (*reference.Function, error) {
	source, err := base64.StdEncoding.DecodeString(signScript)
	if err != nil {
		return nil, err
	}
	return reference.CompileFunction("sign", string(source))
})

// Scripts are the pinned scripts the plugin's panel service commands run.
var Scripts = app.PanelAPIScripts{Sign: sign}

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
