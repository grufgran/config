package stringMask

// set tag
func (cm *StringMask) NewTagAtPos(pos int, val string) {
	cm.tags[pos] = val
}
