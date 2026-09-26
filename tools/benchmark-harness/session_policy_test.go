// This file verifies the public arm policy values and their strict parsing contract.
package main

import "testing"

func TestParseSemeditArmRestrictionIsStrict(t *testing.T) {
	for _, value := range []SemeditArmRestriction{SemeditArmRestrictRead, SemeditArmRestrictWrite, SemeditArmRestrictReadWrite} {
		parsed, err := ParseSemeditArmRestriction(string(value))
		if err != nil || parsed != value {
			t.Fatalf("ParseSemeditArmRestriction(%q) = %q, %v", value, parsed, err)
		}
	}
	for _, value := range []string{"", "READ", "none", "write "} {
		if _, err := ParseSemeditArmRestriction(value); err == nil {
			t.Fatalf("ParseSemeditArmRestriction(%q) unexpectedly succeeded", value)
		}
	}
}

func TestNewRunnerDefaultsArmRestrictionToWrite(t *testing.T) {
	runner := NewRunner(t.TempDir())
	if runner.semeditArmRestriction != SemeditArmRestrictWrite {
		t.Fatalf("default restriction = %q, want write", runner.semeditArmRestriction)
	}
}
