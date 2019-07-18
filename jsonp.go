package jsonp

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// JSON types.
type (
	Any    = interface{}
	Array  = []Any
	Object = map[string]Any
)

// Clone makes a copy of a JSON node.
func Clone(x Any) Any {
	if obj, ok := x.(Object); ok {
		obj2 := make(Object, len(obj))
		for k, v := range obj {
			obj2[k] = Clone(v)
		}
		return obj2
	}
	if arr, ok := x.(Array); ok {
		arr2 := make(Array, len(arr))
		for i := range arr {
			arr2[i] = Clone(arr[i])
		}
		return arr2
	}
	return x
}

// ToPointer converts path to JSON Pointer.
func ToPointer(path []string) string {
	ss := make([]string, len(path)+1)
	for i := range path {
		ss[i+1] = strings.NewReplacer("~", "~0", "/", "~1").Replace(path[i])
	}
	return strings.Join(ss, "/")
}

// ToPath converts JSON Pointer to path.
func ToPath(pointer string) ([]string, error) {
	if pointer == "" {
		return nil, nil
	}
	ss := strings.Split(pointer, "/")
	for i := range ss {
		ss[i] = strings.NewReplacer("~0", "~", "~1", "/").Replace(ss[i])
	}
	if ss[0] != "" {
		return nil, errors.Errorf("invalid JSON Pointer: %q", pointer)
	}
	return ss[1:], nil
}

// Get retrieves a node by path.
func Get(x Any, path []string) (Any, error) {
	return getRecr(x, path, 0)
}

// GetByPointer retrieves node by JSON Pointer.
func GetByPointer(x Any, pointer string) (Any, error) {
	path, err := ToPath(pointer)
	if err != nil {
		return nil, err
	}
	return Get(x, path)
}

func getRecr(x Any, path []string, depth int) (Any, error) {
	if depth >= len(path) {
		return x, nil
	}
	if obj, ok := x.(Object); ok {
		return getRecr(obj[path[depth]], path, depth+1)
	}
	if arr, ok := x.(Array); ok {
		idx, err := strconv.Atoi(path[depth])
		if err != nil {
			return nil, err
		}
		return getRecr(arr[idx], path, depth+1)
	}
	return nil, notArrayOrObjectErr(path[:depth])
}

// Walk applies f to all sub nodes of x.
func Walk(x Any, f func(path []string, x Any)) {
	walkRecr(x, nil, f)
}

func walkRecr(x Any, path []string, f func(path []string, x Any)) {
	f(path, x)
	if obj, ok := x.(Object); ok {
		for k, v := range obj {
			walkRecr(v, append(path, k), f)
		}
	}
	if arr, ok := x.(Array); ok {
		for k, v := range arr {
			walkRecr(v, append(path, strconv.Itoa(k)), f)
		}
	}
}

// WalkByPointer applies f to all sub nodes of x.
func WalkByPointer(x Any, f func(pointer string, x Any)) {
	Walk(x, func(path []string, x Any) {
		f(ToPointer(path), x)
	})
}

// Add inserts value at given position.
func Add(x Any, path []string, value Any) (Any, error) {
	return applyAddRecr(x, path, 0, value)
}

// AddByPointer inserts value at given position.
// The position is specified by JSON Pointer.
func AddByPointer(x Any, pointer string, value Any) (Any, error) {
	path, err := ToPath(pointer)
	if err != nil {
		return nil, err
	}
	return Add(x, path, value)
}

func applyAddRecr(x Any, path []string, depth int, value Any) (Any, error) {
	if len(path[depth:]) == 0 {
		return value, nil
	}
	if obj, ok := x.(Object); ok {
		newObj, err := applyAddRecr(obj[path[depth]], path, depth+1, value)
		if err != nil {
			return nil, err
		}
		obj[path[depth]] = newObj
		return obj, nil
	}
	if arr, ok := x.(Array); ok {
		idx, err := parseIndex(arr, path[depth], len(path[depth:]) == 1)
		if err != nil {
			return nil, errors.Wrap(err, "path:"+ToPointer(path[:depth]))
		}
		if len(path[depth:]) == 1 {
			if idx == len(arr) {
				return append(arr, value), nil
			}
			return append(arr[:idx], append([]Any{value}, arr[idx:]...)...), nil
		}
		newObj, err := applyAddRecr(arr[idx], path, depth+1, value)
		if err != nil {
			return nil, err
		}
		arr[idx] = newObj
		return arr, nil
	}
	return nil, notArrayOrObjectErr(path[:depth])
}

// Remove removes node from given position.
func Remove(x Any, path []string) (Any, error) {
	x, _, err := remove(x, path)
	return x, err
}

// RemoveByPointer removes node from given position.
// The position is specified by JSON Pointer.
func RemoveByPointer(x Any, pointer string) (Any, error) {
	path, err := ToPath(pointer)
	if err != nil {
		return nil, err
	}
	return Remove(x, path)
}

func remove(x Any, path []string) (Any, Any, error) {
	return applyRemoveRecr(x, path, 0)
}

