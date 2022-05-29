package core

type versionGroup struct {
	mainVersion    int
	stableServices []any
	betaServices   []any
	alphaServices  []any
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
	v.stableServices = append(v.stableServices, services...)
}

func (v *versionGroup) SetBeta(services ...any) {
	v.betaServices = append(v.betaServices, services...)
}

func (v *versionGroup) SetAlpha(services ...any) {
	v.alphaServices = append(v.alphaServices, services...)
}
