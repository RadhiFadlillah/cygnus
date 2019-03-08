// +build ignore

package main

import (
	"log"
	"net/http"

	"github.com/shurcooL/vfsgen"
)

func main() {
	err := vfsgen.Generate(http.Dir("view"), vfsgen.Options{
		Filename:     "handler/assets-prod.go",
		PackageName:  "handler",
		BuildTags:    "!dev",
		VariableName: "assets",
	})

	if err != nil {
		log.Fatalln(err)
	}
}
