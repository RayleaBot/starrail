package reference

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dop251/goja"
)

// Function is a pinned upstream script whose value is one function, the way
// StarRail-plugin's getSign evaluates its signing script. Every call runs the
// function in a fresh interpreter under a time limit. The interpreter has no
// file system, network or other host object: the script sees only the
// JavaScript built-ins and the helpers the call passes, which are pure
// functions the script expects in its scope.
type Function struct {
	program *goja.Program
}

// CompileFunction compiles source, a JavaScript expression whose value is a
// function.
func CompileFunction(name, source string) (*Function, error) {
	program, err := goja.Compile(name, "("+source+"\n)", false)
	if err != nil {
		return nil, fmt.Errorf("compile %s: %w", name, err)
	}
	return &Function{program: program}, nil
}

// Call runs the function on args, which reach it as parsed JSON, and returns
// its result as JSON.
func (f *Function) Call(ctx context.Context, helpers map[string]any, args ...any) (json.RawMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(args)
	if err != nil || len(encoded) > 512*1024 {
		return nil, errors.New("script input invalid")
	}
	ctx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()
	vm := goja.New()
	vm.SetMaxCallStackSize(512)
	stop := context.AfterFunc(ctx, func() { vm.Interrupt("script canceled") })
	defer stop()
	for name, helper := range helpers {
		if err = vm.Set(name, helper); err != nil {
			return nil, err
		}
	}
	function, err := vm.RunProgram(f.program)
	if err != nil {
		return nil, errors.New("script initialization failed")
	}
	if _, ok := goja.AssertFunction(function); !ok {
		return nil, errors.New("script is not a function")
	}
	_ = vm.Set("pinnedFunction", function)
	_ = vm.Set("argumentsJSON", string(encoded))
	result, err := vm.RunString("JSON.stringify(pinnedFunction.apply(undefined,JSON.parse(argumentsJSON)))")
	if err != nil {
		return nil, fmt.Errorf("script failed: %w", err)
	}
	raw := []byte(result.String())
	if len(raw) > 1024*1024 || !json.Valid(raw) {
		return nil, errors.New("script output invalid")
	}
	return raw, nil
}
