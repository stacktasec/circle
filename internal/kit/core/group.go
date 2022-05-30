package core

type versionGroup struct {
	mainVersion        int
	stableConstructors []any
	betaConstructors   []any
	alphaConstructors  []any
}

func NewGroup(mainVersion int) *versionGroup {
	if mainVersion < 1 {
		panic("main version must larger than one")
	}

	return &versionGroup{
		mainVersion: mainVersion,
	}
}

func (v *versionGroup) SetStable(services ...any) {
	v.stableConstructors = append(v.stableConstructors, services...)
}

func (v *versionGroup) SetBeta(services ...any) {
	v.betaConstructors = append(v.betaConstructors, services...)
}

func (v *versionGroup) SetAlpha(services ...any) {
	v.alphaConstructors = append(v.alphaConstructors, services...)
}
