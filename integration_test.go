// +build integration

// Move this out of the main tests to not require the CI to download the 400MB
// model pack.
package ner_test

import (
	"testing"

	"github.com/sbl/ner"
)

const txt = `A Pegasus Airlines plane landed at an Istanbul airport Friday
after a passenger "said that there was a bomb on board" and wanted the plane
to land in Sochi, Russia, the site of the Winter Olympics, said officials with
Turkey's Transportation Ministry.

Meredith Vieira will become the first woman to host Olympics primetime
coverage on her own when she fills on Friday night for the ailing Bob Costas,
who is battling a continuing eye infection.  "It's an honor to fill in for
him," Vieira said on TODAY Friday. "You think about the Olympics, and you
think the athletes and then Bob Costas." "Bob's eye issue has improved but
he's not quite ready to do the show," NBC Olympics Executive Producer Jim Bell
told TODAY.com from Sochi on Thursday.

From wikipedia we learn that Josiah Franklin's son, Benjamin Franklin was born
in Boston.  Since wikipedia allows anyone to edit it, you could change the
entry to say that Philadelphia is the birthplace of Benjamin Franklin.
However, that would be a bad edit since Benjamin Franklin was definitely born
in Boston.`

func TestSmokeTest(t *testing.T) {
	ext, err := ner.NewExtractor("/usr/local/share/MITIE-models/english/ner_model.dat")
	if err != nil {
		t.Fatal(err)
	}
	defer ext.Free()

	if want, have := 4, len(ext.Tags()); want != have {
		t.Errorf("want: %+v tags, have: %+v", want, have)
	}

	tokens := ner.Tokenize(txt)

	es, err := ext.Extract(tokens)
	if err != nil {
		t.Fatal(err)
	}

	if got := es[0].Name; got != "Pegasus Airlines" {
		t.Errorf("unexpected token %s", got)
	}
}
