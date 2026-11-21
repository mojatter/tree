# Query and edit semantics

This file is the normative spec for the query language used by `Find`, `Node.Find`, `Edit`, tq and the [schema](schema.md) subpackage. The [README](../README.md#query) shows examples; when the README, the godoc or the code disagree with this file, one of them has a bug.

Each step is described by the Go call it corresponds to, so that the query and edit sides of a step stay aligned with each other and with the `Node` API. `TestFindGetParity` and `TestEditSetParity` pin these equivalences, and every intended deviation is listed in [Deviations](#deviations). `TestQueryDocExamples` and `TestEditDocExamples` run the examples in this file from rows listed in the tests, not parsed from it, so an example added here needs a row there too.

## Table of contents

- [Syntax](#syntax)
- [Results](#results)
- [Steps](#steps)
- [Receivers](#receivers)
- [Methods](#methods)
- [Selects and comparisons](#selects-and-comparisons)
- [Pipes](#pipes)
- [Errors](#errors)
- [Edits](#edits)
  - [Go equivalents](#go-equivalents)
  - [Paths](#paths)
  - [Multiple matches](#multiple-matches)
  - [Null intermediates](#null-intermediates)
  - [Edit errors](#edit-errors)
- [Deviations](#deviations)

## Syntax

A query is a sequence of steps, written without separators: `.store.book[0].title`. Blanks may separate steps (`.a .b`, `.a [0]`), but a key follows its `.` or `..` directly: `. a` fails with `syntax error: unexpected blank at offset 1: ". a"`.

| Step | Meaning |
| - | - |
| `.key`, `."key"`, `["key"]` | Key. An unquoted key is a run of Unicode letters, decimal digits and marks and `_`, such as `.名前`; quote anything else (`."a b"`, `."$a"`, `."1e+3"`). A sign ends an unquoted key, so `.a.1e+3` is a syntax error. |
| `.0` | Key `"0"`, which an array reads as the index 0. |
| `[i]`, `[-i]` | Index, written in decimal digits; `[x]`, `[true]` and `[1.5]` fail with `syntax error: invalid array index: "[x]"`. |
| `[from:to]` | Slice; either bound may be omitted or negative. |
| `[]` | Every element of an array or value of an object. |
| `[selector]` | Every element or value the selector matches, see [Selects and comparisons](#selects-and-comparisons). |
| `..key`, `.."key"` | Every value under `key` anywhere in the tree. |
| `.name(args)`, `name(args)` | Method, see [Methods](#methods). The `.` is optional, so `.a count()` counts `a`. |
| `\|` | Pipe, see [Pipes](#pipes). |
| `(...)` | Group. |
| `.`, `..` | The node itself, as is `..` before anything but a key, so `..[0]` reads like `[0]`. |

A literal is a whole query, alone or in parentheses, which yields it, or one side of a comparison. Anywhere else, next to a step or as a selector without an operator, it is a syntax error: `.a x`, `x.a`, `.a | x` and `[.a == 1 or 2]` fail, the first with `syntax error: unexpected literal "x": ".a x"`. The literals are:

- `null`, `true` and `false`.
- Numbers, any word `strconv.ParseFloat` reads as a finite value, such as `1`, `1.5`, `1e3`, `01` and `0x1p4`, read as `float64`. A negative number such as `-1` is a literal only as one side of a comparison (`[.a == -1]`), not as a query, so `-1`, `(-1)`, `[(-1) == .a]` and `[.a == (-1)]` are syntax errors, the last with `syntax error: invalid token -: "[.a == (-1)]"`. After a `.` a number is a key, so `.5` reads the key `"5"`.
- Strings in double or single quotes; a backslash escapes the next byte (`"a\"b"`).
- Any other bare word is a string, so `[.a == x]` compares with `"x"`. Quote strings to keep them apart from `null`, `true`, `false` and numbers: `"null"` is a string. `and` and `or` are keywords unless they directly follow a `.` or `..`, so `.and` is a key and `[.a == and]` is a syntax error.

Method arguments are separated by commas and end at the `)` that closes the call:

- Quoted strings and paired parentheses and brackets inside an argument are skipped, so `sort(.a.count())` and `has("a)b")` each take one argument. A parenthesis or bracket outside quotes must have its pair inside the same argument, so a key with a lone one is quoted: `has("a]b")`.
- Blanks (spaces, tabs, line feeds and carriage returns) around an argument are dropped, and nothing or only blanks between the parentheses is no argument; any other character, such as a form feed or a full-width space, is part of the argument.
- An argument written as a string literal, in double or single quotes with the same backslash escape as elsewhere, passes its contents: `has("a,b")`, `has('x')` and `has("a\"b")` check the keys `a,b`, `x` and `a"b`.
- Any other argument passes its text as written, with any quoted strings, backslashes and inner blanks kept: `has(isbn)`, `has(0)`, `sort(.price)`, `has(a b)` checks the key `a b`, and `has(a"b")` checks the key `a"b"`. A quote anywhere in an argument opens a string that must be closed, so a key with a lone quote is quoted: `has("it's")`.
- `sort()` and `rsort()` read their argument as a query, so `sort(.price)` and `sort(".price")` are the same.
- An argument that starts with a quote must be that one string literal, and an empty argument (`has(,)`), a parenthesis or bracket without its pair (`has(a]b)`, `sort(.a[0)`), a quote without its pair (`has(it's)`) or a missing `)` is a syntax error.

An empty expression, such as the empty string, `()`, `[.a == ]`, `sort("")` or the left side of `|` in `| .a` and `.a || .b`, is a syntax error.

Outside quotes and method arguments, a blank between tokens is skipped, except between a `.` or `..` and its key. Any other character there that starts no token, such as `$`, `!` or a full-width space, is a syntax error, and so is an unterminated string: `.a!` fails with `syntax error: unexpected character "!" at offset 2: ".a!"`, where the offset counts bytes of the expression shown in quotes, before it is escaped. A method name is a run of `a` to `z` followed directly by its `(`: otherwise the word is a key, or a bare word without a `.` before it, and the parentheses are a group, so `.a.count1(.x)` reads `.a.count1.x`, `.a.count (.x)` reads `.a.count.x`, `.a.count ()` fails for its empty group with `syntax error: empty expression: ".a.count ()"` and `count (.a)` fails as a literal next to a step.

## Results

A step maps one node to zero or more nodes, and a query runs each step on every result of the step before it. `Find` returns the results of the last step, reading a Go `nil` node, a nil pointer node such as `(*tree.Array)(nil)` and a zero `Any` as `null`.

Single-valued steps (a key, an index, `first()` and `last()`) yield exactly one result, which is `null` when there is nothing to read. Multi-valued steps (`[]`, a select, a slice and `..key`) yield nothing when there is nothing to read. `keys()` and `values()` are neither: they yield one array on an array or an object and nothing otherwise. Every other method yields exactly one result. So `.store.zz` yields `null`, `.store.book[].isbn` yields one `null` per book without an ISBN, and `.store.zz[]` yields nothing.

## Steps

| Step | Go equivalent |
| - | - |
| `.key`, `["key"]` | `n.Get("key")` |
| `[i]` | `n.Get(i)`, except that a negative `i` on an array counts from the end |
| `first()`, `last()` | `a.Get(0)`, `a.Get(len(a)-1)` for an array `a`; `Nil` otherwise |
| `[from:to]` | `a[from:to]` for an array `a`, with each bound clamped to `[0, len(a)]` and a negative bound counted from the end; nothing otherwise |
| `[]` | the elements of an array, or `m.Values()` (in sorted key order) of an object; nothing otherwise |
| `..key` | `Walk(n, ...)` collecting `c.Get("key")` from every node `c` for which `c.Has("key")`, the root included, parents before children and object keys in sorted order |
| `.`, `..` | `n` |

`n.Get` already reads a missing key, an out-of-range index, a scalar receiver and a Go `nil` value as `Nil`, so the key and index steps never fail. `Map.Get(i)` reads the key `strconv.Itoa(i)`, so `.m[0]` reads `"0"` and `.m[-1]` reads `"-1"`; `Array.Get("1")` reads the index 1, so `.a.1` reads the second element and `.a["x"]` reads `null`.

An array has the key `"i"` for every index `i` in range, so `..0` also yields the first element of every non-empty array, and `..key` with a non-numeric `key` never matches an array. A key is read as an index as `strconv.Atoi` reads it, so `"01"` and `"+1"` are the index 1 too.

## Receivers

What each step yields, by the node it runs on. A missing key reads as `null`, so it behaves as the `null` column.

| Step | null | scalar | array | object |
| - | - | - | - | - |
| `.key` | `null` | `null` | the element at `key` when it is a non-negative integer, else `null` | the value, else `null` |
| `[i]` | `null` | `null` | the element, counting from the end when `i < 0`, else `null` | the value at key `"i"`, else `null` |
| `first()`, `last()` | `null` | `null` | the first or last element, `null` when empty | `null` |
| `[from:to]` | nothing | nothing | the elements in range | nothing |
| `[]` | nothing | nothing | each element | each value, in sorted key order |
| `[selector]` | nothing | nothing | each matching element | each matching value, in sorted key order |
| `..key` | nothing | nothing | see [Steps](#steps) | see [Steps](#steps) |

A `null` element of an array or value of an object is yielded as `null`.

## Methods

| Method | null | scalar | array | object |
| - | - | - | - | - |
| `count()` | `0` | `0` | the length | the number of keys |
| `keys()` | nothing | nothing | `[0, 1, ...]` | the keys, sorted |
| `values()` | nothing | nothing | the array itself | a new array of the values, in sorted key order |
| `empty()` | `true` | `true` for `""`, else `false` | `true` when empty | `true` when empty |
| `type()` | `"null"` | `"string"`, `"number"` or `"boolean"` | `"array"` | `"object"` |
| `has(k)` | `false` | `false` | `true` when `k` is a non-negative index in range | `true` when the key exists, even with a `null` value |
| `contains(v)` | `false` | `true` when a string contains the substring `v`; `false` for numbers and booleans | `true` when a string, number or boolean element formats as `v` | `true` when a string, number or boolean value formats as `v` |
| `first()`, `last()` | `null` | `null` | see [Receivers](#receivers) | `null` |
| `sort([expr])` | the node as is | the node as is | a sorted copy | the node as is |
| `rsort([expr])` | the node as is | the node as is | a copy sorted in descending order | the node as is |

Each of these methods fails to parse when given a number of arguments it does not take. `has()` and `contains()` take exactly one, `sort()` and `rsort()` take at most one, the expression, and the other methods take none: `has()`, `sort(.k, .x)` and `count("x")` fail with `has() requires exactly 1 argument, got 0`, `sort() requires at most 1 argument, got 2` and `count() requires no arguments, got 1`. Nothing or only blanks between the parentheses is no argument, so `count( )` counts, while `""` is one argument: `count("")` fails and `has("")` checks the empty key.

`values()`, `sort()` and `rsort()` yield an array whose own elements hold no Go `nil` node, each read as `null`: `values()` copies an array only when it holds a Go `nil` element, and `sort()` and `rsort()` sort a copy anyway. Nested arrays and objects are yielded as they are.

`keys()` and `values()` yield one array, not one result per key, so `.a.keys()[]` yields each key. Array keys are numbers: `.a.keys()[. == 0]` matches and `.a.keys()[. == "0"]` does not. `first()` and `last()` are the single-valued steps of [Steps](#steps), listed here because they are written as methods.

`contains()` never matches a `null`, array or object element, so `contains("")` matches only an empty string. A string element must equal `v`: on `{"s": "hello", "a": ["hello"]}`, `.s.contains("ell")` is `true` and `.a.contains("ell")` is `false`.

`sort()` orders `null < false < true < numbers < strings < arrays < objects`; numbers compare by value with NaN first, strings bytewise, and arrays and objects keep their original order among themselves. With an expression, each element is sorted by the first result of `expr` on it, or `null` when it yields nothing. The sort is stable, and so is `rsort()`: elements with equal keys, arrays and objects included, keep their original order, so `rsort()` is not `sort()` reversed: `rsort(".k")` on `[{"k": 1, "i": "a"}, {"k": 1, "i": "b"}]` returns it unchanged.

## Selects and comparisons

A select `[selector]` runs the selector on each element of an array or value of an object. A selector is:

- `left op right`, a comparison, where each side is a query or a literal and `op` is `==`, `!=`, `<`, `<=`, `>`, `>=` or `~=`. Either side can be in parentheses, except a negative number: `[(.a) == 1]`, `[(1) == .a]` and `[1 == (.a)]` compare `.a` with 1, while `[(-1) == .a]` and `[.a == (-1)]` are syntax errors.
- A query alone, other than a literal, which matches when any of its results is truthy.
- Selectors joined by `and` and `or`, where `and` binds tighter, and grouped with parentheses. A `(` at the start of a selector opens a group when a comparison operator, `and` or `or` comes before its `)`, outside any `[...]`; otherwise it starts the left side of a comparison or a query, so `[(.a == 1) and .b]` is a group and `[(.a).b == 1]` compares `.a.b`.

Each side of a comparison yields at most one value; a side that yields more fails with `<side> returns more than one value [...]`. To match when any element matches, use a nested select: `[.tags[.x == 1]]`, not `[.tags[].x == 1]`. A side that yields nothing, such as `.tags[]` on an empty array, never matches, with any operator.

| Left and right | Go equivalent | Result |
| - | - | - |
| Both scalars or `null` | `l.Value().Compare(op, r.Value())` | See below. |
| Either an array or an object | `Equal(l, r)` | `==` and `!=` compare by value; an ordered operator or `~=` never matches. |

Scalar comparison:

- Values of different types are never equal, so only `!=` matches: `1 == "1"` is false.
- Numbers compare as `float64` (`1 == 1.0`), and strings bytewise (`"B" < "a"`).
- Booleans and `null` support only `==` and `!=`; an ordered operator never matches.
- `null` equals only `null`, so `[.isbn == null]` matches a null or missing ISBN, and `[.isbn != "x"]` keeps books without one.
- `l ~= r` matches when the string `l` matches the [RE2](https://pkg.go.dev/regexp/syntax) pattern `r`; it never matches a non-string, and an invalid pattern never matches.

A query used alone is truthy when a result is `true`, a number greater than 0, a non-empty string, or a non-empty array or object. `null`, `false`, `0`, negative numbers and empty strings, arrays and objects are falsy. `empty()` is not the negation of this: `.n.empty()` is `false` for `0`, so on `{"n": [0, 1, "", null]}` `.n[.]` yields only `1` while `.n[.empty() == false]` yields `0` and `1`.

## Pipes

`left | right` collects every result of `left` into one array and runs `right` on that array, so `..author | [0]` yields the first author, `.store.book[] | count()` counts the books and `.zz | count()` yields `1`, as `.zz` yields one `null` and `|` collects `[null]`; `.zz[] | count()` yields `0`. Unlike jq's pipe, it does not run `right` once per result. A `|` with nothing before it is an empty expression, while a trailing `|`, as in `.a |`, yields the collected array as `.a | .` does. `.a[] | .` yields what `tq -s '.a[]'` prints; `tq -x` does the opposite and prints each element of an array result.

## Errors

A query fails only for:

- A syntax error, reported as `syntax error: ...`, including an empty expression.
- An unknown method (`unknown method: foo`) or a method given a number of arguments it does not take (`has() requires exactly 1 argument, got 0`), reported when the query is parsed.
- A comparison side that yields more than one value.
- An error from the expression of `sort()` or `rsort()`.

A step never fails because of the data it reads.

## Edits

`Edit(&n, expr)` and tq `-e` take `<query> = <value>` (set), `<query> += <value>` (append) or `<query> ^?` (delete), where only blanks may follow `^?`. The expression is read as a query up to the first `=`, `+=` or `^?` outside quotes, parentheses, brackets and method arguments, and the rest is `<value>`, parsed as a single YAML document, so `.url = "http://x?a=1"` sets the whole URL and `.a = [1, 2]` sets an array. Comparison operators (`==`, `!=`, `<=`, `>=`, `~=`) are never read as an assignment, so `.a[.b == 1].c = 2` sets `c` on the matching elements and `.a ==1` is a syntax error; write `.a = "=1"` to set that string. An edit operator inside parentheses or brackets is a syntax error: `.a[.b = 1].c = 2` fails with `syntax error: unexpected character "=" at offset 6: ".a[.b = 1].c = 2"`. Inside a method argument it is part of the argument and not an edit operator, so `[has(a=b)].v = 1` sets `v` where the key `a=b` exists, and `.a.sort(.k = 1) = 1` fails on the `sort()` expression alone: `syntax error: unexpected character "=" at offset 3: ".k = 1"`.

### Go equivalents

Where a query step reads with `Get`, the edit of that step writes with `Set` and `Delete` on the same receiver.

| Edit | Go equivalent |
| - | - |
| `.key = v`, `[i] = v` | `n.Set(key, v)`, `n.Set(i, v)` |
| `.key += v`, `[i] += v` | `n.Set(key, append(n.Get(key).Array(), v))` when the value is an array or `null`, else an error |
| `.key ^?`, `[i] ^?` | `n.Delete(key)`, `n.Delete(i)` |
| `first()`, `last()` | `[0]` and `[-1]` on an array; an error on an object |
| `. = v` | `n = v` |
| `. += v` | `n = append(a, v)` for an array `a`, `Array{v}` for `null`, else an error |
| `. ^?` | an error |

So, like `Map.Set` and `(*Array).Set`:

- On an object, `[i]` writes the key `"i"`, including a negative `i`: `.m[-1] = 1` sets `"-1"`.
- On an array, `.key` writes the index `key` when it is a non-negative integer, and fails otherwise.
- Writing past the end of an array grows it, filling the gap with `null`; an index of `1<<24` or more that `=` or `+=` writes to or through fails.
- Deleting a missing key or an index past the end changes nothing, while a negative index before the start fails, as it does for `=` and `+=`.
- `^?` on a plain `Array` shifts its elements in place, as `(*Array).Delete` does, so another slice sharing its backing array sees the shift; copy the array first, or use a `*Array`.
- `=` on a key with a `null` value stores `null` and keeps the key, while `^?` removes it: `.a = null` and `.a ^?` differ.
- A step on a scalar fails, unlike the same step in a query: a key, an index, `first()`, `last()`, `[]`, a select and a slice.
- A step through an `Any` or `*Any` edits the node it wraps, so a wrapped `*Array` or other `EditorNode` changes in place and the wrapper stays; a wrapped plain `Array` may be replaced by the edited `Array`, since an `Array` value cannot grow or shrink in place. `+=` stores the appended `Array` as the value (see the table above), so an `Any` or `*Array` that is itself appended to is replaced by a plain `Array`, while `.a[0] += v` inside a `*Array` edits it in place.

### Paths

An edit writes through keys, indexes, `first()` and `last()`, and through `[]`, selects, slices and `..key`, which edit every match. Any other step, such as another method or a parenthesised group containing a pipe inside a longer path, is a syntax error before any data is read (`.a.sort()[0] = 1` fails with `syntax error: unsupported edit query: sort()`), and so is a query made of a literal alone (`"x" = 1`). A trailing `|` (as in jq's `.a |= 1`) is also a syntax error as the last step. Parentheses around steps without a pipe are ignored, so `(.a[0]).b = 1` edits like `.a[0].b = 1`, and so are parentheses around the whole query. An edit after a top-level `|` needs at least two steps after it, or a `..key`, since any other single step would write into the array `|` collects rather than the document: `.a | .[0] = 9` and `(.a | .[]) = 9` fail with `syntax error: unsupported edit query: | .[0]` and `| .[]`, while `.a[] | .[].x = 9` sets `x` in every object of `a` and `.a | ..x = 9` sets every `x` under `a`.

Bare `.` and `..` steps are ignored, so `.[0]` edits like `[0]`, `.a.` like `.a` and `..b. ^?` deletes like `..b ^?`. `[]`, a select or a slice as the last step edits the matched elements themselves: `.a[] = 1` sets every element of `a`, `null` elements included, `.a[] += 1` turns a `null` element into `[1]` as `.a[0] += 1` would, and `.a[] ^?` deletes every element, `null` elements included, as `.a[0] ^?` would, so `.a[. == 1] ^?` deletes the elements equal to `1` and leaves the others in place, as jq's `del(.a[] | select(. == 1))` does, and on an object it deletes the matching keys: `.m[. == 1] ^?` deletes every key of `m` whose value is `1`. A trailing `.` after them, as in `.a[]. = 1`, edits the same way.

`[]`, a select and a slice edit nothing on `null`. A slice edits each element in range as `[i]` would.

### Multiple matches

An edit through `..key`, `[]`, a select or a slice gives each match its own copy of the RHS, so editing one match later leaves the others as they are. A `..key` edit also edits inner matches before outer ones, so it never edits inside the RHS, i.e. a value it has just assigned: `..b = {"b": 1}` on `{"b": {"b": 2}}` gives `{"b": {"b": 1}}`. If an inner match does not fit the rest of the path, or `+=` finds a value that is not an array or `null`, the edit stops with an error, as in jq: `..b.c = 1` on `{"b": {"c": {"b": []}}}` fails with `cannot index array with "c"`, and `..a += 1` on `{"a": 1, "b": {"a": []}}` sets `b.a` to `[1]` and then fails with `cannot append to number`. `..key` with a numeric `key` edits arrays too, inner arrays first: `..0 = 9` on `{"a": [[1], [2]]}` gives `{"a": [9, [9]]}`. Edits are not transactional: when a match fails, the matches edited before it stay edited, except that `^?` whose last step is `[]`, a select or a slice deletes from a container only after its selector has accepted or rejected every element, so a selector that fails there deletes nothing from that container (see [Edit errors](#edit-errors)).

### Null intermediates

A `null` reached by a step, including a Go `nil` value such as a hole left by growing an array, a nil pointer node such as `(*tree.Array)(nil)` and a nil `Map` or `Array`, is treated as missing:

- `=` and `+=` replace it with what the rest of the path needs when every later step is a key, a non-negative index or `first()`: `.a[0].x = 1` on `{"a": [null]}` gives `{"a": [{"x": 1}]}`, and `.a += 1` on `{"a": null}` gives `{"a": [1]}`.
- Otherwise it stays as it is: `.a.b[].c = 1` and `.a.last() = 1` on `{"a": null}` change nothing: `a` would become an empty array, which has no last element to count from the end. `^?` never replaces it, and deletes it only when it is what `^?` names: `.a.b ^?` on `{"a": null}` changes nothing, while `.a ^?` gives `{}` and `.a[] ^?` on `{"a": [null]}` gives `{"a": []}`.

A `null` root is replaced the same way: `.a.b = 1` on `null` gives `{"a": {"b": 1}}`, `. += 1` on `null` gives `[1]`, and `.[-1] = 1` on `null` changes nothing.

An array whose `Array()` is nil, such as a pointer to a nil `Array`, edits as an empty array: `[]`, a select and a slice change nothing, and `+=` gives `[v]`. A pointer to a nil `Map` edits as an empty object through the pointer: `.a.x = 1` on `{"a": new(tree.Map)}` sets `x` in the `Map` it points to.

### Edit errors

An edit fails for these, before any data is read:

- A syntax error, an unknown method or a wrong number of method arguments in the query (see [Errors](#errors)).
- An expression without a query, without an operator, without a value after `=` or `+=`, or with anything but blanks after `^?` (`syntax error: invalid edit expression "..."`). A value of white space alone is no value, and white space here is wider than blanks: a form feed or a full-width space alone is no value either.
- An unsupported step (see [Paths](#paths)).
- A value that is not valid YAML (the YAML parser's own message, `yaml: ...`) or that holds more than one YAML document (`syntax error: multiple YAML documents in edit value ...`).

It also fails for these conditions, reported when the edit reaches them:

| Condition | Example | Message |
| - | - | - |
| A key, an index, `first()`, `last()` or a slice on a scalar | `.a.b = 1` or `.a.b ^?`, `.a[0] = 1`, `.a.first() = 1`, `.a[1:] = 1` on `{"a": 1}` | `cannot index number with "b"`, `cannot index number with 0`, `cannot index number with first()`, `cannot index number with [1:]` |
| A key that is not a non-negative integer on an array | `.a.b = 1` on `{"a": []}` | `cannot index array with "b"` |
| A negative index before the start of an array, `last()` on an empty array included | `.a[-5] = 1` or `.a.last() = 1` on `{"a": []}` | `cannot index array with -5`, `cannot index array with -1` |
| `first()`, `last()` or a slice on an object | `.a.first() = 1`, `.a[1:] = 1` on `{"a": {}}` | `cannot index object with first()`, `cannot index object with [1:]` |
| `[]` or a select on a scalar | `.a[] = 1` on `{"a": 1}` | `cannot iterate over number` |
| `+=` on a value that is not an array or `null`, however the edit reaches it | `.a += 1` on `{"a": 1}`, `. += 1` on `1`, `.a[] += 1` on `{"a": [1]}` | `cannot append to number` |
| An array index of `1<<24` or more that `=` or `+=` writes to or through | `.a[16777216] = 1` or `.a[16777216].b = 1` on `{}` | `array index 16777216 exceeds maximum 16777215` |
| `^?` on the root itself | `. ^?` | `cannot delete .` |
| A comparison side that yields more than one value | `.a[.b[] == 1].c = 1` on `{"a": [{"b": [1, 2]}]}` | `.b[] returns more than one value [1 2]` |

A syntax error found while reading the query quotes the whole edit expression, value included, and an offset in it counts bytes from the start of that expression: `.foo! = 9` fails with `syntax error: unexpected character "!" at offset 4: ".foo! = 9"`. As in a query, an error inside the expression of `sort()` or `rsort()` quotes that expression alone, and an unknown method or a wrong number of arguments quotes nothing. An unsupported step is found once the query is read, and its message shows that step, or a `|` with what follows it, rather than the edit expression (see [Paths](#paths)). Any of these errors in the query is reported before an error in the value: `.a] = [` fails with `syntax error: no left bracket: ".a] = ["`, and `.a.sort()[0] = [` with `syntax error: unsupported edit query: sort()`. A comparison side that yields more than one value depends on the data and comes after the value: `.a[.b[] == 1].c = [` fails with the YAML error. The expression itself is checked once the operator has been looked for, before the query is parsed: `.a] =` and `.a.sort()[0] =` fail with `syntax error: invalid edit expression "..."`, not with the query's error. Only an error met while looking for the operator comes before that, namely an unexpected character, an unexpected blank, an unterminated string or a method argument list that breaks the rules of [Syntax](#syntax), such as `has(,)` or a missing `)`: `.foo! =` fails with `syntax error: unexpected character "!" at offset 4: ".foo! ="` and `. a =` with `syntax error: unexpected blank at offset 1: ". a ="`, while `.a.has() =`, `.a.sort(.k!) =` and `.a x =` fail with `syntax error: invalid edit expression "..."`.

Each `cannot index`, `cannot iterate over` and `cannot append to` message names the receiver's type as `type()` does, whatever the operator and whether an array is an `Array`, a `*Array` or inside an `Any`: `.a.b += 1` and `.a.b ^?` on `{"a": []}` fail like `.a.b = 1`. A key, a method or a slice is shown as the query prints it, a key always quoted and an index as a number (`.a[ 1 : ] = 1` says `[1:]`), except that `last()` on an empty array fails as the index `-1` it resolves to. A custom `EditorNode` reports its own `Set` and `Delete` errors.

An edit whose steps are all keys, indexes, `first()` and `last()` leaves the document as it was when it fails, a `null` root included: `.a[16777216] = 1` on `{}` leaves `{}` and on `null` leaves `null`. An edit with several matches keeps the matches edited before the failure (see [Multiple matches](#multiple-matches)); `^?` whose last step is `[]`, a select or a slice runs the selector on every element or value of a container before it deletes any of them, so a selector that fails deletes nothing from that container, while the containers it deleted from before, through `..key` or an earlier `[]`, stay edited: `..a[.t[] == 1] ^?` on `{"x": {"a": [{"t": [1]}]}, "y": {"a": [{"t": [1, 2]}]}}` empties `x.a` and then fails on `y.a`. A caller that needs the document untouched after any failure edits a deep copy, `c := tree.CloneDeep(n)` (`Clone` copies only the top level); tq `-U` writes a file only when every edit on it succeeds.

## Deviations

From the Go API, on purpose:

- A negative index counts from the end of an array in queries and edits, while `Array.Get(-1)` gives `Nil` and `(*Array).Set(-1, v)` fails.
- An edit replaces a `null` intermediate or root with what the rest of the path needs (see [Null intermediates](#null-intermediates)); `Nil` has no `Set`.
- `first()` and `last()` give `null` on an object, while `Map.Get(0)` reads the key `"0"`: a key is not a position, and `[0]` already reads that key as `Get` does.
- `[]`, selects, slices, `..key`, pipes and the methods other than `first()` and `last()` have no `Node` method of their own; [Steps](#steps) gives the Go each one amounts to.

From jq:

- A key or an index on a scalar and a key on an array read `null` instead of failing, and an index on an object reads the key with that number.
- `|` collects results into one array instead of streaming them, see [Pipes](#pipes).
- `..` alone, or before anything but a key, is the node itself and does not recurse.
- A bare word is a string literal, so `[.a == x]` compares with `"x"`.
- An edit through `|` writes through the array `|` collects, so it needs at least two steps after `|`, or a `..key`, see [Paths](#paths); there is no `|=`.
- Each side of a comparison yields at most one value, and an ordered operator with `null`, a boolean or a container never matches.
- `0`, negative numbers and empty strings, arrays and objects are falsy in a select.
- `count()` on a string is `0`.
- `[]` on an object yields its values in sorted key order.
- `^?` with a negative index before the start of an array fails, while jq's `del(.[-5])` ignores it.
- A slice edits each element in range: `.a[1:] = 9` on `{"a": [0, 1, 2]}` gives `{"a": [0, 9, 9]}` and `+=` appends to each element in range, which must be an array or `null`, while jq's `.a[1:] = v` replaces the slice with the array `v`.
- `[]` or a select on `null` edits nothing: `.a[] = 1` on `{"a": null}` leaves it, while jq fails with `Cannot iterate over null`.
- `+=` appends `v` as one element, so `.a += [1]` on `{"a": [0]}` gives `{"a": [0, [1]]}`; jq's `+=` adds, and would give `[0, 1]`.
