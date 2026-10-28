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

A query is a sequence of steps, written without separators: `.store.book[0].title`.

| Step | Meaning |
| - | - |
| `.key`, `."key"`, `["key"]` | Key. An unquoted key is a run of `A-Z`, `a-z`, `0-9` and `_`; quote anything else (`."a b"`). |
| `.0` | Key `"0"`, which an array reads as the index 0. |
| `[i]`, `[-i]` | Index. |
| `[from:to]` | Slice; either bound may be omitted or negative. |
| `[]` | Every element of an array or value of an object. |
| `[selector]` | Every element or value the selector matches, see [Selects and comparisons](#selects-and-comparisons). |
| `..key` | Every value under `key` anywhere in the tree. |
| `.name(args)` | Method, see [Methods](#methods). |
| `\|` | Pipe, see [Pipes](#pipes). |
| `(...)` | Group. |
| `.`, `..` | The node itself. |

Literals appear in selectors, and a query made of a literal alone yields it:

- `null`, `true` and `false`.
- Numbers such as `1`, `1.5` and `1e3`, read as `float64`. A negative number such as `-1` is a literal only in a selector (`[.a == -1]`) and a syntax error elsewhere.
- Strings in double or single quotes; a backslash escapes the next byte (`"a\"b"`).
- Any other bare word is a string, so `[.a == x]` compares with `"x"`. Quote strings to keep them apart from `null`, `true`, `false` and numbers: `"null"` is a string.

Method arguments are read as one CSV record, so `sort(".a")` passes `.a`; a `"` inside an argument is written `""` and a backslash is not an escape: `has("a""b")` checks the key `a"b`, and `has("a\"b")` fails with `failed to parse args: ...`.

An empty expression, such as ``, `()`, `[.a == ]` or `sort("")`, is a syntax error.

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
| `values()` | nothing | nothing | the array itself, not a copy | a new array of the values, in sorted key order |
| `empty()` | `true` | `true` for `""`, else `false` | `true` when empty | `true` when empty |
| `type()` | `"null"` | `"string"`, `"number"` or `"boolean"` | `"array"` | `"object"` |
| `has(k)` | `false` | `false` | `true` when `k` is a non-negative index in range | `true` when the key exists, even with a `null` value |
| `contains(v)` | `false` | `true` when a string contains the substring `v`; `false` for numbers and booleans | `true` when a string, number or boolean element formats as `v` | `true` when a string, number or boolean value formats as `v` |
| `first()`, `last()` | `null` | `null` | see [Receivers](#receivers) | `null` |
| `sort([expr])` | the node as is | the node as is | a sorted copy | the node as is |
| `rsort([expr])` | the node as is | the node as is | a copy sorted in descending order | the node as is |

`keys()` and `values()` yield one array, not one result per key, so `.a.keys()[]` yields each key. Array keys are numbers: `.a.keys()[. == 0]` matches and `.a.keys()[. == "0"]` does not. `first()` and `last()` are the single-valued steps of [Steps](#steps), listed here because they are written as methods.

`contains()` never matches a `null`, array or object element, so `contains("")` matches only an empty string. A string element must equal `v`: on `{"s": "hello", "a": ["hello"]}`, `.s.contains("ell")` is `true` and `.a.contains("ell")` is `false`. `has()` and `contains()` take exactly one argument and fail to parse otherwise. `sort()` and `rsort()` take an optional expression and ignore any further argument, so `sort(".k",".x")` sorts by `.k`. The other methods ignore their arguments, so `count("x")` counts.

`sort()` orders `null < false < true < numbers < strings < arrays < objects`; numbers compare by value with NaN first, strings bytewise, and arrays and objects keep their original order among themselves. With an expression, each element is sorted by the first result of `expr` on it, or `null` when it yields nothing. The sort is stable, and so is `rsort()`: elements with equal keys, arrays and objects included, keep their original order, so `rsort()` is not `sort()` reversed: `rsort(".k")` on `[{"k": 1, "i": "a"}, {"k": 1, "i": "b"}]` returns it unchanged.

## Selects and comparisons

A select `[selector]` runs the selector on each element of an array or value of an object. A selector is:

- `left op right`, a comparison, where each side is a query or a literal and `op` is `==`, `!=`, `<`, `<=`, `>`, `>=` or `~=`.
- A query alone, which matches when any of its results is truthy.
- Selectors joined by `and` and `or`, where `and` binds tighter, and grouped with parentheses.

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

`left | right` collects every result of `left` into one array and runs `right` on that array, so `..author | [0]` yields the first author, `.store.book[] | count()` counts the books and `.zz | count()` yields `1`, as `.zz` yields one `null` and `|` collects `[null]`; `.zz[] | count()` yields `0`. Unlike jq's pipe, it does not run `right` once per result. `.a[] | .` yields what `tq -s '.a[]'` prints; `tq -x` does the opposite and prints each element of an array result.

## Errors

A query fails only for:

- A syntax error, reported as `syntax error: ...`, including an empty expression.
- An unknown method (`unknown method: foo`), `has()` or `contains()` without exactly one argument (`has() requires exactly 1 argument, got 0`) or an argument that is not a CSV record (`failed to parse args: ...`), reported when the query is parsed.
- A comparison side that yields more than one value.
- An error from the expression of `sort()` or `rsort()`.

A step never fails because of the data it reads.

## Edits

`Edit(&n, expr)` and tq `-e` take `<query> = <value>` (set), `<query> += <value>` (append) or `<query> ^?` (delete). The expression is split at the first `=`, `+=` or `^?` outside quotes, parentheses and method arguments, and `<value>` is parsed as a single YAML document, so `.url = "http://x?a=1"` sets the whole URL and `.a = [1, 2]` sets an array. Comparison operators (`==`, `!=`, `<=`, `>=`, `~=`) are never read as an assignment, so `.a[.b == 1].c = 2` splits after `]` and `.a ==1` is a syntax error; write `.a = "=1"` to set that string.

### Go equivalents

Where a query step reads with `Get`, the edit of that step writes with `Set` and `Delete` on the same receiver.

| Edit | Go equivalent |
| - | - |
| `.key = v`, `[i] = v` | `n.Set(key, v)`, `n.Set(i, v)` |
| `.key += v`, `[i] += v` | `n.Set(key, append(n.Get(key).Array(), v))` when the value is an array or `null`, else an error |
| `.key ^?`, `[i] ^?` | `n.Delete(key)`, `n.Delete(i)` |
| `first()`, `last()` | `[0]` and `[-1]` on an array; an error on an object |
| `. = v` | `*pn = v` |
| `. += v` | `*pn = append(a, v)` for an array `a`, `Array{v}` for `null`, else an error |
| `. ^?` | an error |

So, like `Map.Set` and `(*Array).Set`:

- On an object, `[i]` writes the key `"i"`, including a negative `i`: `.m[-1] = 1` sets `"-1"`.
- On an array, `.key` writes the index `key` when it is a non-negative integer, and fails otherwise.
- Writing past the end of an array grows it, filling the gap with `null`; an index of `1<<24` or more fails.
- Deleting a missing key or an index past the end changes nothing, while a negative index before the start fails, as it does for `=` and `+=`.
- `^?` on a plain `Array` shifts its elements in place, as `(*Array).Delete` does, so another slice sharing its backing array sees the shift; copy the array first, or use a `*Array`.
- `=` on a key with a `null` value stores `null` and keeps the key, while `^?` removes it: `.a = null` and `.a ^?` differ.
- A step on a scalar fails, unlike the same step in a query: a key, an index, `first()`, `last()`, `[]`, a select and a slice.
- A step through an `Any` or `*Any` edits the node it wraps, so a wrapped `*Array` or other `EditorNode` changes in place and the wrapper stays; a wrapped plain `Array` may be replaced by the edited `Array`, since an `Array` value cannot grow or shrink in place. `+=` stores the appended `Array` as the value (see the table above), so an `Any` or `*Array` that is itself appended to is replaced by a plain `Array`, while `.a[0] += v` inside a `*Array` edits it in place.

### Paths

An edit writes through keys, indexes, `first()` and `last()`, and through `[]`, selects, slices and `..key`, which edit every match. Any other step, such as another method, a literal, or a parenthesised group containing a pipe inside a longer path, is a syntax error before any data is read (`.a.sort()[0] = 1` fails with `syntax error: unsupported edit query: sort()`). `[]`, a select, a slice or a trailing `|` (as in jq's `.a |= 1`) is also a syntax error as the last step. Parentheses around steps without a pipe are ignored, so `(.a[0]).b = 1` edits like `.a[0].b = 1`, and so are parentheses around the whole query: `(.a | .[0]) = 9` edits like `.a | .[0] = 9`, which changes only the array `|` collects, not the document.

Bare `.` and `..` steps are ignored, so `.[0]` edits like `[0]`, `.a.` like `.a` and `..b. ^?` deletes like `..b ^?`. After `[]`, a select or a slice, a trailing `.` names the matched elements themselves: `.a[]. = 1` sets every element of `a`, `null` elements included, `.a[]. += 1` turns a `null` element into `[1]` as `.a[0] += 1` would, and `.a[]. ^?` deletes every element, `null` elements included, as `.a[0] ^?` would, so `.a[. == 1]. ^?` deletes the elements equal to `1` and leaves the others in place, as jq's `del(.a[] | select(. == 1))` does, and on an object it deletes the matching keys: `.m[. == 1]. ^?` deletes every key of `m` whose value is `1`.

On `null`, `[]`, a select and a slice edit nothing. A slice edits each element in range as `[i]` would.

### Multiple matches

An edit through `..key`, `[]`, a select or a slice gives each match its own copy of the RHS, so editing one match later leaves the others as they are. A `..key` edit also edits inner matches before outer ones, so it never edits inside the RHS, i.e. a value it has just assigned: `..b = {"b": 1}` on `{"b": {"b": 2}}` gives `{"b": {"b": 1}}`. If an inner match does not fit the rest of the path, or `+=` finds a value that is not an array or `null`, the edit stops with an error, as in jq: `..b.c = 1` on `{"b": {"c": {"b": []}}}` fails with `cannot index array with c`, and `..a += 1` on `{"a": 1, "b": {"a": []}}` sets `b.a` to `[1]` and then fails with `cannot append to "a"`. `..key` with a numeric `key` edits arrays too, inner arrays first: `..0 = 9` on `{"a": [[1], [2]]}` gives `{"a": [9, [9]]}`. Edits are not transactional: when a match fails, the matches edited before it stay edited, except that `^?` with a trailing `.` deletes from a container only after its selector has accepted or rejected every element, so a selector that fails there deletes nothing from that container (see [Edit errors](#edit-errors)).

### Null intermediates

A `null` reached by a step, including a Go `nil` value such as a hole left by growing an array, a nil pointer node such as `(*tree.Array)(nil)` and a nil `Map` or `Array`, is treated as missing:

- `=` and `+=` replace it with what the rest of the path needs when every later step is a key, a non-negative index or `first()`: `.a[0].x = 1` on `{"a": [null]}` gives `{"a": [{"x": 1}]}`, and `.a += 1` on `{"a": null}` gives `{"a": [1]}`.
- Otherwise it stays as it is: `.a.b[].c = 1` and `.a.last() = 1` on `{"a": null}` change nothing: `a` would become an empty array, which has no last element to count from the end. `^?` never replaces it, and deletes it only when it is what `^?` names: `.a.b ^?` on `{"a": null}` changes nothing, while `.a ^?` gives `{}` and `.a[]. ^?` on `{"a": [null]}` gives `{"a": []}`.

A `null` root is replaced the same way: `.a.b = 1` on `null` gives `{"a": {"b": 1}}`, `. += 1` on `null` gives `[1]`, and `.[-1] = 1` on `null` changes nothing.

An array whose `Array()` is nil, such as a pointer to a nil `Array`, edits as an empty array: `[]`, a select and a slice change nothing, and `+=` gives `[v]`.

### Edit errors

An edit fails for a syntax error, an unsupported step (see [Paths](#paths)), more than one YAML document in the value (`syntax error: multiple YAML documents in edit value ...`), and for these conditions, reported when the edit reaches them:

| Condition | Example | Message |
| - | - | - |
| A key or index on a scalar | `.a.b = 1` on `{"a": 1}` | `cannot index array with "b"` |
| A key that is not a non-negative integer on an array | `.a.b = 1` on `{"a": []}` | `cannot index array with b` |
| A negative index before the start of an array | `.a[-5] = 1` on `{"a": []}` | `cannot index array with -5`; `+=` and `^?` say `cannot append to array with -5` and `cannot delete array with -5` |
| `+=` on a value that is not an array or `null` | `.a += 1` on `{"a": 1}` | `cannot append to "a"` |
| `^?` on a scalar | `.a.b ^?` on `{"a": 1}` | `cannot delete "b"` |
| `first()` or `last()` on an object | `.a.first() = 1` on `{"a": {}}` | `cannot edit first() of a map` |
| `[]` or a select on a scalar | `.a[]. = 1` on `{"a": 1}` | `cannot index array with []` |
| A slice on a scalar | `.a[1:]. = 1` on `{"a": 1}` | `cannot index array with range [1:]` |
| An index of `1<<24` or more | `.a[16777216] = 1` | `array index 16777216 exceeds maximum 16777215` |
| `+=` on a root, or on an element a trailing `.` names, that is not an array or `null` | `. += 1` on `1`, `.a[]. += 1` on `{"a": [1]}` | `cannot append to .` |
| `^?` on the root itself | `. ^?` | `cannot delete .` |

An edit whose steps are all keys, indexes, `first()` and `last()` leaves the document as it was when it fails, a `null` root included: `.a[16777216] = 1` on `{}` leaves `{}` and on `null` leaves `null`. An edit with several matches keeps the matches edited before the failure (see [Multiple matches](#multiple-matches)); `^?` with a trailing `.` runs the selector on every element or value of a container before it deletes any of them, so a selector that fails deletes nothing from that container, while the containers it deleted from before, through `..key` or an earlier `[]`, stay edited: `..a[.t[] == 1]. ^?` on `{"x": {"a": [{"t": [1]}]}, "y": {"a": [{"t": [1, 2]}]}}` empties `x.a` and then fails on `y.a`. A caller that needs the document untouched after any failure edits a deep copy, `c := tree.CloneDeep(n)` (`Clone` copies only the top level); tq `-U` writes a file only when every edit on it succeeds.

Messages that say "array" for another receiver, and whether they quote the key, are tracked in [#98](https://github.com/mojatter/tree/issues/98).

## Deviations

From the Go API, on purpose:

- A negative index counts from the end of an array in queries and edits, while `Array.Get(-1)` gives `Nil` and `(*Array).Set(-1, v)` fails.
- An edit replaces a `null` intermediate or root with what the rest of the path needs (see [Null intermediates](#null-intermediates)); `Nil` has no `Set`.
- `first()` and `last()` give `null` on an object, while `Map.Get(0)` reads the key `"0"`: a key is not a position, and `[0]` already reads that key as `Get` does.
- Multi-valued steps (`[]`, selects, slices, `..key`, pipes and the other methods) have no single Go call.

From jq:

- A key or an index on a scalar and a key on an array read `null` instead of failing, and an index on an object reads the key with that number.
- `|` collects results into one array instead of streaming them, see [Pipes](#pipes).
- An edit through `|` edits the array `|` collects, not the document, see [Paths](#paths); there is no `|=`.
- Each side of a comparison yields at most one value, and an ordered operator with `null`, a boolean or a container never matches.
- `0`, negative numbers and empty strings, arrays and objects are falsy in a select.
- `count()` on a string is `0`.
- `[]` on an object yields its values in sorted key order.
- `^?` with a negative index before the start of an array fails, while jq's `del(.[-5])` ignores it.
- `+=` appends `v` as one element, so `.a += [1]` on `{"a": [0]}` gives `{"a": [0, [1]]}`; jq's `+=` adds, and would give `[0, 1]`.
