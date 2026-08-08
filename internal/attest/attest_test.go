package attest_test

import (
	"testing"

	"github.com/afelin/cyberready/internal/attest"
)

func TestReproducibleStateHash(t *testing.T) {
	a := attest.ComputeStateHash("abc", "parent", "sbom1", "vex1")
	b := attest.ComputeStateHash("abc", "parent", "sbom1", "vex1")
	if a != b {
		t.Fatal("state hash must be reproducible")
	}
	c := attest.ComputeStateHash("abc", "parent", "sbom2", "vex1")
	if a == c {
		t.Fatal("sbom digest must affect hash")
	}
	seed := attest.StateSeed("abc", "parent", "sbom1", "vex1")
	if seed != "abc|parent|sbom=sbom1|vex=vex1" {
		t.Fatalf("seed=%q", seed)
	}
}
