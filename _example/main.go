package main

import (
	"flag"
	"io/ioutil"
	"log"

	"github.com/sbl/ner"
)

func main() {
	path := flag.String("model-path", "/usr/local/share/MITIE-models/english/ner_model.dat", "path to mitie model data")
	flag.Parse()

	ext, err := ner.NewExtractor(*path)
	if err != nil {
		log.Fatal(err)
	}
	defer ext.Free()

	log.Printf("available tags: %+v", ext.Tags())

	txt, err := ioutil.ReadFile("11231.txt")
	if err != nil {
		log.Fatal(err)
	}

	tokens := ner.Tokenize(string(txt))

	es, err := ext.Extract(tokens)
	if err != nil {
		log.Fatal(err)
	}

	for _, v := range es {
		log.Printf("%+v", v)
	}
}
