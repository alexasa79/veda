package editor

import "unicode/utf8"

func clampCursorByte(line string, x int) int {
	if x < 0 {
		return 0
	}
	if x > len(line) {
		return len(line)
	}
	return x
}

// clampCursorToRuneStart ensures x is on a UTF-8 rune boundary at or before x.
func clampCursorToRuneStart(line string, x int) int {
	x = clampCursorByte(line, x)
	for x > 0 && x < len(line) && !utf8.RuneStart(line[x]) {
		x--
	}
	return x
}

func prevRuneStart(line string, x int) int {
	x = clampCursorToRuneStart(line, x)
	if x == 0 {
		return 0
	}
	_, size := utf8.DecodeLastRuneInString(line[:x])
	if size <= 0 {
		return x - 1
	}
	return x - size
}

func nextRuneStart(line string, x int) int {
	x = clampCursorToRuneStart(line, x)
	if x >= len(line) {
		return len(line)
	}
	_, size := utf8.DecodeRuneInString(line[x:])
	if size <= 0 {
		return x + 1
	}
	return x + size
}

func lastRuneStart(line string) int {
	if len(line) == 0 {
		return 0
	}
	_, size := utf8.DecodeLastRuneInString(line)
	if size <= 0 {
		return len(line) - 1
	}
	return len(line) - size
}
