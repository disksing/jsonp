package jsonp_test

import (
	"encoding/json"
	"testing"

	"github.com/disksing/jsonp"
	"github.com/stretchr/testify/require"
)

func TestPointer(t *testing.T) {
	r := require.New(t)
	js := `
{
	"foo": ["bar", "baz"],
	"": 0,
	"a/b": 1,
	"c%d": 2,
	"e^f": 3,
	"g|h": 4,
	"i\\j": 5,
	"k\"l": 6,
	" ": 7,
	"m~n": 8
 }
`
	var x jsonp.Any
	err := json.Unmarshal([]byte(js), &x)
	r.Nil(err)

	jsonp.WalkByPointer(x, func(pointer string, c jsonp.Any) {
		c2, err := jsonp.GetByPointer(x, pointer)
		r.Nil(err, "pointer:%v", pointer)
		r.Equal(c2, c, "pointer:%v", pointer)
	})
}

func TestPatch(t *testing.T) {
	r := require.New(t)
	cases := [][3]string{
		{`{ "foo": "bar"}`, `[{ "op": "add", "path": "/baz", "value": "qux" }]`, `{"baz": "qux","foo": "bar"}`},
		{`{ "foo": [ "bar", "baz" ] }`, `[{ "op": "add", "path": "/foo/1", "value": "qux" }]`, `{ "foo": [ "bar", "qux", "baz" ] }`},
		{`{"baz": "qux","foo": "bar"}`, `[{ "op": "remove", "path": "/baz" }]`, `{ "foo": "bar" }`},
		{`{ "foo": [ "bar", "qux", "baz" ] }`, `[{ "op": "remove", "path": "/foo/1" }]`, `{ "foo": [ "bar", "baz" ] }`},
		{`{"baz": "qux","foo": "bar"}`, `[{ "op": "replace", "path": "/baz", "value": "boo" }]`, `{"baz": "boo","foo": "bar"}`},
		{`{"baz": "qux","foo": "bar"}`, `[{ "op": "replace", "path": "/baz", "value": "boo" }]`, `{"baz": "boo","foo": "bar"}`},
		{`{"foo": {"bar": "baz","waldo": "fred"},"qux": {"corge": "grault"}}`, `[{ "op": "move", "from": "/foo/waldo", "path": "/qux/thud" }]`, `{"foo": {"bar": "baz"},"qux": {"corge": "grault","thud": "fred"}}`},
		{`{ "foo": [ "all", "grass", "cows", "eat" ] }`, `[{ "op": "move", "from": "/foo/1", "path": "/foo/3" }]`, `{ "foo": [ "all", "cows", "eat", "grass" ] }`},
		{`{"baz": "qux","foo": [ "a", 2, "c" ]}`, `[{ "op": "test", "path": "/baz", "value": "qux" },{ "op": "test", "path": "/foo/1", "value": 2 }]`, `#NOERR`},
		{`{ "baz": "qux" }`, `[{ "op": "test", "path": "/baz", "value": "bar" }]`, `#ERR`},
		{`{ "foo": "bar" }`, `[{ "op": "add", "path": "/child", "value": { "grandchild": { } } }]`, `{"foo": "bar","child": {"grandchild": {}}}`},
		{`{ "foo": "bar" }`, `[{ "op": "add", "path": "/baz", "value": "qux", "xyz": 123 }]`, `{"foo": "bar","baz": "qux"}`},
		{`{ "foo": "bar" }`, `[{ "op": "add", "path": "/baz/bat", "value": "qux" }]`, "#ERR"},
		{`{"/": 9,"~1": 10}`, `[{"op": "test", "path": "/~01", "value": 10}]`, `{"/": 9,"~1": 10}`},
		{`{"/": 9,"~1": 10}`, `[{"op": "test", "path": "/~01", "value": "10"}]`, `#ERR`},
		{`{ "foo": ["bar"] }`, `[{ "op": "add", "path": "/foo/-", "value": ["abc", "def"] }]`, `{ "foo": ["bar", ["abc", "def"]] }`},

		{`{"foo": ["bar", "baz"]}`, `[{"op":"remove", "path": "/foo/0"}]`, `{"foo": ["baz"]}`},
		{`{"foo": ["bar", "baz"]}`, `[{"op":"remove", "path": "/foo/1"}]`, `{"foo": ["bar"]}`},
	}
	for _, c := range cases {
		origin := mustUnmarshal(r, c[0])
		patch := mustUnmarshalPatch(r, c[1])
		result, err := jsonp.ApplyPatch(origin, patch)
		if err != nil {
			r.Equal(c[2], "#ERR", "CASE:%v, ERR:%v", c, err)
			continue
		}
		if c[2] != "#NOERR" {
			expect := mustUnmarshal(r, c[2])
			r.Equal(result, expect, "CASE:%v", c)
		}
	}
}

func mustUnmarshalPatch(r *require.Assertions, s string) jsonp.Patch {
	var patch jsonp.Patch
	err := json.Unmarshal([]byte(s), &patch)
	r.Nil(err)
	return patch
}

func mustUnmarshal(r *require.Assertions, s string) jsonp.Any {
	var x jsonp.Any
	err := json.Unmarshal([]byte(s), &x)
	r.Nil(err, "JSON:%s", s)
	return x
}
