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
	// ErrCantOpen is returned by NewExtractor when a language model file can't
	// be loaded.
	ErrCantOpen = errors.New("Unable to open model file")
	// ErrMemory occurs when underlying C structs cannot be allocated.
	ErrMemory = errors.New("Could not allocate memory")
)

// Tokenize returns a slice that contains a tokenized copy of the input text.
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

// TokenizeWithOffests is identical to calling Tokenize(text)
// but it also outputs the positions of each token within the input text data.
func TokenizeWithOffsets(text string) ([]string, []uint32) {
	cs := C.CString(text)
	defer C.free(unsafe.Pointer(cs))
	var cOffsets *C.ulong
	defer C.free(unsafe.Pointer(cOffsets))
	ctokens := C.mitie_tokenize_with_offsets(cs, &cOffsets)
	defer C.mitie_free(unsafe.Pointer(ctokens))
	i := 0
	// a hack since mitie arrays are NULL terminated.
	p := (*[1 << 30]*C.char)(unsafe.Pointer(ctokens))
	q := (*[1 << 30]C.ulong)(unsafe.Pointer(cOffsets))
	tokens := make([]string, 0, 20)
	offsets := make([]uint32, 0, 20)
	for p[i] != nil {
		tokens = append(tokens, C.GoString(p[i]))
		offsets = append(offsets, uint32(q[i]))
		i++
	}
	return tokens, offsets
}

// Range specifies the position of an Entity within a token slice.
type Range struct {
	Start int
	End   int
}

// Entity is a detected entity.
type Entity struct {
	Score     float64
	Tag       int
	TagString string
	Name      string
	Range     Range
}

// Extractor detects entities based on a language model file.
type Extractor struct {
	ner  *C.mitie_named_entity_extractor
	tags []string // E.g. PERSON or LOCATION, etc…
}

// NewExtractor returns an Extractor given the path to a language model.
func NewExtractor(path string) (*Extractor, error) {
	model := C.CString(path)
	defer C.free(unsafe.Pointer(model))
	ner := C.mitie_load_named_entity_extractor(model)
	if ner == nil {
		return nil, ErrCantOpen
	}

	num := int(C.mitie_get_num_possible_ner_tags(ner))
	tags := make([]string, num, num)
	for i := 0; i < num; i++ {
		tags[i] = C.GoString(C.mitie_get_named_entity_tagstr(ner, C.ulong(i)))
	}

	return &Extractor{
		ner:  ner,
		tags: tags,
	}, nil
}

// Free frees the underlying used C memory.
func (ext *Extractor) Free() {
	C.mitie_free(unsafe.Pointer(ext.ner))
}

// Tags returns a slice of Tags that are part of this language model.
// E.g. PERSON or LOCATION, etc…
func (ext *Extractor) Tags() []string {
	return ext.tags
}

func (ext *Extractor) tagString(index int) string {
	return C.GoString(C.mitie_get_named_entity_tagstr(ext.ner, C.ulong(index)))
}

// Extract runs the extractor and returns a slice of Entities found in the
// given tokens. It is a convenience function.
func (ext *Extractor) Extract(tokens []string) ([]Entity, error) {
	extraction, err := ext.NewExtraction(tokens)
	if err != nil {
		return nil, err
	}
	defer extraction.Free()
	return extraction.Entities, nil
}

// Extraction describes the result of an extract run.
type Extraction struct {
	Tokens    []string
	Entities  []Entity
	extractor *Extractor
	ctokens   **C.char
	dets      *C.struct_mitie_named_entity_detections
	numDets   int
}

// NewExtraction completes an extraction task and returns the extraction results for future use in relationship extraction.
func (ext *Extractor) NewExtraction(tokens []string) (*Extraction, error) {
	extn := &Extraction{
		extractor: ext,
		Tokens:    tokens,
	}
	extn.ctokens = C.ner_arr_make(C.int(len(tokens)) + 1) // NULL termination
	for i, t := range tokens {
		cs := C.CString(t) // released by ner_arr_free
		C.ner_arr_set(extn.ctokens, cs, C.int(i))
	}

	extn.dets = C.mitie_extract_entities(ext.ner, extn.ctokens)
	if extn.dets == nil {
		C.ner_arr_free(extn.ctokens, C.int(len(extn.Tokens))+1)
		return nil, ErrMemory
	}

	extn.numDets = int(C.mitie_ner_get_num_detections(extn.dets))

	extn.Entities = make([]Entity, extn.numDets, extn.numDets)

	tagNames := ext.Tags()
	for i := 0; i < extn.numDets; i++ {
		pos := int(C.mitie_ner_get_detection_position(extn.dets, C.ulong(i)))
		len := int(C.mitie_ner_get_detection_length(extn.dets, C.ulong(i)))
		tagID := int(C.mitie_ner_get_detection_tag(extn.dets, C.ulong(i)))

		extn.Entities[i] = Entity{
			Tag:       tagID,
			TagString: tagNames[tagID],
			Score:     float64(C.mitie_ner_get_detection_score(extn.dets, C.ulong(i))),
			Name:      strings.Join(extn.Tokens[pos:pos+len], " "),
			Range:     Range{pos, pos + len},
		}
	}

	return extn, nil
}

// Free the C mamory used by the extraction.
func (extn *Extraction) Free() {
	C.ner_arr_free(extn.ctokens, C.int(len(extn.Tokens))+1)
	C.mitie_free(unsafe.Pointer(extn.dets))
}
