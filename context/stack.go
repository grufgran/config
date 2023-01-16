package context

type stack []string

// IsEmpty: check if stack is empty
func (s *stack) IsEmpty() bool {
	return len(*s) == 0
}

// Push a new ronwInfo onto the stack
func (s *stack) Push(fileName string) {
	*s = append(*s, fileName)
}

// Remove and return top element of stack. Return false if stack is empty.
func (s *stack) Pop() (string, bool) {
	if s.IsEmpty() {
		return "", false
	} else {
		index := len(*s) - 1   // Get the index of the top most element.
		element := (*s)[index] // Index into the slice and obtain the element.
		*s = (*s)[:index]      // Remove it from the stack by slicing it off.
		return element, true
	}
}

// check if filename already exist in stack
func (s *stack) Contains(fileName *string) bool {

	// loop thru stack and check if fileName exists
	for i := 0; i < len(*s); i++ {
		if (*s)[i] == *fileName {
			return true
		}
	}
	return false
}
