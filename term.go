package goshift

type term struct {
	name    string
	isArray bool
	mapping []string

	pm []interface{} //a b 1 c
}

func newTerm(name string, isArray bool, mapping []string) term {
	return term{
		name:    name,
		isArray: isArray,
		mapping: mapping,
		pm:      genPathMapping(isArray, mapping),
	}
}

func genPathMapping(isArray bool, mapping []string) []interface{} {
	pm := make([]interface{}, len(mapping))
	for i, m := range mapping {
		pm[i] = m
	}
	if isArray {
		pm = append(pm, 0)
	}
	return pm
}
