package goshift

import (
	"fmt"
	"strings"
)

type manualShifter struct {
	elems []elem
}

type elem struct {
	terms   []term
	origSrc string
	origDst string
}

func (ms *manualShifter) Apply(src map[string]interface{}, opts ...ShiftOption) (map[string]interface{}, error) {
	op := &shiftOp{}
	for _, opt := range opts {
		opt(op)
	}

	var m map[string]interface{}
	if op.dst != nil {
		m = op.dst
	} else {
		m = make(map[string]interface{}, len(ms.elems))
	}

	currentElem := elem{}
	reporter := func(path []interface{}, value interface{}) {
		if op.reporter != nil {
			value = op.reporter(currentElem.origSrc, currentElem.origDst, value)
		}
		if value != nil {
			set(m, path, value)
		}
	}
	for _, elem := range ms.elems {
		currentElem = elem
		if err := shift(elem.terms, make([]interface{}, 0, len(elem.terms)*2), src, reporter); err != nil {
			if op.errCatcher != nil {
				continueShifting := op.errCatcher(err)
				if continueShifting {
					continue
				} else {
					return nil, err
				}
			}
			return nil, err
		}
	}
	return m, nil
}

func NewShifter(mapping map[string]string) (Shifter, error) {
	multiMap := make(map[string][]string, len(mapping))
	for src, dst := range mapping {
		multiMap[src] = []string{dst}
	}
	return NewShifterV2(multiMap)
}

func NewShifterV2(mapping map[string][]string) (Shifter, error) {
	elems, err := compile(mapping)
	if err != nil {
		return nil, err
	}

	shifter := &manualShifter{elems: elems}

	return shifter, nil
}

func compile(mapping map[string][]string) ([]elem, error) {
	arr := make([]elem, 0, len(mapping))
	for srcPath, dstArray := range mapping {
		for _, dstPath := range dstArray {
			terms, err := compilePair(srcPath, dstPath)
			if err != nil {
				return nil, err
			}
			e := elem{terms: terms, origSrc: srcPath, origDst: dstPath}
			arr = append(arr, e)
		}
	}

	return arr, nil
}

func compilePair(srcPath, dstPath string) ([]term, error) {
	srcParts := strings.Split(srcPath, ".")
	dstParts := strings.Split(dstPath, ".")

	srcArrayIndexes := getArraysIndexes(srcParts)
	dstArrayIndexes := getArraysIndexes(dstParts)
	if len(srcArrayIndexes) != len(dstArrayIndexes) {
		return nil, fmt.Errorf("invalid array mapping: %s -> %s", srcPath, dstPath)
	}

	var terms []term
	if len(srcArrayIndexes) == 0 {
		terms = compileTerms(srcParts, dstParts)
	} else {
		prevSrcArrayIndex := 0
		prevDstArrayIndex := 0
		terms = make([]term, 0)
		for i, srcArrayIndex := range srcArrayIndexes {
			dstArrayIndex := dstArrayIndexes[i]

			currentSrc := srcParts[prevSrcArrayIndex:srcArrayIndex]
			currentDst := dstParts[prevDstArrayIndex:dstArrayIndex]
			terms = append(terms, compileTerms(currentSrc, currentDst)...)

			mapping := make([]string, 0)
			lastPart := strings.Replace(dstParts[dstArrayIndex], "[]", "", -1)
			if len(terms) == 0 {
				mapping = append(mapping, dstParts[prevSrcArrayIndex:dstArrayIndex]...)
				mapping = append(mapping, lastPart)
			} else {
				mapping = append(mapping, lastPart)
			}
			terms = append(terms, newTerm(
				strings.Replace(srcParts[srcArrayIndex], "[]", "", -1),
				true,
				mapping,
			))

			prevSrcArrayIndex = srcArrayIndex + 1
			prevDstArrayIndex = dstArrayIndex + 1
		}

		currentSrc := srcParts[prevSrcArrayIndex:]
		currentDst := dstParts[prevDstArrayIndex:]
		terms = append(terms, compileTerms(currentSrc, currentDst)...)
	}

	return terms, nil
}

func getArraysIndexes(src []string) []int {
	arr := make([]int, 0, 2)
	for i, s := range src {
		if strings.Contains(s, "[]") {
			arr = append(arr, i)
		}
	}
	return arr
}

