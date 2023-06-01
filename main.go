package main

import (
	"github.com/savannahostrowski/ghost/cmd"
)



func main() {
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
