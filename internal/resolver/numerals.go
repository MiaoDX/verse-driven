package resolver

var cnDigit = map[rune]int{
	'零': 0, '〇': 0,
	'一': 1, '二': 2, '两': 2, '三': 3, '四': 4,
	'五': 5, '六': 6, '七': 7, '八': 8, '九': 9,
}

// parseChineseNumeral parses Chinese numerals in the range 1..999.
// Recognized forms include 一, 十, 十一, 二十, 二十三, 一百, 一百零一,
// 一百二十三. Returns (n, true) on success, (0, false) on failure.
func parseChineseNumeral(s string) (int, bool) {
	rs := []rune(s)
	if len(rs) == 0 {
		return 0, false
	}

	n := 0
	if i := indexRune(rs, '百'); i >= 0 {
		if i != 1 {
			return 0, false
		}
		d, ok := cnDigit[rs[0]]
		if !ok || d == 0 {
			return 0, false
		}
		n += d * 100
		rs = rs[2:]
		if len(rs) > 0 && (rs[0] == '零' || rs[0] == '〇') {
			rs = rs[1:]
			if len(rs) != 1 {
				return 0, false
			}
			d, ok := cnDigit[rs[0]]
			if !ok {
				return 0, false
			}
			return n + d, true
		}
		if len(rs) == 0 {
			return n, true
		}
	}

	if i := indexRune(rs, '十'); i >= 0 {
		tens := 1
		switch i {
		case 0:
			// implicit one (e.g. 十, 十一)
		case 1:
			d, ok := cnDigit[rs[0]]
			if !ok || d == 0 {
				return 0, false
			}
			tens = d
		default:
			return 0, false
		}
		n += tens * 10
		rest := rs[i+1:]
		switch len(rest) {
		case 0:
			return n, true
		case 1:
			d, ok := cnDigit[rest[0]]
			if !ok {
				return 0, false
			}
			return n + d, true
		default:
			return 0, false
		}
	}

	if len(rs) == 1 {
		d, ok := cnDigit[rs[0]]
		if !ok {
			return 0, false
		}
		return n + d, true
	}
	return 0, false
}

func indexRune(rs []rune, r rune) int {
	for i, c := range rs {
		if c == r {
			return i
		}
	}
	return -1
}
