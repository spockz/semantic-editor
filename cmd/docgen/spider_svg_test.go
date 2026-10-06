// spider_svg_test.go protects the visible scale and accessible labels of task radar charts.
package main

import (
	"strings"
	"testing"
)

func TestRenderRadarSVGShowsAllSevenComplexityAxes(t *testing.T) {
	profile := ComplexityProfile{Locality: 1, Context: 2, Ambiguity: 3, Composition: 4, Invariants: 5, Scale: 2, Staging: 4}
	svg, err := RenderRadarSVG(profile, 620, 460)
	if err != nil {
		t.Fatal(err)
	}
	for _, label := range []string{"Locality", "Context", "Ambiguity", "Composition", "Invariants", "Scale", "Staging"} {
		if !strings.Contains(svg, ">"+label+"</text>") {
			t.Errorf("radar chart is missing %s axis label", label)
		}
	}
	if got := strings.Count(svg, `<polygon points=`); got != 6 {
		t.Fatalf("radar chart polygon count = %d, want five scale rings and one task profile", got)
	}
}

func TestRenderRadarSVGRejectsInvalidProfilesAndDimensions(t *testing.T) {
	valid := ComplexityProfile{Locality: 1, Context: 1, Ambiguity: 1, Composition: 1, Invariants: 1, Scale: 1, Staging: 1}
	if _, err := RenderRadarSVG(valid, 279, 460); err == nil {
		t.Fatal("radar chart accepted dimensions too small for axis labels")
	}
	valid.Staging = 6
	if _, err := RenderRadarSVG(valid, 620, 460); err == nil {
		t.Fatal("radar chart accepted a complexity value above five")
	}
}
