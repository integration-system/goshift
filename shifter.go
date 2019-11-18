package goshift

type Shifter interface {
	Apply(src map[string]interface{}, opts ...ShiftOption) (map[string]interface{}, error)
}

type ShiftOption func(o *shiftOp)

type ShiftReporter func(src, dst string, value interface{}) interface{}

type shiftOp struct {
	errCatcher func(err error) bool
	reporter   ShiftReporter
	base       map[string]interface{}
}

func WithErrorCatching(f func(err error) bool) ShiftOption {
	return func(op *shiftOp) {
		op.errCatcher = f
	}
}

func WithReporter(reporter ShiftReporter) ShiftOption {
	return func(o *shiftOp) {
		o.reporter = reporter
	}
}

func WithBase(base map[string]interface{}) ShiftOption {
	return func(o *shiftOp) {
		o.base = base
	}
}
