package examples

import (
	"fmt"
	"log"

	gojson "github.com/goccy/go-json"
	"github.com/mojatter/tree"
	jsoniter "github.com/json-iterator/go"
)

func Example_goJSONUnmarshal() {
	data := []byte(`[
  {"Name": "Platypus", "Order": "Monotremata"},
  {"Name": "Quoll",    "Order": "Dasyuromorphia"}
]`)

	var animals tree.Array
	err := gojson.Unmarshal(data, &animals)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%+v\n", animals)

	// Output:
	// [map[Name:Platypus Order:Monotremata] map[Name:Quoll Order:Dasyuromorphia]]
}

func Example_goJSONUnmarshal_combined() {
	data := []byte(`[
  {"Name": "Platypus", "Order": "Monotremata"},
  {"Name": "Quoll",    "Order": "Dasyuromorphia"}
]`)
	type Animal struct {
		Name  string
		Order tree.StringValue
	}
	var animals []Animal
	err := gojson.Unmarshal(data, &animals)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%+v\n", animals)

	// Output:
	// [{Name:Platypus Order:Monotremata} {Name:Quoll Order:Dasyuromorphia}]
}

func Example_goJSONMarshal() {
	group := tree.Map{
		"ID":     tree.ToValue(1),
		"Name":   tree.ToValue("Reds"),
		"Colors": tree.ToArrayValues("Crimson", "Red", "Ruby", "Maroon"),
	}
	b, err := gojson.Marshal(group)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))

	// Output:
	// {"Colors":["Crimson","Red","Ruby","Maroon"],"ID":1,"Name":"Reds"}
}

func Example_jsonIteratorUnmarshal() {
	json := jsoniter.ConfigCompatibleWithStandardLibrary

	data := []byte(`[
  {"Name": "Platypus", "Order": "Monotremata"},
  {"Name": "Quoll",    "Order": "Dasyuromorphia"}
]`)

	var animals tree.Array
	err := json.Unmarshal(data, &animals)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%+v\n", animals)

	// Output:
	// [map[Name:Platypus Order:Monotremata] map[Name:Quoll Order:Dasyuromorphia]]
}

func Example_jsonIteratorUnmarshal_combined() {
	json := jsoniter.ConfigCompatibleWithStandardLibrary

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

func Example_jsonIteratorMarshal() {
	json := jsoniter.ConfigCompatibleWithStandardLibrary

	group := tree.Map{
		"ID":     tree.ToValue(1),
		"Name":   tree.ToValue("Reds"),
		"Colors": tree.ToArrayValues("Crimson", "Red", "Ruby", "Maroon"),
	}
	b, err := json.Marshal(group)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))

	// Output:
	// {"Colors":["Crimson","Red","Ruby","Maroon"],"ID":1,"Name":"Reds"}
}
