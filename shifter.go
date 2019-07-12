package goshift

type Shifter interface {
	Apply(src map[string]interface{}, opts ...ShiftOption) (map[string]interface{}, error)
}

type ShiftOption func(o *shiftOp)

type ShiftReporter func(src, dst string, value interface{}) interface{}

type shiftOp struct {
	errCatcher func(err error) bool
	reporter   ShiftReporter
	mutable    bool
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

func Mutable() ShiftOption {
	return func(o *shiftOp) {
		o.mutable = true
	}
}