func compileTerms(srcParts, dstParts []string) []term {
	terms := make([]term, len(srcParts))

	if len(srcParts) == len(dstParts) {
		for i, s := range srcParts {
			terms[i] = newTerm(s, false, []string{dstParts[i]})
		}
	} else if len(srcParts) > len(dstParts) && len(dstParts) > 0 {
		for i, s := range srcParts {
			isLast := i == len(srcParts)-1
			if i < len(dstParts)-1 {
				terms[i] = newTerm(s, false, []string{dstParts[i]})
			} else if i >= len(dstParts)-1 && !isLast {
				terms[i] = newTerm(s, false, []string{})
			} else if isLast {
				terms[i] = newTerm(s, false, []string{dstParts[len(dstParts)-1]})
			}
		}
	} else {
		for i, s := range srcParts {
			isLast := i == len(srcParts)-1
			if isLast {
				var mapping []string
				if len(dstParts) > 0 {
					mapping = dstParts[i:]
				}
				terms[i] = newTerm(s, false, mapping)
			} else if i < len(dstParts)-1 {
				terms[i] = newTerm(s, false, []string{dstParts[i]})
			}
		}
	}
	return terms
}

func shift(terms []term, dstPath []interface{}, src interface{}, report func(path []interface{}, value interface{})) error {
	isLast := len(terms) == 0
	if isLast {
		report(dstPath, src)
		return nil
	}
	currentTerm := terms[0]
	switch v := src.(type) {
	case map[string]interface{}:
		val, _ := v[currentTerm.name]
		if currentTerm.isArray {
			return shift(terms, append(dstPath, currentTerm.pm...), val, report)
		} else {
			return shift(terms[1:], append(dstPath, currentTerm.pm...), val, report)
		}

	case []interface{}:
		if currentTerm.isArray {
			lastIndex := len(dstPath) - 1
			for i, val := range v {
				dstPath[lastIndex] = i
				if err := shift(terms[1:], dstPath, val, report); err != nil {
					return err
				}
			}
		} else {
			if len(v) > 0 {
				return shift(terms, dstPath, v[0], report)
			}
		}
	case interface{}:
		if currentTerm.isArray {
			return fmt.Errorf("expecting array, got primitive. Key: %s", fmtKey(dstPath, currentTerm))
		} else {
			return fmt.Errorf("expecting obj, got primitive. Key: %s", fmtKey(dstPath, currentTerm))
		}
	}

	return nil
}

func set(dst interface{}, path []interface{}, value interface{}) interface{} {
	if len(path) == 0 {
		return value
	}

	p := path[0]
	switch p := p.(type) {
	case string:
		if dst == nil {
			dst = make(map[string]interface{}, 3)
		}
		if m, ok := dst.(map[string]interface{}); ok {
			if val, ok := m[p]; ok {
				m[p] = set(val, path[1:], value)
			} else {
				m[p] = set(nil, path[1:], value)
			}
		}

	case int:
		if dst == nil {
			dst = make([]interface{}, 0, 3)
		}
		if arr, ok := dst.([]interface{}); ok {
			i := len(arr)
			if i == p {
				arr = append(arr, set(nil, path[1:], value))
			} else if i < p {
				toInsert := make([]interface{}, p-i)
				newItem := set(nil, path[1:], value)
				switch newItem.(type) {
				case []interface{}:
					for i := range toInsert {
						toInsert[i] = make([]interface{}, 0)
					}
				case map[string]interface{}:
					for i := range toInsert {
						toInsert[i] = make(map[string]interface{}, 0)
					}
				}
				arr = append(arr[:i], append(toInsert, newItem)...)
			} else {
				arr[p] = set(arr[p], path[1:], value)
			}

			dst = arr
		}

	}

	return dst
}

func fmtKey(path []interface{}, currentTerm term) string {
	parts := make([]string, len(path))
	for i, p := range path {
		switch p := p.(type) {
		case string:
			parts[i] = p
		case int:
			parts[i] = fmt.Sprintf("[%d]", p)
		default:
			continue
		}
	}

	parts = append(parts, currentTerm.mapping...)

	return strings.Join(parts, ".")
}
