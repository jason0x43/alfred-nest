package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) <= 1 {
		println("Not enough arguments")
		return
	}

	file, err := os.Create(os.Args[1])
	if err != nil {
		println("Error opening file")
		return
	}

	// This is barely an obfuscation, but the goal is just to make the key
	// *slightly* less obvious in a string dump
	secret := os.Getenv("NEST_CLIENT_SECRET")

	file.WriteString("package main\n")
	file.WriteString("func init() {\n")
	file.WriteString("	secret := []byte{\n")
	for _, r := range secret {
		fmt.Fprintf(file, "		0x%x,\n", r^0xFF)
	}
	file.WriteString("	}\n")
	file.WriteString("	for _, r := range secret {\n")
	file.WriteString("		ClientSecret += string(r^0xFF)\n")
	file.WriteString("	}\n")
	file.WriteString("}\n")

	file.Close()
}
