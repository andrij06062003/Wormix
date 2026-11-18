package app

import (
	"github.com/andrij06062003/Wormix/services/api/internal/db"
	"github.com/andrij06062003/Wormix/services/api/internal/php"
)

// The 1.61.7 client's complex weapons are owned for good at a level (db.ComplexMaxLevel).

// complexLevelBase: a count of complexLevelBase-L is level L of a complex weapon.
const complexLevelBase = -10

// complexLevel is the level a count keeps of a complex weapon (0 none).
func complexLevel(id, count int64) int64 {
	top := db.ComplexMaxLevel[id]
	switch {
	case count == -1:
		return top
	case count < complexLevelBase && count >= complexLevelBase-top:
		return complexLevelBase - count
	case count > 0:
		// plain shots (or a count kept before levels): level 1 at least, as many as it has
		return min(count, top)
	}
	return 0
}

// complexCount is the count keeping level of a complex weapon.
func complexCount(id, level int64) int64 {
	if level >= db.ComplexMaxLevel[id] {
		return -1
	}
	return complexLevelBase - max(level, 1)
}

// isComplex tells a complex weapon by its id.
func isComplex(id any) bool {
	_, ok := db.ComplexMaxLevel[php.ToInt(id)]
	return ok
}

// addComplexLevels gives levels of a complex weapon on top of the count kept (nil: none yet).
func addComplexLevels(id int64, kept any, levels int64) int64 {
	var have int64
	if kept != nil {
		have = complexLevel(id, php.ToInt(kept))
	}
	return complexCount(id, have+levels)
}
