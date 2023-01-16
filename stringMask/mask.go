package stringMask

// Mask from given pos in given direction. This is the general function, used by most of the functions below. runesToMatch can be nil, meaning any rune
func (sm *StringMask) MaskFromPos(fromPos, direction, maxHits, maxFlops int, runesToMatch []rune, setMaskTo rune, whereMasksAre ...rune) *[]MaskPoint {
	points := sm.GetMaskPoints(fromPos, direction, maxHits, maxFlops, runesToMatch, whereMasksAre...)
	for _, p := range *points {
		sm.mask[p.Pos] = setMaskTo
	}
	return points
}

func (sm *StringMask) MaskSpaces(fromPos, direction int, setMaskTo, whereMaskIs rune) []MaskPoint {
	// create space runes
	runesToMatch := GetSpaces()
	mp := sm.NewMaskPoint(fromPos)
	if mp.Pos == -1 {
		return make([]MaskPoint, 0)
	}
	pos := mp.Pos
	points := sm.MaskFromPos(pos, direction, -1, 1, runesToMatch, setMaskTo, whereMaskIs)
	return *points
}

// mask leading and trailing white space where mask is
func (sm *StringMask) MaskStartEndSpaces(setMaskTo, whereMaskIs rune) []MaskPoint {
	startPoses := sm.MaskStartSpaces(setMaskTo, whereMaskIs)
	endPoses := sm.MaskEndSpaces(setMaskTo, whereMaskIs)
	poses := append(startPoses, endPoses...)
	return poses
}

// mask leading white space where mask is
func (sm *StringMask) MaskStartSpaces(setMaskTo, whereMaskIs rune) []MaskPoint {
	if mp := sm.NewMaskPoint(0); mp.Mask == whereMaskIs {
		return sm.MaskSpaces(0, 1, setMaskTo, whereMaskIs)
	}
	return make([]MaskPoint, 0)
}

// mask trailing white space where mask is
func (sm *StringMask) MaskEndSpaces(setMaskTo, whereMaskIs rune) []MaskPoint {
	// Find last pos whereMasksAre
	if mp := sm.GetEndPoint(); mp != nil && mp.Mask == whereMaskIs {
		return sm.MaskSpaces(mp.Pos, -1, setMaskTo, whereMaskIs)
	}
	return make([]MaskPoint, 0)
}

// mask leading and trailing white space around mask
func (sm *StringMask) MaskLeftRightSpacesAround(pos int, setMaskTo, whereMaskIs rune) []MaskPoint {
	leftPoses := sm.MaskLeftSpacesFromPos(pos-1, setMaskTo, whereMaskIs)
	rightPoses := sm.MaskRightSpacesFromPos(pos+1, setMaskTo, whereMaskIs)
	return append(leftPoses, rightPoses...)
}

func (sm *StringMask) MaskLeftRightSpacesAroundPoints(points *[]MaskPoint, setMaskTo, whereMaskIs rune) []MaskPoint {
	spacePoints := make([]MaskPoint, 0)
	for _, mp := range *points {
		spacePoints = append(spacePoints, sm.MaskLeftRightSpacesAround(mp.Pos, setMaskTo, whereMaskIs)...)
	}
	return spacePoints
}

// mask leading white space around mask
func (sm *StringMask) MaskLeftSpacesFromPos(pos int, setMaskTo, whereMaskIs rune) []MaskPoint {
	if mp := sm.NewMaskPoint(pos); mp.Pos > -1 && mp.Mask == whereMaskIs {
		return sm.MaskSpaces(mp.Pos, -1, setMaskTo, whereMaskIs)
	}
	return make([]MaskPoint, 0)
}

// mask trailing white space around mask
func (sm *StringMask) MaskRightSpacesFromPos(pos int, setMaskTo, whereMaskIs rune) []MaskPoint {
	if mp := sm.NewMaskPoint(pos); mp.Pos > -1 && mp.Mask == whereMaskIs {
		return sm.MaskSpaces(mp.Pos, 1, setMaskTo, whereMaskIs)
	}
	return make([]MaskPoint, 0)
}

// mask with provided runeMask at pos. This overwrites preveous mask
func (sm *StringMask) MaskAtPos(pos int, setMaskTo rune) {
	sm.mask[pos] = setMaskTo
}

// mask with provided runeMask. This overwrites previous mask
func (sm *StringMask) MaskBetween(fromPos, toPos int, setMaskTo rune) {
	for i := fromPos; i <= toPos; i++ {
		sm.mask[i] = setMaskTo
	}
}

// mask all runeToMatch with setMaskTo, whereMaskIs
func (sm *StringMask) Mask(runeToMatch, setMaskTo, whereMaskIs rune) *[]MaskPoint {
	return sm.MaskFromPos(0, 1, -1, -1, []rune{runeToMatch}, setMaskTo, whereMaskIs)
}

// mask first runeToMatch with setMaskTo, whereMaskIs
func (sm *StringMask) MaskFirst(runeToMatch, setMaskTo, whereMaskIs rune) *MaskPoint {
	points := sm.MaskFromPos(0, 1, 1, -1, []rune{runeToMatch}, setMaskTo, whereMaskIs)
	if len(*points) > 0 {
		return &(*points)[0]
	}
	return nil
}

// mask last runeToMatch with setMaskTo, whereMaskIs
func (sm *StringMask) MaskLast(runeToMatch, setMaskTo, whereMaskIs rune) *MaskPoint {
	points := sm.MaskFromPos(-1, -1, 1, -1, []rune{runeToMatch}, setMaskTo, whereMaskIs)
	if len(*points) > 0 {
		return &(*points)[0]
	}
	return nil
}

// mask first word whereMaskIs
func (sm *StringMask) MaskFirstWord(wordToFind string, setMaskTo rune, whereMaskIs ...rune) *MaskPoint {
	if pos := sm.search(&wordToFind, whereMaskIs...); pos == -1 {
		return nil
	} else {
		sm.MaskBetween(pos, pos+len(wordToFind)-1, setMaskTo)
		point := sm.NewMaskPoint(pos)
		return &point
	}
}

func (sm *StringMask) search(word *string, whereMaskIs ...rune) int {
	runeWord := []rune(*word)
	for i := range sm.String {
		found := true
		for j := range runeWord {
			if sm.String[i+j] != runeWord[j] {
				found = false
				break
			} else if len(whereMaskIs) > 0 && sm.mask[i+j] != whereMaskIs[0] {
				found = false
				break
			}
		}
		if found {
			return i
		}
	}
	return -1
}
