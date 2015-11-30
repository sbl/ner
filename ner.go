package ner

/*
#cgo LDFLAGS: -lmitie

#include <stdlib.h>
#include <stdio.h>
#include "mitie.h"

static char** ner_arr_make(int size) {
	return calloc(sizeof(char*), size);
}

static void ner_arr_set(char **a, char *s, int n) {
	a[n] = s;
}

static void ner_arr_free(char **a, int size) {
	int i;
	for (i = 0; i < size; i++) {
		free(a[i]);
	}
	free(a);
}
*/
import "C"
import (
	"errors"
	"strings"
	"unsafe"
)

var (
	ErrCantOpen = errors.New("Unable to open model file")
	ErrMemory   = errors.New("Could not allocate memory")
)

func Tokenize(text string) []string {
	cs := C.CString(text)
	defer C.free(unsafe.Pointer(cs))
	ctokens := C.mitie_tokenize(cs)
	defer C.mitie_free(unsafe.Pointer(ctokens))
	i := 0
	// a hack since mitie arrays are NULL terminated.
	p := (*[1 << 30]*C.char)(unsafe.Pointer(ctokens))
	tokens := make([]string, 0, 20)
	for p[i] != nil {
		tokens = append(tokens, C.GoString(p[i]))
		i++
	}
	return tokens
}

type Range struct {
	Start int
	End   int
}

type Entity struct {
	Score float64
	Tag   int
	Name  string
	Range Range
}

type Extractor struct {
	ner *C.mitie_named_entity_extractor
}

func NewExtractor(path string) (*Extractor, error) {
	model := C.CString(path)
	defer C.free(unsafe.Pointer(model))
	ner := C.mitie_load_named_entity_extractor(model)
	if ner == nil {
		return nil, ErrCantOpen
	}

	return &Extractor{
		ner: ner,
	}, nil
}

func (ner *Extractor) Free() {
	C.mitie_free(unsafe.Pointer(ner.ner))
}

func (ext *Extractor) Tags() []string {
	num := int(C.mitie_get_num_possible_ner_tags(ext.ner))
	tags := make([]string, num, num)
	for i := 0; i < num; i++ {
		tags[i] = ext.TagString(i)
	}
	return tags
}

func (ext *Extractor) TagString(index int) string {
	return C.GoString(C.mitie_get_named_entity_tagstr(ext.ner, C.ulong(index)))
}

func (ext *Extractor) Extract(tokens []string) ([]Entity, error) {
	ctokens := C.ner_arr_make(C.int(len(tokens)) + 1) // NULL termination
	defer C.ner_arr_free(ctokens, C.int(len(tokens))+1)
	for i, t := range tokens {
		cs := C.CString(t) // released by ner_arr_free
		C.ner_arr_set(ctokens, cs, C.int(i))
	}

	dets := C.mitie_extract_entities(ext.ner, ctokens)
	defer C.mitie_free(unsafe.Pointer(dets))
	if dets == nil {
		return nil, ErrMemory
	}

	n := int(C.mitie_ner_get_num_detections(dets))
	entities := make([]Entity, n, n)

	for i := 0; i < n; i++ {
		pos := int(C.mitie_ner_get_detection_position(dets, C.ulong(i)))
		len := int(C.mitie_ner_get_detection_length(dets, C.ulong(i)))

		entities[i] = Entity{
			Tag:   int(C.mitie_ner_get_detection_tag(dets, C.ulong(i))),
			Score: float64(C.mitie_ner_get_detection_score(dets, C.ulong(i))),
			Name:  strings.Join(tokens[pos:pos+len], " "),
			Range: Range{pos, pos + len},
		}
	}
	return entities, nil
}
