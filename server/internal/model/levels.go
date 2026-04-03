package model

// Level ordinal mappings for each sport.
// Ordinals define the skill ordering for filtering and sorting.
// Gaps of 10 allow inserting new levels between existing ones.
//
// Usage:
//   ord := LevelOrd(SportBadminton, "HB") // returns &30
//   code := LevelCode(SportBadminton, 30) // returns &"HB"
//
// To query "plays where a HB player can join":
//   WHERE level_min_ord <= 30 AND (level_max_ord >= 30 OR level_max_ord IS NULL)

// BadmintonLevels defines the badminton skill levels in Singapore.
//
//	LB (10) - Low Beginner
//	MB (20) - Mid Beginner
//	HB (30) - High Beginner
//	LI (40) - Low Intermediate
//	MI (50) - Mid Intermediate
//	HI (60) - High Intermediate
//	A  (70) - Advanced
var BadmintonLevels = []LevelDef{
	{Code: "LB", Ord: 10, Name: "Low Beginner"},
	{Code: "MB", Ord: 20, Name: "Mid Beginner"},
	{Code: "HB", Ord: 30, Name: "High Beginner"},
	{Code: "LI", Ord: 40, Name: "Low Intermediate"},
	{Code: "MI", Ord: 50, Name: "Mid Intermediate"},
	{Code: "HI", Ord: 60, Name: "High Intermediate"},
	{Code: "A", Ord: 70, Name: "Advanced"},
}

// LevelDef defines a single level within a sport's skill scale.
type LevelDef struct {
	Code string // short code: "HB", "3.5"
	Ord  int    // numeric ordinal for sorting/filtering
	Name string // human-readable name
}

// sportLevelIndex is a pre-built lookup for fast code->ord and ord->code.
var sportLevelIndex map[Sport]*levelIndex

type levelIndex struct {
	byCode map[string]LevelDef
	byOrd  map[int]LevelDef
}

func init() {
	sportLevelIndex = map[Sport]*levelIndex{
		SportBadminton: buildLevelIndex(BadmintonLevels),
		// Tennis: use NTRP rating directly as the ordinal (e.g. 35 for 3.5)
		// Football: TBD
	}
}

func buildLevelIndex(defs []LevelDef) *levelIndex {
	idx := &levelIndex{
		byCode: make(map[string]LevelDef, len(defs)),
		byOrd:  make(map[int]LevelDef, len(defs)),
	}
	for _, d := range defs {
		idx.byCode[d.Code] = d
		idx.byOrd[d.Ord] = d
	}
	return idx
}

// LevelOrd returns the numeric ordinal for a level code within a sport.
// Returns nil if the sport or code is not recognized.
func LevelOrd(sport Sport, code string) *int {
	idx, ok := sportLevelIndex[sport]
	if !ok {
		return nil
	}
	if def, ok := idx.byCode[code]; ok {
		return &def.Ord
	}
	return nil
}

// LevelCode returns the level code for a numeric ordinal within a sport.
// Returns nil if the sport or ordinal is not recognized.
func LevelCode(sport Sport, ord int) *string {
	idx, ok := sportLevelIndex[sport]
	if !ok {
		return nil
	}
	if def, ok := idx.byOrd[ord]; ok {
		return &def.Code
	}
	return nil
}

// LevelName returns the human-readable name for a level code within a sport.
// Returns nil if not recognized.
func LevelName(sport Sport, code string) *string {
	idx, ok := sportLevelIndex[sport]
	if !ok {
		return nil
	}
	if def, ok := idx.byCode[code]; ok {
		return &def.Name
	}
	return nil
}

// LevelDefs returns all level definitions for a sport, ordered by ordinal.
func LevelDefs(sport Sport) []LevelDef {
	switch sport {
	case SportBadminton:
		return BadmintonLevels
	default:
		return nil
	}
}
