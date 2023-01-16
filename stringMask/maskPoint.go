package stringMask

import (
	"fmt"
	"strconv"
	"unicode"
)

type MaskPoint struct {
	Rune rune
	Mask rune
	Pos  int
}

// init MaskPoint
func (cm *StringMask) NewMaskPoint(pos int) MaskPoint {
	if pos < 0 {
		return MaskPoint{
			Rune: rune(0),
			Mask: rune(0),
			Pos:  -1,
		}
	} else {
		return MaskPoint{
			Rune: cm.String[pos],
			Mask: cm.mask[pos],
			Pos:  pos,
		}
	}
}

// return num MaskPoints looping in direktion and wherePointsAre. Any -1 value in wherePointsAre means all
func (sm *StringMask) GetMaskPoints(fromPos, direction, maxHits, maxFlops int, runesToMatch []rune, whereMasksAre ...rune) *[]MaskPoint {
	// init and error check
	points := make([]MaskPoint, 0, 10)

	// Get endPoint
	endPoint := sm.GetEndPoint()
	if endPoint == nil {
		// No point to continue, since the string is empty
		return &points
	}

	// if fromPos is -1 we start from end
	if fromPos == -1 {
		fromPos = endPoint.Pos
	}

	// if runesToMatch contains ' ', '\t', '\v' then we will check agains unicode.IsWhiteSpace
	isSpaceSearch := isSpaces(runesToMatch)

	flopNum := 0
	// Loop
	for i := fromPos; i <= endPoint.Pos && i >= 0; i += direction {
		spaceMatch := (isSpaceSearch && unicode.IsSpace(sm.String[i])) || !isSpaceSearch
		runeMatch := runeMatches(sm.String[i], &runesToMatch)
		maskMatch := runeMatches(sm.mask[i], &whereMasksAre)
		if spaceMatch && runeMatch && maskMatch {

			// Create new Maskpoint and append to poses
			mp := sm.NewMaskPoint(i)
			points = append(points, mp)

			// Break if we found enough runes
			if maxHits > 0 && len(points) >= maxHits {
				break
			}
		} else if maxFlops > 0 {
			flopNum++
			if flopNum >= maxFlops {
				break
			}
		}
	}
	return &points
}

// return MaskPoint nr #
func getPosN(sm *StringMask, source *[]rune, nr int, whereIs rune, otherSource *[]rune, whereOtherIs ...rune) *MaskPoint {
	pos := -1
	numHits := 0
	otherLen := len(whereOtherIs)
	lp := sm.GetEndPoint()
	if lp == nil {
		return nil
	}

	for i, m := range *source {
		if m == whereIs {
			pos = i
			if (otherLen > 0 && (*otherSource)[pos] == whereOtherIs[0]) || otherLen == 0 {
				numHits++
				if numHits == nr {
					break
				}
			}
		}
		if i == lp.Pos {
			break
		}
	}
	if pos == -1 {
		return nil
	}
	mp := sm.NewMaskPoint(pos)
	return &mp
}

func (sm *StringMask) GetRunePointN(nr int, whereRuneIs rune, whereMaskIs ...rune) *MaskPoint {
	return getPosN(sm, &sm.String, nr, whereRuneIs, &sm.mask, whereMaskIs...)
}

func (sm *StringMask) GetMaskPointN(nr int, whereMaskIs rune, whereRuneIs ...rune) *MaskPoint {
	return getPosN(sm, &sm.mask, nr, whereMaskIs, &sm.String, whereRuneIs...)
}

// return first MaskPoint whereAnyMaskIs
func (sm *StringMask) GetFirstRunePoint(whereRuneIs rune, whereMaskIs ...rune) *MaskPoint {
	return sm.GetRunePointN(1, whereRuneIs, whereMaskIs...)
}

// return last MaskPoint whereMaskIs
func (sm *StringMask) GetLastRunePoint(whereRuneIs rune, whereMaskIs ...rune) *MaskPoint {
	return sm.GetRunePointN(-1, whereRuneIs, whereMaskIs...)
}

// return first MaskPoint whereAnyMaskIs
func (sm *StringMask) GetFirstMaskPoint(whereMaskIs rune, whereRuneIs ...rune) *MaskPoint {
	return sm.GetMaskPointN(1, whereMaskIs, whereRuneIs...)
}

// return last MaskPoint whereMaskIs
func (sm *StringMask) GetLastMaskPoint(whereMaskIs rune, whereRuneIs ...rune) *MaskPoint {
	return sm.GetMaskPointN(-1, whereMaskIs, whereRuneIs...)
}

// return all
func (cm *StringMask) GetAllMaskPoints(runeToMatch rune, whereMaskIs rune) *[]MaskPoint {
	return cm.GetMaskPoints(0, 1, -1, -1, []rune{runeToMatch}, whereMaskIs)
}

// Keep track of level.
// [ = level 1
// [[ = level 2
// ...
func (cm *StringMask) GetMaskPointsForOppositeRunes(leftRune rune, rightRune rune, whereMasksAre ...rune) (*[]MaskPoint, *[]MaskPoint, error) {
	// get EndPoint
	endPoint := cm.GetEndPoint()

	// Loop thru data string
	leftPoints := make([]MaskPoint, 0, 5)
	rightPoints := make([]MaskPoint, 0, 5)
	level := 0

	var err error
	for i, c := range cm.String {
		if c == leftRune && runeMatches(cm.mask[i], &whereMasksAre) {
			leftPoints = append(leftPoints, cm.NewMaskPoint(i))
			rightPoints = append(rightPoints, cm.NewMaskPoint(-1))
			level++
		}
		if c == rightRune && runeMatches(cm.mask[i], &whereMasksAre) {
			num := len(leftPoints)

			// check so we don't have more rightPoses than leftPoses
			if num-level >= len(rightPoints) {
				// fatal, we have a rightRune before a corresponding leftRune
				leftPoints = append(leftPoints, cm.NewMaskPoint(-i))
				rightPoints = append(rightPoints, cm.NewMaskPoint(i))
				err = fmt.Errorf("missing %s in \"%s\"", strconv.QuoteRune(leftRune), string(cm.String))
			} else {
				// Add postition in squareBracketsEnd slice
				rightPoints[num-level].Pos = i
				level--
			}
		}
		if i == endPoint.Pos {
			break
		}
	}
	// check to see that we have found all rightRunes also
	for i := range rightPoints {
		if rightPoints[i].Pos == -1 {
			err = fmt.Errorf("missing %s in \"%s\"", strconv.QuoteRune(rightRune), string(cm.String))
			break
		}
	}
	return &leftPoints, &rightPoints, err
}

// Remove from MaskPoint from slice
func RemoveMaskPointFromSlice(mps *[]MaskPoint, i int) *[]MaskPoint {
	if len(*mps) > 1 {
		(*mps)[i] = (*mps)[len(*mps)-1]
		reducedMaskPointSlice := (*mps)[:len(*mps)-1]
		return &reducedMaskPointSlice
	} else if len(*mps) == 1 {
		emptyMaskPointSlice := make([]MaskPoint, 0, cap(*mps))
		return &emptyMaskPointSlice
	}
	return mps
}
