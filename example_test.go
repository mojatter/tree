package tree_test

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/mojatter/tree"
	"go.yaml.in/yaml/v3"
)

func ExampleMarshalJSON() {
	group := tree.Map{
		"ID":     tree.V(1),
		"Name":   tree.V("Reds"),
		"Colors": tree.A("Crimson", "Red", "Ruby", "Maroon"),
	}
	b, err := json.Marshal(group)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))

	// Output:
	// {"Colors":["Crimson","Red","Ruby","Maroon"],"ID":1,"Name":"Reds"}
}

func ExampleMarshalJSON_combined() {
	type ColorGroup struct {
		ID     int
		Name   string
		Colors tree.Array
	}
	group := ColorGroup{
		ID:     1,
		Name:   "Reds",
		Colors: tree.A("Crimson", "Red", "Ruby", "Maroon"),
	}
	b, err := json.Marshal(group)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))

	// Output:
	// {"ID":1,"Name":"Reds","Colors":["Crimson","Red","Ruby","Maroon"]}
}

func ExampleUnmarshalJSON() {
	data := []byte(`[
  {"Name": "Platypus", "Order": "Monotremata"},
  {"Name": "Quoll",    "Order": "Dasyuromorphia"}
]`)

	var animals tree.Array
	err := json.Unmarshal(data, &animals)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%v\n", animals)

	// Output:
	// [map[Name:Platypus Order:Monotremata] map[Name:Quoll Order:Dasyuromorphia]]
}

func ExampleUnmarshalJSON_any() {
	data := []byte(`[
  {"Name": "Platypus", "Order": "Monotremata"},
  {"Name": "Quoll",    "Order": "Dasyuromorphia"}
]`)

	var animals tree.Any
	err := json.Unmarshal(data, &animals)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%v\n", animals.Type().IsArray())
	fmt.Printf("%v\n", animals.Array())

	// Output:
	// true
	// [map[Name:Platypus Order:Monotremata] map[Name:Quoll Order:Dasyuromorphia]]
}

func ExampleUnmarshalJSON_combined() {
	data := []byte(`[
  {"Name": "Platypus", "Order": "Monotremata"},
  {"Name": "Quoll",    "Order": "Dasyuromorphia"}
]`)
	type Animal struct {
		Name  string
		Order tree.StringValue
	}
	var animals []Animal
	err := json.Unmarshal(data, &animals)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%+v\n", animals)

	// Output:
	// [{Name:Platypus Order:Monotremata} {Name:Quoll Order:Dasyuromorphia}]
}

func ExampleMarshalYAML() {
	group := tree.Map{
		"ID":     tree.V(1),
		"Name":   tree.V("Reds"),
		"Colors": tree.A("Crimson", "Red", "Ruby", "Maroon"),
	}
	b, err := tree.MarshalYAML(group)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))

	// Output:
	// Colors:
	//   - Crimson
	//   - Red
	//   - Ruby
	//   - Maroon
	// ID: 1
	// Name: Reds
}

func ExampleUnmarshalYAML() {
	data := []byte(`---
Colors:
- Crimson
- Red
- Ruby
- Maroon
ID: 1
Name: Reds
`)

	var group tree.Map
	if err := yaml.Unmarshal(data, &group); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%+v\n", group)

	// Output:
	// map[Colors:[Crimson Red Ruby Maroon] ID:1 Name:Reds]
}

func ExampleNode_Get() {
	group := tree.Map{
		"ID":     tree.V(1),
		"Name":   tree.V("Reds"),
		"Colors": tree.A("Crimson", "Red", "Ruby", "Maroon"),
		"Nil":    nil,
	}
	fmt.Println(group.Get("Colors").Get(1))
	fmt.Println(group.Get("Colors", 2))
	fmt.Println(group.Get("Colors").Get(5).IsNil())

	// Get returns NilValue for both missing keys and nil values.
	// Use Has to distinguish between the two.
	fmt.Println(group.Get("Nil").IsNil())
	fmt.Println(group.Has("Nil"))
	fmt.Println(group.Get("NoSuchKey").IsNil())
	fmt.Println(group.Has("NoSuchKey"))

	// Output:
	// Red
	// Ruby
	// true
	// true
	// true
	// true
	// false
}

func ExampleFind() {
	group := tree.Map{
		"ID":     tree.V(1),
		"Name":   tree.V("Reds"),
		"Colors": tree.A("Crimson", "Red", "Ruby", "Maroon"),
	}

	rs, err := group.Find(".Colors[1:3]")
	if err != nil {
		log.Fatal(err)
	}
	for _, r := range rs {
		fmt.Println(r)
	}

	// Output:
	// Red
	// Ruby
}

func ExampleEdit() {
	var group tree.Node = tree.Map{
		"ID":     tree.V(1),
		"Name":   tree.V("Reds"),
		"Colors": tree.A("Crimson", "Red", "Ruby", "Maroon"),
	}

	if err := tree.Edit(&group, ".Colors += \"Pink\""); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Append Pink to Colors:\n  %+v\n", group)

	if err := tree.Edit(&group, ".Name = \"Blue\""); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Set Blue to Name:\n  %+v\n", group)

	if err := tree.Edit(&group, ".Colors ^?"); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Delete Colors:\n  %+v\n", group)

	// Output:
	// Append Pink to Colors:
	//   map[Colors:[Crimson Red Ruby Maroon Pink] ID:1 Name:Reds]
	// Set Blue to Name:
	//   map[Colors:[Crimson Red Ruby Maroon Pink] ID:1 Name:Blue]
	// Delete Colors:
	//   map[ID:1 Name:Blue]
}
