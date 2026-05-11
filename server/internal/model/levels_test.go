package model

import "testing"

func TestLevelOrd_Tennis(t *testing.T) {
	ord := LevelOrd(SportTennis, "3.5")
	if ord == nil {
		t.Fatal("LevelOrd returned nil for tennis 3.5")
	}
	if *ord != 35 {
		t.Fatalf("LevelOrd(SportTennis, 3.5) = %d, want 35", *ord)
	}
}

func TestLevelOrd_TennisFineGrainedRange(t *testing.T) {
	min := LevelOrd(SportTennis, "1.0")
	max := LevelOrd(SportTennis, "7.0")
	if min == nil || *min != 10 {
		t.Fatalf("LevelOrd(SportTennis, 1.0) = %v, want 10", min)
	}
	if max == nil || *max != 70 {
		t.Fatalf("LevelOrd(SportTennis, 7.0) = %v, want 70", max)
	}

	granular := LevelOrd(SportTennis, "3.4")
	if granular == nil || *granular != 34 {
		t.Fatalf("LevelOrd(SportTennis, 3.4) = %v, want 34", granular)
	}

	invalid := LevelOrd(SportTennis, "7.1")
	if invalid != nil {
		t.Fatalf("LevelOrd(SportTennis, 7.1) = %v, want nil", invalid)
	}
}

func TestLevelDefs_Tennis(t *testing.T) {
	defs := LevelDefs(SportTennis)
	if len(defs) == 0 {
		t.Fatal("LevelDefs(SportTennis) returned no levels")
	}
	if defs[0].Code != "1.0" {
		t.Fatalf("first tennis level = %q, want 1.0", defs[0].Code)
	}
	if defs[len(defs)-1].Code != "7.0" {
		t.Fatalf("last tennis level = %q, want 7.0", defs[len(defs)-1].Code)
	}
}
