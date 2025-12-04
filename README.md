# 📝 TableWriter

[![Go Reference](https://pkg.go.dev/badge/github.com/Scrayil/TableWriter.git.svg)](https://pkg.go.dev/github.com/Scrayil/TableWriter)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

**TableWriter** is a Go package that implements the standard `io.Writer` interface to automatically format tab-separated text (`\t`) into properly aligned and stylized tables, designed for console output (CLI).

It uses the writer concept to process the input data buffer, calculate the optimal column width based on the terminal size, and send the formatted table to the desired output.

## 🚀 Installation

To add `TableWriter` to your Go project:

```bash
go get github.com/Scrayil/TableWriter
```

## 💡 Basic Usage

The `TableWriter` package uses Go's `io.Writer` interface to process data. You should instantiate a new `Writer` by specifying the destination output (e.g., `os.Stdout`) and configuration flags.  
Data for the table is written to the `Writer`, where columns are separated by the tab character (`\t`) and rows by the newline character (`\n`). The table is only rendered when the `Flush()` method is called. 

```go
package main

import (
	"fmt"
	"os"

	"github.com/Scrayil/TableWriter"
)

func main() {
	// Creates a new Writer that writes to Stdout, without specifying any flag
	w := TableWriter.NewWriter(os.Stdout, 0)

	// Writing table rows
	fmt.Fprintf(w, "%s\t%s\n", "Name", "Age")
	fmt.Fprintf(w, "%s\t%d\n", "Alice", 30)
	fmt.Fprintf(w, "%s\t%d\n", "Bob", 25)

	// Flush renders the actual table
	if err := w.Flush(); err != nil {
		fmt.Println("Flushing error:", err)
	}

	// Output result
	//┌──────┬────┐
	//│Name  │Age │
	//├──────┼────┤
	//│Alice │30  │
	//├──────┼────┤
	//│Bob   │25  │
	//└──────┴────┘
}
```

## ⚙️ Configuration Flags

You can customise the behaviour and style of the table by passing one or more flags when creating the Writer (using the bitwise OR operator |).  
|Constant|Value|Description|
|:-|:-|:-|
|0|0|**Left** alignment, **Unicode** borders and **enabled** truncation.|
|TableWriter.StripColours|1 << 0|Removes ANSI colour codes from the output.|
|TableWriter.AlignMiddle|1 << 1|Aligns the content of each column to the **Centre**.|
|TableWriter.AlignRight|1 << 2|Aligns the contents of each column to the **Right**.|
|TableWriter.RemoveLeastPad|1 << 3|Removes the minimum padding space (1 byte) used to separate text from neighbouring columns.|
|TableWriter.PreserveLongFields|1 << 4|**Disables truncation** of long strings. This completely disables padding if the column width exceeds the terminal width, allowing long lines to wrap.|
|TableWriter.AsciiTable|1 << 5|Uses only **ASCII** separator characters (+, -, \|)|

**Note on Alignment**: The `AlignMiddle` and `AlignRight` flags are mutually exclusive. If both are specified, `AlignRight` logically prevails due to the implementation.

### Example with Flags

```go
// Create a Writer with centre alignment and remove ANSI colour codes
flags := TableWriter.AlignMiddle | TableWriter.StripColours

w := TableWriter.NewWriter(os.Stdout, flags)
// ... write data ...
w.Flush()
```

## 🛠️ Main Methods

`NewWriter(output io.Writer, flags uint) *Writer`
Instantiates and initialises a new Writer with the specified output and configuration flags.

`Write(buf []byte) (n int, err error)`
Implements the `io.Writer` interface. Appends tabulated data to the internal buffer of the `Writer`.

`Flush() (err error)`
Processes the internal buffer, calculates the table formatting (column width, truncation, alignment) and writes the formatted table to the destination `io.Writer`. **Must be called to display the table.**
`Clear()`
Resets the internal state of the `Writer` (buffer, columns, and rows), removing any traces of previously processed content. It is automatically called by **Flush().**

## 🎨 ANSI Colour Support

The package can handle ANSI colour codes within cells. When colour codes are present, the package calculates the column width based on **visual length** (ignoring escape codes). If a string is truncated and contains colour codes, the package attempts to preserve the colours and insert orange [...] notation.

## Example output with ASCII colours and truncated fields

<img width="995" height="660" alt="image" src="https://github.com/user-attachments/assets/de66b6bc-3301-46b6-b976-31f18a1e5e8f" />
