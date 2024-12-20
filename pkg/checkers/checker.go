package checkers

type Checker interface {
	Check(address string) (success bool, responseTime int64, err error)
}
