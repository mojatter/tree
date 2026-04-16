package examples

import (
	"fmt"
	"log"

	"github.com/mojatter/tree"
)

type evenIndexQuery struct{}

func (q *evenIndexQuery) Exec(n tree.Node) ([]tree.Node, error) {
	if !n.Type().IsArray() {
		return nil, nil
	}
	var rs []tree.Node
	for i, r := range n.Array() {
		if i%2 == 0 {
			rs = append(rs, r)
		}
	}
	return rs, nil
}

func (q *evenIndexQuery) String() string {
	return "even-index-query"
}

func Example_customQuery() {
	group := tree.Array{
		tree.Map{
			"ID":     tree.V(1),
			"Name":   tree.V("Reds"),
			"Colors": tree.ToArrayValues("Crimson", "Red", "Ruby", "Maroon"),
		},
		tree.Map{
			"ID":     tree.V(2),
			"Name":   tree.V("Greens"),
			"Colors": tree.ToArrayValues("Green", "Lime", "Olive", "Teal"),
		},
		tree.Map{
			"ID":     tree.V(3),
			"Name":   tree.V("Blues"),
			"Colors": tree.ToArrayValues("Aqua", "Blue", "Cyan", "SkyBlue"),
		},
	}

	q := tree.FilterQuery{
		tree.SelectQuery{},
		tree.MapQuery("Colors"),
		&evenIndexQuery{},
	}

	rs, err := q.Exec(group)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%v\n", rs)

	// Output:
	// [Crimson Ruby Green Olive Aqua Cyan]
}

type primaryColorSelector struct {
}

func (s *primaryColorSelector) Matches(n tree.Node) (bool, error) {
	if !n.Type().IsStringValue() {
		return false, nil
	}
	switch n.Value().String() {
	case "Red", "Green", "Blue":
		return true, nil
	}
	return false, nil
}

func (s *primaryColorSelector) String() string {
	return "odd-index-selector"
}

func Example_customSelector() {
	group := tree.Array{
		tree.Map{
			"ID":     tree.V(1),
			"Name":   tree.V("Reds"),
			"Colors": tree.ToArrayValues("Crimson", "Red", "Ruby", "Maroon"),
		},
		tree.Map{
			"ID":     tree.V(2),
			"Name":   tree.V("Greens"),
			"Colors": tree.ToArrayValues("Green", "Lime", "Olive", "Teal"),
		},
		tree.Map{
			"ID":     tree.V(3),
			"Name":   tree.V("Blues"),
			"Colors": tree.ToArrayValues("Aqua", "Blue", "Cyan", "SkyBlue"),
		},
	}

	q := tree.FilterQuery{
		tree.SelectQuery{},
		tree.MapQuery("Colors"),
		tree.SelectQuery{Selector: &primaryColorSelector{}},
	}

	rs, err := q.Exec(group)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%v\n", rs)

	// Output:
	// [Red Green Blue]
}
