package examples

import (
	"fmt"
	"log"

	goyaml "github.com/goccy/go-yaml"
	"github.com/mojatter/tree"
	yamlv3 "gopkg.in/yaml.v3"
)

func Example_v3YAMLUnmarshal() {
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
	if err := yamlv3.Unmarshal(data, &group); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%+v\n", group)

	// Output:
	// map[Colors:[Crimson Red Ruby Maroon] ID:1 Name:Reds]
}

func Example_yamlV3Marshal() {
	group := tree.Map{
		"ID":     tree.ToValue(1),
		"Name":   tree.ToValue("Reds"),
		"Colors": tree.ToArrayValues("Crimson", "Red", "Ruby", "Maroon"),
	}
	b, err := yamlv3.Marshal(group)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))

	// Output:
	// Colors:
	//     - Crimson
	//     - Red
	//     - Ruby
	//     - Maroon
	// ID: 1
	// Name: Reds
}

func Example_goYAMLUnmarshal() {
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
	if err := goyaml.Unmarshal(data, &group); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%+v\n", group)

	// Output:
	// map[Colors:[Crimson Red Ruby Maroon] ID:1 Name:Reds]
}

func Example_goYAMLMarshal() {
	group := tree.Map{
		"ID":     tree.ToValue(1),
		"Name":   tree.ToValue("Reds"),
		"Colors": tree.ToArrayValues("Crimson", "Red", "Ruby", "Maroon"),
	}
	b, err := goyaml.Marshal(group)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))

	// Output:
	// Colors:
	// - Crimson
	// - Red
	// - Ruby
	// - Maroon
	// ID: 1.0
	// Name: Reds
}
