package stringMask

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

type StringMask struct {
	String     []rune
	mask       []rune
	tags       map[int]string
	Comment    MaskPoint
	InitMarker rune
}

// init cmask. Default CommentMask is ⛔
func NewStringMask(indataString string, initMarker rune, commentOptions ...rune) *StringMask {
	sm := StringMask{
		String:     []rune(indataString),
		mask:       make([]rune, utf8.RuneCountInString(indataString)),
		tags:       make(map[int]string),
		InitMarker: initMarker,
		Comment: MaskPoint{
			Mask: '⛔',
			Pos:  -1,
		},
	}

	// set default commentMarker
	commentMarker := rune(0)
	commentMarkerDefined := false
	// check if there was a commentMarker provided
	if len(commentOptions) > 0 {
		commentMarker = commentOptions[0]
		commentMarkerDefined = true
	}

	// check if there was a custom commentMask provided
	if len(commentOptions) > 1 {
		sm.Comment.Mask = commentOptions[1]
	}

	// Loop thru data string and set initMarkers and endMarker
	for i, c := range indataString {
		if commentMarkerDefined && c == commentMarker && sm.Comment.Pos == -1 {
			sm.mask[i] = sm.Comment.Mask
			sm.Comment.Pos = i
		} else {
			sm.mask[i] = initMarker
		}
	}
	return &sm
}

// Get mask endPoint. Its either the rune before the commentMarker (if present) or the last rune in string
func (sm *StringMask) GetEndPoint() *MaskPoint {
	if sm.Comment.Pos > 0 {
		endPoint := sm.NewMaskPoint(sm.Comment.Pos - 1)
		return &endPoint
	} else if len(sm.String) > 0 {
		endPoint := sm.NewMaskPoint(len(sm.String) - 1)
		return &endPoint
	}
	return nil
}

// Check if whereMasksAre contains runeToCheck
func runeMatches(runeToCheck rune, whereMasksAre *[]rune) bool {
	// If no whereMasksAre is defined, we return true
	if len(*whereMasksAre) == 0 {
		return true
	}
	for i := range *whereMasksAre {
		if (*whereMasksAre)[i] == runeToCheck {
			return true
		}
	}
	return false
}

// Returns a slice which forces funtions here to check against unicode.IsSpace
func GetSpaces() []rune {
	return []rune{' ', '\t', '\v'}
}

func isSpaces(runesToMatch []rune) bool {
	if len(runesToMatch) == 3 {
		if runesToMatch[0] == ' ' && runesToMatch[1] == '\t' && runesToMatch[2] == '\v' {
			return true
		}
	}
	return false
}

// string!
func (sm *StringMask) GetString(whereMaskIs rune, useTagsWhereMaskIs ...rune) string {
	endPoint := sm.GetEndPoint()
	sb := strings.Builder{}
	for i, c := range sm.String {
		if len(useTagsWhereMaskIs) > 0 && sm.mask[i] == useTagsWhereMaskIs[0] {
			if val, exists := sm.tags[i]; exists {
				sb.WriteString(val)
			}
		}
		if sm.mask[i] == whereMaskIs {
			sb.WriteRune(c)
		}
		if i >= endPoint.Pos {
			break
		}
	}
	return sb.String()
}

// string!
func (sm *StringMask) GetStrings(whereMaskIs rune, useTagsWhereMaskIs ...rune) []string {
	endPoint := sm.GetEndPoint()
	sb := strings.Builder{}
	delimiterWritten := true
	for i, c := range sm.String {
		if len(useTagsWhereMaskIs) > 0 && sm.mask[i] == useTagsWhereMaskIs[0] {
			if val, exists := sm.tags[i]; exists {
				sb.WriteString(val)
			}
		}
		if sm.mask[i] == whereMaskIs {
			sb.WriteRune(c)
			delimiterWritten = false
		} else if !delimiterWritten {
			sb.WriteRune(rune(0))
			delimiterWritten = true
		}
		if i >= endPoint.Pos {
			break
		}
	}
	strs := strings.Split(sb.String(), string(rune(0)))
	// if delimiterWritten == true we have one to many strings. Lets delete the last
	if delimiterWritten {
		if len(strs) > 1 {
			strs = strs[:len(strs)-1]
		} else {
			// strange, strings where found. Create a new empty slice
			return make([]string, 0)
		}
	}
	return strs

}

func (sm *StringMask) GetStringBetween(fromPos, toPos int, trimSpace bool) string {
	// find new fromPos and toPos if trimSpace = true
	if trimSpace {
		// find new toPos
		for i := toPos; i >= fromPos; i-- {
			if unicode.IsSpace(sm.String[i]) {
				toPos = i - 1
			} else {
				break
			}
		}
		// if there were only spaces, no need to go further, just return empty string
		if toPos < fromPos {
			return ""
		}
		// find from toPos
		for i := fromPos; i <= toPos; i++ {
			if unicode.IsSpace(sm.String[i]) {
				fromPos = i + 1
			} else {
				break
			}
		}
	}

	// start looping
	var sb strings.Builder
	sb.Grow(toPos - fromPos + 1)
	for i := fromPos; i <= toPos; i++ {
		sb.WriteRune(sm.String[i])
	}
	return sb.String()
}
