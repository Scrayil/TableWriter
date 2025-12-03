package TableWriter

import (
	"bytes"
	"io"
	"os"
	"regexp"
	"strings"
	"unicode"

	"github.com/Scrayil/TableWriter/utils"
)

var escapeColorCodesRegex = regexp.MustCompile(`\033\[[0-9;]+m`)

const (
	// StripColours Removes ANSI color codes from output text
	StripColours uint = 1 << iota
	// AlignMiddle centers the text horizontally in the column
	AlignMiddle
	// AlignRight shifts each cell content to the right
	AlignRight
	// RemoveLeastPad removes the minimum padding bytes used to separate text from nearby columns.
	RemoveLeastPad
	// PreserveLongFields disables the logic used to truncate long strings in the table.
	// This flag disables padding completely
	PreserveLongFields
	// AsciiTable allows using only ASCII dividers.
	// Useful for environments that do not support utf-8 encodings
	AsciiTable
)

// column represents the base structure to keep track of each table's column width over time
// This is later used to determine the minimum columns' width and related fields' padding
type column struct {
	textWidth int
}

// Writer the [io.Writer] struct used to process and format received text in order to create nice looking tables
// and style them according to the specified flags
type Writer struct {
	// Configuration
	output   io.Writer
	dividers utils.Dividers
	flags    uint

	// State
	termCols int
	buffer   []byte
	columns  []column
	lines    []string
}

// NewWriter allocates and initializes a new [Writer].
// The parameters are the same as for the init function.
func NewWriter(output io.Writer, flags uint) *Writer {
	return new(Writer).init(output, flags)
}

// Write appends the external content received to the [Writer]'s internal buffer
// This is automatically called by functions piping data into this io.Writer
func (w *Writer) Write(buf []byte) (n int, err error) {
	w.buffer = append(w.buffer, buf...)
	return len(buf), nil
}

// cleanInvisibleChars removes all control, format, non-spacing, and
// non-standard space characters (Zs).
// This preserve \t, \n, and the ANSI escape character (\x1b).
func cleanInvisibleChars(s string) string {
	return strings.Map(func(r rune) rune {
		// Do nothing with characters that are needed by the Writer to correctly format the output
		if r == ' ' || r == '\x1b' || r == '\t' || r == '\n' {
			return r
		}

		// Removing invisible characters that cause misalignment
		// Unicode categories to remove:
		// Cf: format character (zero width joiner, LTR/RTL symbols)
		// Cc: control character (null, carriage return other than \n)
		// Mn: non-spacing characters (Accents or symbols that take no space)
		// Zs: space separator (non-breaking space: U+00A0 etc.)
		if unicode.Is(unicode.Cf, r) ||
			unicode.Is(unicode.Cc, r) ||
			unicode.Is(unicode.Mn, r) ||
			unicode.Is(unicode.Zs, r) {
			return -1
		}

		// Any uncaught space character is removed to avoid alignment problems
		if unicode.IsSpace(r) {
			return -1
		}

		// Preserves everything else
		return r
	}, s)
}

// Flush processes the output buffer by creating the corresponding table content and sends it to the chosen
// output's file descriptor
func (w *Writer) Flush() (err error) {
	defer w.Clear()
	cleanedBuffer := cleanInvisibleChars(string(w.buffer))
	w.lines = strings.Split(cleanedBuffer, "\n")
	if len(w.lines[len(w.lines)-1]) == 0 {
		w.lines = w.lines[:len(w.lines)-1]
	}
	formattedBuffer := w.formatBuffer()

	n, err := w.output.Write(formattedBuffer)
	if err != nil || n != len(formattedBuffer) {
		return io.ErrShortWrite
	}
	return nil
}

// Clear resets the state of the [Writer] to remove any traces of previously flushed content
func (w *Writer) Clear() {
	w.columns = make([]column, 0)
	w.buffer = make([]byte, 0)
	w.lines = make([]string, 0)
}

