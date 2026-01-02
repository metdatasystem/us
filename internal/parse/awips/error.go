package internal

import (
	"github.com/rs/zerolog"
)

type ErrorCollector struct {
	total  int
	errors map[zerolog.Level][]string
}

func NewErrorCollector() *ErrorCollector {
	errors := map[zerolog.Level][]string{
		zerolog.WarnLevel:  []string{},
		zerolog.ErrorLevel: []string{},
		zerolog.FatalLevel: []string{},
	}

	return &ErrorCollector{
		total:  0,
		errors: errors,
	}
}

func (ec *ErrorCollector) Warn(msg string) {
	slice, ok := ec.errors[zerolog.WarnLevel]
	if !ok {
		ec.errors[zerolog.WarnLevel] = []string{msg}
	} else {
		slice = append(slice, msg)
	}
	ec.total++
}

func (ec *ErrorCollector) Error(msg string) {
	slice, ok := ec.errors[zerolog.ErrorLevel]
	if !ok {
		ec.errors[zerolog.ErrorLevel] = []string{msg}
	} else {
		slice = append(slice, msg)
	}
	ec.total++
}

func (ec *ErrorCollector) Fatal(msg string) {
	slice, ok := ec.errors[zerolog.FatalLevel]
	if !ok {
		ec.errors[zerolog.FatalLevel] = []string{msg}
	} else {
		slice = append(slice, msg)
	}
	ec.total++
}
