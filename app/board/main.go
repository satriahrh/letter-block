package main

import (
	"fmt"
)

func main() {
	alphabet := "abcdefghijklmnopqrstuvwxyz"
	boardBase := []uint8{
		5,
		24,
		10,
		0,
		9,
		6,
		15,
		19,
		7,
		5,
		1,
		23,
		3,
		18,
		6,
		23,
		16,
		8,
		17,
		8,
		9,
		17,
		6,
		0,
		21,
	}

	for i, b := range boardBase {
		fmt.Printf("%v %v --- ", i, string(alphabet[b]))
	}

}
