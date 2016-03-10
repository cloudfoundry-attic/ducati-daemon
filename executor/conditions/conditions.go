package conditions

//go:generate counterfeiter --fake-name Context . Context
type Context interface{}

//go:generate counterfeiter --fake-name Condition . Condition
type Condition interface {
	Satisfied(context Context) bool
	String() string
}
