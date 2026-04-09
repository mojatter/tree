package examples

import (
	"fmt"
	"log"

	goyaml "github.com/goccy/go-yaml"
	"github.com/mojatter/tree"
	yamlv3 "go.yaml.in/yaml/v3"
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

	// NOTE: goccy/go-yaml used to be able to unmarshal directly into a
	// tree.Map in the same style as Example_v3YAMLUnmarshal, but newer
	// versions of goccy/go-yaml no longer do. As a workaround, decode
	// into a plain map[string]any first and then convert it to a
	// tree.Map via tree.ToNode. Sorry for the extra step.
	var raw map[string]any
	if err := goyaml.Unmarshal(data, &raw); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%+v\n", raw)

	group := tree.ToNode(raw).Map()
	fmt.Printf("%+v\n", group)

	// Output:
	// map[Colors:[Crimson Red Ruby Maroon] ID:1 Name:Reds]
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
