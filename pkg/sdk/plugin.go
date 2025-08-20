package sdk

type Plugin interface {
	Init(ctx Context) error
	Stop() error
}