// init initializes the [Writer] by defining its initial configuration and state
func (w *Writer) init(output io.Writer, flags uint) *Writer {
	if flags&AsciiTable != 0 {
		w.dividers = utils.Dividers{
			HLine:  "-",
			VLine:  "|",
			TL:     "+", // Use '+' for corners/junctions
			TR:     "+",
			BL:     "+",
			BR:     "+",
			TUp:    "+", // Should be '+' to mark the intersection point
			TDown:  "+",
			Cross:  "+",
			VLeft:  "+",
			VRight: "+",
		}
	} else {
		w.dividers = utils.Dividers{
			HLine:  "─",
			VLine:  "│",
			TL:     "┌",
			TR:     "┐",
			BL:     "└",
			BR:     "┘",
			TUp:    "┬",
			TDown:  "┴",
			Cross:  "┼",
			VLeft:  "├",
			VRight: "┤",
		}
	}

	w.termCols, _, _ = utils.GetTerminalSize(os.Stdout.Fd())
	w.output = output
	w.flags = flags
	w.Clear()
	return w
}

// truncateLongField cuts the last exceeding field and postpends a suffix indicating that the output has been truncated.
// If the [PreserveLongFields] flag is set, the cut operation is not performed
func (w *Writer) truncateLongField(l int, c int, maxFieldLen int, fields []string, colorlessFields [][]string) {
	if w.flags&PreserveLongFields == 0 && w.termCols > 0 && c == len(fields)-1 && len(colorlessFields[l][c]) > maxFieldLen {
		newColorLessStr := colorlessFields[l][c][:maxFieldLen-5] + "[...]"
		newFieldStr := strings.Replace(fields[c], colorlessFields[l][c], colorlessFields[l][c][:maxFieldLen-5]+utils.ColorOrange+"[...]"+utils.ColorReset, -1)
		w.lines[l] = strings.Replace(w.lines[l], fields[c], newFieldStr, -1)
		colorlessFields[l][c] = newColorLessStr
	}
}

// createColumns computes the total width of each field for each line and updates the column structure to keep track of
// minimum required sizes
func (w *Writer) createColumns() [][]string {
	colorlessFields := make([][]string, len(w.lines))
	for l, line := range w.lines {
		if len(line) == 0 {
			continue
		}
		fields := strings.Split(line, "\t")
		maxFieldLen := w.termCols/len(fields) - 3
		// Ensures there are enough columns for each field
		if len(fields) > len(w.columns) {
			w.columns = append(w.columns, make([]column, len(fields)-len(w.columns))...)
		}

		// Computing maximum widths
		colorlessFields[l] = make([]string, len(fields))
		for c := range fields {
			escapeColorCodes := escapeColorCodesRegex.FindAllString(fields[c], -1)
			colorlessFields[l][c] = strings.Clone(fields[c])
			for _, cc := range escapeColorCodes {
				colorlessFields[l][c] = strings.Replace(colorlessFields[l][c], cc, "", -1)
			}

			w.truncateLongField(l, c, maxFieldLen, fields, colorlessFields)
			columnWidth := len(colorlessFields[l][c])
			if columnWidth > w.columns[c].textWidth {
				w.columns[c].textWidth = columnWidth
			}
		}
	}
	return colorlessFields
}

// getPadding determines the correct amount of spaces in order to correctly position and align each field inside its column
func (w *Writer) getPadding(c int, colorlessField string) (int, []byte, []byte) {
	totalPadding := w.columns[c].textWidth - len(colorlessField)
	if w.flags&RemoveLeastPad == 0 {
		totalPadding += 1
	}
	var leftPaddingStr []byte
	var rightPaddingStr []byte
	if w.flags&PreserveLongFields == 0 {
		if w.flags&AlignMiddle != 0 {
			if w.flags&RemoveLeastPad == 0 {
				totalPadding += 1
			}
			halfPadding := totalPadding / 2
			leftPaddingStr = bytes.Repeat([]byte{' '}, halfPadding)
			rightPaddingStr = bytes.Repeat([]byte{' '}, totalPadding-halfPadding)
		} else if w.flags&AlignRight != 0 {
			leftPaddingStr = bytes.Repeat([]byte{' '}, totalPadding)
		} else {
			rightPaddingStr = bytes.Repeat([]byte{' '}, totalPadding)
		}
	} else {
		totalPadding = 0
		leftPaddingStr = bytes.Repeat([]byte{' '}, 0)
		rightPaddingStr = bytes.Repeat([]byte{' '}, 0)
	}
	return totalPadding, leftPaddingStr, rightPaddingStr
}