func applyRemoveRecr(x Any, path []string, depth int) (Any, Any, error) {
	if depth >= len(path) {
		return nil, x, nil
	}
	if obj, ok := x.(Object); ok {
		newObj, removed, err := applyRemoveRecr(obj[path[depth]], path, depth+1)
		if err != nil {
			return nil, nil, err
		}
		if newObj == nil {
			delete(obj, path[depth])
		} else {
			obj[path[depth]] = newObj
		}
		return obj, removed, nil
	}
	if arr, ok := x.(Array); ok {
		idx, err := parseIndex(arr, path[depth], false)
		if err != nil {
			return nil, nil, errors.Wrap(err, "path:"+ToPointer(path[:depth]))
		}
		left, removed, err := applyRemoveRecr(arr[idx], path, depth+1)
		if err != nil {
			return nil, nil, err
		}
		if left == nil {
			arr = append(arr[:idx], arr[idx+1:]...)
		} else {
			arr[idx] = left
		}
		return arr, removed, nil
	}
	return nil, nil, notArrayOrObjectErr(path[:depth])
}

// Replace replaces node by new value at given position.
func Replace(x Any, path []string, value Any) (Any, error) {
	x2, err := Remove(x, path)
	if err != nil {
		return nil, err
	}
	return Add(x2, path, value)
}

// ReplaceByPointer replaces node by new value at given position.
// The position is specified by JSON Pointer.
func ReplaceByPointer(x Any, pointer string, value Any) (Any, error) {
	path, err := ToPath(pointer)
	if err != nil {
		return nil, err
	}
	return Replace(x, path, value)
}

// Move moves a node to another position.
func Move(x Any, from, to []string) (Any, error) {
	x2, removed, err := remove(x, from)
	if err != nil {
		return nil, err
	}
	return Add(x2, to, removed)
}

// MoveByPointer moves a node to another position.
// The positions are specified by JSON Pointer.
func MoveByPointer(x Any, from, to string) (Any, error) {
	fromPath, err := ToPath(from)
	if err != nil {
		return nil, err
	}
	toPath, err := ToPath(to)
	if err != nil {
		return nil, err
	}
	return Move(x, fromPath, toPath)
}

// Copy copies a node to another position.
func Copy(x Any, from, to []string) (Any, error) {
	v, err := Get(x, from)
	if err != nil {
		return nil, err
	}
	return Add(x, to, v)
}

// CopyByPointer copies a node to another position.
// The positions are specified by JSON Pointer.
func CopyByPointer(x Any, from, to string) (Any, error) {
	fromPath, err := ToPath(from)
	if err != nil {
		return nil, err
	}
	toPath, err := ToPath(to)
	if err != nil {
		return nil, err
	}
	return Copy(x, fromPath, toPath)
}

// Test checks if a node value is the same as expect.
func Test(x Any, path []string, value Any) error {
	v, err := Get(x, path)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(v, value) {
		return errors.Errorf("test fail, value of %s is %v, expect value is %v", path, v, value)
	}
	return nil
}

// TestByPointer checks if a node value is the same as expect.
// The position is specified by JSON Pointer.
func TestByPointer(x Any, pointer string, value Any) error {
	path, err := ToPath(pointer)
	if err != nil {
		return err
	}
	return Test(x, path, value)
}

// Patch is a JSON Patch.
type Patch []PatchOperation

// PatchOperation represents an item in JSON Patch.
type PatchOperation struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value Any    `json:"value,omitempty"`
	From  string `json:"from,omitempty"`
}

// ApplyPatch applies a JSON patch to a JSON node and returns a new one.
// Note that the old node will be updated too.
func ApplyPatch(x Any, patch Patch) (Any, error) {
	var err error
	for _, p := range patch {
		switch p.Op {
		case "add":
			x, err = AddByPointer(x, p.Path, p.Value)
		case "remove":
			x, err = RemoveByPointer(x, p.Path)
		case "replace":
			x, err = ReplaceByPointer(x, p.Path, p.Value)
		case "move":
			x, err = MoveByPointer(x, p.From, p.Path)
		case "copy":
			x, err = CopyByPointer(x, p.From, p.Path)
		case "test":
			err = TestByPointer(x, p.Path, p.Value)
		}
		if err != nil {
			return nil, err
		}
	}
	return x, nil
}

func notArrayOrObjectErr(path []string) error {
	return errors.Errorf("node '%s' is not an Array or Object", ToPointer(path))
}

func parseIndex(arr Array, idxStr string, allowAppend bool) (idx int, err error) {
	if idxStr == "-" {
		idx = len(arr)
	} else {
		idx, err = strconv.Atoi(idxStr)
		if err != nil {
			return 0, errors.Errorf("bad format index '%v'", idxStr)
		}
	}
	if idx < 0 || idx > len(arr) || (idx == len(arr) && !allowAppend) {
		return 0, errors.Errorf("index '%v' out of range '%v'", idx, len(arr))
	}
	return idx, nil
}
