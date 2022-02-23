package tests

// TestParams is Implemented by all parameter test classes.
//
// Name is used to dispatch to the relevant test
type TestParams interface {
	Name() string
}
