package context

type paramType int8
type saveDest int8

const (
	CurrSect paramType = iota
	CurrMacro

	Macros saveDest = iota
	Sects
)

type runTimeValues struct {
	Macros map[string]*macro
	Params map[paramType]string
	SaveTo saveDest
}

func (r *runTimeValues) SetCurrentSect(cs string) {
	r.Params[CurrSect] = cs
	r.SaveTo = Sects
}

func (r *runTimeValues) SetCurrentMacro(m string) {
	r.Params[CurrMacro] = m
	r.SaveTo = Macros
}

func (r *runTimeValues) GetCurrentMacro() *macro {
	cm := r.Params[CurrMacro]
	m := r.Macros[cm]
	return m
}
