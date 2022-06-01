package internal

type Request interface {
	Validate() error
}

type VersionGroup struct {
	MainVersion        int
	StableConstructors []any
	BetaConstructors   []any
	AlphaConstructors  []any
}

func (v *VersionGroup) SetStable(services ...any) {
	v.StableConstructors = append(v.StableConstructors, services...)
}

func (v *VersionGroup) SetBeta(services ...any) {
	v.BetaConstructors = append(v.BetaConstructors, services...)
}

func (v *VersionGroup) SetAlpha(services ...any) {
	v.AlphaConstructors = append(v.AlphaConstructors, services...)
}