// updateHLine computes the length of the horizontal divider line and appends new dividers to it based on the currently
// available space in the terminal
func (w *Writer) updateHLine(hLine *string, hLineLength int, l int, isLastRow bool, isLastField bool) {
	// Unicode dividers might consist into multiple bytes, but represent only 1 visual character
	// In order to always compute the visual hLine, we must divide its length by the number of bytes used by a divider
	// This only works because hLine is only made up from the same repeated divider types (all 3 bytes for box lines)
	availableTermSpace := w.termCols - (len(*hLine) / len(w.dividers.HLine))
	var xDivider string
	switch {
	case l == 0 && !isLastField:
		xDivider = w.dividers.TUp
	case l == 0:
		xDivider = w.dividers.TR
	case isLastRow && !isLastField:
		xDivider = w.dividers.TDown
	case isLastRow:
		xDivider = w.dividers.BR
	case isLastField:
		xDivider = w.dividers.VRight
	default:
		xDivider = w.dividers.Cross
	}

	// Adding a prefix to the hLine to render the first column's left border
	if isLastField {
		if l == 0 {
			*hLine = w.dividers.TL + *hLine
		} else if isLastRow {
			*hLine = w.dividers.BL + *hLine
		} else {
			*hLine = w.dividers.VLeft + *hLine
		}
	}
	if availableTermSpace > 0 {
		if availableTermSpace >= hLineLength {
			*hLine += strings.Repeat(w.dividers.HLine, hLineLength-1) + xDivider
		} else {
			*hLine += strings.Repeat(w.dividers.HLine, availableTermSpace-1) + xDivider
		}
	}
}

// createTable transforms the [Writer]'s internal buffer data into a styled and formatted table
func (w *Writer) createTable(colorlessFields [][]string) []byte {
	formattedBuffer := make([]byte, 0)
	for l, line := range w.lines {
		if len(line) == 0 {
			continue
		}
		fields := strings.Split(line, "\t")

		// Writing to the output
		hLine := ""
		prefixHLine := ""
		for c, field := range fields {
			if w.flags&StripColours != 0 {
				field = colorlessFields[l][c]
			}

			totalPadding, leftPaddingStr, rightPaddingStr := w.getPadding(c, colorlessFields[l][c])
			// Used to render the first column's left border segments
			if c == 0 {
				formattedBuffer = append(append(append(append(append(formattedBuffer, w.dividers.VLine...), leftPaddingStr...), field...), rightPaddingStr...), w.dividers.VLine...)
			} else {
				formattedBuffer = append(append(append(append(formattedBuffer, leftPaddingStr...), field...), rightPaddingStr...), w.dividers.VLine...)
			}
			hLineLength := len(colorlessFields[l][c]) + totalPadding + 1
			// Necessary to add a top border to the table header or first row
			if l == 0 {
				w.updateHLine(&prefixHLine, hLineLength, l, l == len(w.lines)-1, c == len(fields)-1)
			}
			w.updateHLine(&hLine, hLineLength, l+1, l == len(w.lines)-1, c == len(fields)-1)

		}
		// Necessary to add a top border to the table header or first row
		if l == 0 {
			formattedBuffer = append([]byte(prefixHLine+"\n"), formattedBuffer...)
			prefixHLine = ""
		}
		formattedBuffer = append(formattedBuffer, '\n')
		formattedBuffer = append(formattedBuffer, hLine...)
		formattedBuffer = append(formattedBuffer, '\n')
	}
	return formattedBuffer
}

// formatBuffer processes the [Writer]'s buffered data, restyles it and generates a formatted output string that
// can be sent to the final [io.Writer]
func (w *Writer) formatBuffer() []byte {
	colorlessFields := w.createColumns()
	return w.createTable(colorlessFields)
}
