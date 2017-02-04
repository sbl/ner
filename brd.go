package ner

/*
#include <stdlib.h>
#include <stdio.h>
#include "mitie.h"
*/
import "C"
import (
	"errors"
	"io/ioutil"
	"path/filepath"
	"unsafe"
)

var (
	// ErrIncompatibleNER occurs when an incompatible NER object is used with the relation detector.
	ErrIncompatibleNER = errors.New("an incompatible NER object was used with the relation detector")
)

// Detector of binary relationships.
type Detector struct {
	brd *C.mitie_binary_relation_detector
}

// NewDetector returns a Detector given the path to a binary relationship detection model.
func NewDetector(path string) (*Detector, error) {
	model := C.CString(path)
	defer C.free(unsafe.Pointer(model))
	brd := C.mitie_load_binary_relation_detector(model)
	if brd == nil {
		return nil, ErrCantOpen
	}

	return &Detector{
		brd: brd,
	}, nil
}

// Free frees the underlying used C memory.
func (det *Detector) Free() {
	C.mitie_free(unsafe.Pointer(det.brd))
}

// String gives the name of the detector.
func (det *Detector) String() string {
	return C.GoString(C.mitie_binary_relation_detector_name_string(det.brd))
}

// AllDetectorsFromDir a directory can be loaded using this utility.
func AllDetectorsFromDir(svmModelDir string) (detectors []*Detector, err error) {

	files, err := ioutil.ReadDir(svmModelDir)
	if err != nil {
		return nil, err
	}
	for _, fi := range files {
		if filepath.Ext(fi.Name()) == ".svm" {
			svmPath := svmModelDir + string(filepath.Separator) + fi.Name()
			det, err := NewDetector(svmPath)
			if err != nil {
				return nil, err
			}
			detectors = append(detectors, det)
		}
	}

	return detectors, nil
}

type Relation struct {
	Relationship string
	From, To     Range
	Score        float64
}

// Detect binary relationships in the results of a previous extraction.
func (extn *Extraction) Detect(detectors []*Detector) ([]Relation, error) {

	ret := []Relation{}

	// Now let's scan along the entities and ask the relation detector which pairs of
	// entities are instances of the type of relation we are looking for.
	for i := 0; i+1 < extn.numDets; i++ {
		rels, err := detectRelation(detectors, extn.extractor.ner, extn.ctokens, extn.dets, C.ulong(i), C.ulong(i+1))
		if err != nil {
			return nil, err
		}
		ret = append(ret, rels...)

		// Relations have an ordering to their arguments.  So even if the above
		// relation check failed we still might have a valid relation if we try
		// swapping the two arguments.  So that's what we do here.
		rels, err = detectRelation(detectors, extn.extractor.ner, extn.ctokens, extn.dets, C.ulong(i+1), C.ulong(i))
		if err != nil {
			return nil, err
		}
		ret = append(ret, rels...)
	}

	return ret, nil
}

// detectRelation logic copied from MITIE C example
func detectRelation(
	detectors []*Detector,
	ner *C.mitie_named_entity_extractor,
	tokens **C.char,
	dets *C.mitie_named_entity_detections,
	idx1 C.ulong,
	idx2 C.ulong,
) (rels []Relation, err error) {
	idx1pos := C.mitie_ner_get_detection_position(dets, idx1)
	idx1len := C.mitie_ner_get_detection_length(dets, idx1)
	idx2pos := C.mitie_ner_get_detection_position(dets, idx2)
	idx2len := C.mitie_ner_get_detection_length(dets, idx2)

	if C.mitie_entities_overlap(idx1pos, idx1len, idx2pos, idx2len) != 0 {
		return nil, nil
	}

	// The relation detection process in MITIE has two steps.  First you extract a set of
	// "features" that describe a particular relation mention.  Then you call
	// mitie_classify_binary_relation() on those features and see if it is an instance of a
	// particular kind of relation.  The reason we have this two step process is because,
	// in many applications, you will have a large set of relation detectors you need to
	// evaluate for each possible relation instance and it is more efficient to perform the
	// feature extraction once and then reuse the results for multiple calls to
	// mitie_classify_binary_relation().  However, in this case, we are simply running one
	// type of relation detector.
	relation := C.mitie_extract_binary_relation(ner, tokens, idx1pos, idx1len, idx2pos, idx2len)
	if relation == nil {
		return nil, ErrMemory
	}
	defer C.mitie_free(unsafe.Pointer(relation))

	for _, detector := range detectors {
		var score C.double

		// Calling this function runs the relation detector on the relation and stores the
		// output into score.  If score is > 0 then the detector is indicating that this
		// relation mention is an example of the type of relation this detector is looking for.
		// Moreover, the larger score the more confident the detector is that it is that this
		// is a correct relation detection.
		if C.mitie_classify_binary_relation(detector.brd, relation, &score) != 0 {
			// When you train a relation detector it uses features derived from a MITIE NER
			// object as part of its processing.  This is also evident in the interface of
			// mitie_extract_binary_relation() which requires a NER object to perform feature
			// extraction.  Because of this, every relation detector depends on a NER object
			// and, moreover, it is important that you use the same NER object which was used
			// during training when you run the relation detector.  If you don't use the same
			// NER object instance the mitie_classify_binary_relation() routine will return an
			// error.
			return nil, ErrIncompatibleNER
		}
		if float64(score) > 0 {
			rels = append(rels, Relation{
				Relationship: detector.String(),
				From:         Range{Start: int(idx1pos), End: int(idx1len) + int(idx1pos)},
				To:           Range{Start: int(idx2pos), End: int(idx2len) + int(idx2pos)},
				Score:        float64(score),
			})
		}
	}
	return rels, nil
}
