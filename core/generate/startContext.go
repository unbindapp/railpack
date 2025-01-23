package generate

type StartContext struct {
	BaseImage string
	Command   string
	Paths     []string
}

func NewStartContext() *StartContext {
	return &StartContext{}
}
