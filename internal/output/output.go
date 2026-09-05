package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/yashiels/investec/internal/api"
)

// Mode selects the rendering of a result.
type Mode int

const (
	ModeAuto Mode = iota // table on a TTY, JSON otherwise
	ModeJSON
	ModePlain
)

// Options carries the resolved output configuration for a command.
type Options struct {
	Mode    Mode
	NoColor bool
	Out     io.Writer
}

// Resolve determines the effective mode given flags and TTY state. --json and
// --plain are mutually exclusive; the caller validates that before calling.
func Resolve(jsonFlag, plainFlag bool) Mode {
	switch {
	case jsonFlag:
		return ModeJSON
	case plainFlag:
		return ModePlain
	default:
		return ModeAuto
	}
}

// IsTTY reports whether f is a terminal.
func IsTTY(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// effective converts ModeAuto into a concrete mode based on TTY state.
func effective(m Mode, out io.Writer) Mode {
	if m != ModeAuto {
		return m
	}
	if f, ok := out.(*os.File); ok && IsTTY(f) {
		return ModePlain // human table path handled by table renderers below
	}
	return ModeJSON
}

// Envelope renders a decoded API envelope. A table renderer is chosen by the
// caller for the human path; here we handle JSON verbatim and a generic plain
// fallback.
func Envelope(opts Options, r *api.Raw, table func(io.Writer, *api.Raw) error) error {
	m := effective(opts.Mode, opts.Out)
	switch m {
	case ModeJSON:
		return writeJSON(opts.Out, r.Full)
	default:
		// ModePlain (explicit or auto-TTY): use the table renderer when the
		// command supplies one; else fall back to indented JSON.
		if table != nil {
			return table(opts.Out, r)
		}
		return writeJSON(opts.Out, r.Full)
	}
}

func writeJSON(w io.Writer, raw json.RawMessage) error {
	if len(raw) == 0 {
		_, err := io.WriteString(w, "{}\n")
		return err
	}
	var buf []byte
	var v any
	if json.Unmarshal(raw, &v) == nil {
		buf, _ = json.MarshalIndent(v, "", "  ")
	} else {
		buf = raw
	}
	_, err := fmt.Fprintln(w, string(buf))
	return err
}
