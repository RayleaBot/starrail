package reference

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func md5Hex(text string) string {
	sum := md5.Sum([]byte(text))
	return hex.EncodeToString(sum[:])
}

// A stub in the shape of StarRail-plugin's signing script: it hashes the UID
// with the time and a key through the md5 helper the call passes in.
func TestFunctionRunsWithTheHelpersItIsGiven(t *testing.T) {
	sign, err := CompileFunction("sign", "function getKey(e){let o=Math.round((new Date).getTime()/1e3),t=md5(`${e},${o},stub`);return `${o},${t}`}")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := sign.Call(context.Background(), map[string]any{"md5": md5Hex}, "100000001")
	if err != nil {
		t.Fatal(err)
	}
	var header string
	if err = json.Unmarshal(raw, &header); err != nil {
		t.Fatal(err)
	}
	stamp, hash, _ := strings.Cut(header, ",")
	if hash != md5Hex("100000001,"+stamp+",stub") {
		t.Fatalf("header %q is not the stub's hash", header)
	}
}

func TestFunctionHasNoHostAccessAndKeepsNoState(t *testing.T) {
	probe, err := CompileFunction("probe", "function(){globalThis.calls=(globalThis.calls||0)+1;return [typeof require,typeof process,typeof fetch,typeof XMLHttpRequest,typeof md5,calls].join()}")
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		raw, err := probe.Call(context.Background(), nil)
		if err != nil {
			t.Fatal(err)
		}
		if string(raw) != `"undefined,undefined,undefined,undefined,undefined,1"` {
			t.Fatalf("the script saw %s", raw)
		}
	}
}

func TestFunctionIsBounded(t *testing.T) {
	loop, err := CompileFunction("loop", "function(){for(;;){}}")
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if _, err = loop.Call(context.Background(), nil); err == nil {
		t.Fatal("an endless script completed")
	}
	if time.Since(start) > 3*time.Second {
		t.Fatal("the call was not bounded")
	}
	if _, err = CompileFunction("broken", "function("); err == nil {
		t.Fatal("a broken script compiled")
	}
	value, err := CompileFunction("value", "42")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = value.Call(context.Background(), nil); err == nil {
		t.Fatal("a script without a function was called")
	}
}
