package core

type versionGroup struct {
	mainVersion    int
	stableServices []Service
	betaServices   []Service
	alphaServices  []Service
}

func NewGroup(mainVersion int) *versionGroup {
	if mainVersion < 1 {
		panic("main version must larger than one")
	}

	return &versionGroup{
		mainVersion: mainVersion,
	}
}

func (v *versionGroup) SetStable(services ...Service) {
	v.stableServices = append(v.stableServices, services...)
}

func (v *versionGroup) SetBeta(services ...Service) {
	v.betaServices = append(v.betaServices, services...)
}

func (v *versionGroup) SetAlpha(services ...Service) {
	v.alphaServices = append(v.alphaServices, services...)
}
