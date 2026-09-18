package snow

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func newCLIModule(i *Interp, name string) *Module {
	m := &Module{Name: name, Dict: NewDict(24)}

	// Colors and styles
	m.Dict.Set("color", Native(cliColor))
	m.Dict.Set("red", Native(cliColorFn("\033[31m")))
	m.Dict.Set("green", Native(cliColorFn("\033[32m")))
	m.Dict.Set("yellow", Native(cliColorFn("\033[33m")))
	m.Dict.Set("blue", Native(cliColorFn("\033[34m")))
	m.Dict.Set("magenta", Native(cliColorFn("\033[35m")))
	m.Dict.Set("cyan", Native(cliColorFn("\033[36m")))
	m.Dict.Set("gray", Native(cliColorFn("\033[90m")))
	m.Dict.Set("white", Native(cliColorFn("\033[37m")))
	m.Dict.Set("bold", Native(cliColorFn("\033[1m")))
	m.Dict.Set("dim", Native(cliColorFn("\033[2m")))
	m.Dict.Set("italic", Native(cliColorFn("\033[3m")))
	m.Dict.Set("underline", Native(cliColorFn("\033[4m")))
	m.Dict.Set("reset", Native(cliColorFn("\033[0m")))

	// Status messages
	m.Dict.Set("info", Native(cliInfo))
	m.Dict.Set("success", Native(cliSuccess))
	m.Dict.Set("warn", Native(cliWarn))
	m.Dict.Set("error", Native(cliError))

	// Interactive prompts
	m.Dict.Set("ask", Native(cliAsk))
	m.Dict.Set("confirm", Native(cliConfirm))

	// Visual components
	m.Dict.Set("box", Native(cliBox))
	m.Dict.Set("table", Native(cliTable))
	m.Dict.Set("divider", Native(cliDivider))
	m.Dict.Set("clear", Native(cliClear))

	// Argument parser
	m.Dict.Set("parse", Native(cliParse))
	m.Dict.Set("args", Native(bArgs))

	return m
}

func cliColorFn(code string) Native {
	return func(i *Interp, args []Val) ([]Val, error) {
		if err := want(args, "color", 1); err != nil {
			return nil, err
		}
		s := SnowStr(args[0])
		return []Val{Str(code + s + "\033[0m")}, nil
	}
}

var ansiMap = map[string]string{
	"black":      "\033[30m",
	"red":        "\033[31m",
	"green":      "\033[32m",
	"yellow":     "\033[33m",
	"blue":       "\033[34m",
	"magenta":    "\033[35m",
	"cyan":       "\033[36m",
	"white":      "\033[37m",
	"gray":       "\033[90m",
	"bold":       "\033[1m",
	"dim":        "\033[2m",
	"italic":     "\033[3m",
	"underline":  "\033[4m",
	"bg_black":   "\033[40m",
	"bg_red":     "\033[41m",
	"bg_green":   "\033[42m",
	"bg_yellow":  "\033[43m",
	"bg_blue":    "\033[44m",
	"bg_magenta": "\033[45m",
	"bg_cyan":    "\033[46m",
	"bg_white":   "\033[47m",
}

func cliColor(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "cli.color", 2); err != nil {
		return nil, err
	}
	name, err := strArg(args[0], "cli.color")
	if err != nil {
		return nil, err
	}
	text := SnowStr(args[1])
	code, ok := ansiMap[strings.ToLower(string(name))]
	if !ok {
		code = "\033[0m"
	}
	return []Val{Str(code + text + "\033[0m")}, nil
}

func cliInfo(i *Interp, args []Val) ([]Val, error) {
	parts := make([]string, len(args))
	for k, a := range args {
		parts[k] = SnowStr(a)
	}
	fmt.Fprintf(i.out, "\033[36;1m[INFO]\033[0m %s\n", strings.Join(parts, " "))
	return []Val{}, nil
}

func cliSuccess(i *Interp, args []Val) ([]Val, error) {
	parts := make([]string, len(args))
	for k, a := range args {
		parts[k] = SnowStr(a)
	}
	fmt.Fprintf(i.out, "\033[32;1m[OK]\033[0m %s\n", strings.Join(parts, " "))
	return []Val{}, nil
}

func cliWarn(i *Interp, args []Val) ([]Val, error) {
	parts := make([]string, len(args))
	for k, a := range args {
		parts[k] = SnowStr(a)
	}
	fmt.Fprintf(i.out, "\033[33;1m[WARN]\033[0m %s\n", strings.Join(parts, " "))
	return []Val{}, nil
}

func cliError(i *Interp, args []Val) ([]Val, error) {
	parts := make([]string, len(args))
	for k, a := range args {
		parts[k] = SnowStr(a)
	}
	fmt.Fprintf(i.errOut, "\033[31;1m[ERROR]\033[0m %s\n", strings.Join(parts, " "))
	return []Val{}, nil
}

func cliAsk(i *Interp, args []Val) ([]Val, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("cli.ask expects a prompt string")
	}
	prompt := SnowStr(args[0])
	defaultVal := ""
	if len(args) >= 2 {
		defaultVal = SnowStr(args[1])
	}
	if defaultVal != "" {
		fmt.Fprintf(i.out, "%s [%s]: ", prompt, defaultVal)
	} else {
		fmt.Fprintf(i.out, "%s: ", prompt)
	}

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && len(line) == 0 {
		if defaultVal != "" {
			return []Val{Str(defaultVal)}, nil
		}
		return []Val{Nil}, nil
	}
	ans := strings.TrimSpace(line)
	if ans == "" && defaultVal != "" {
		ans = defaultVal
	}
	return []Val{Str(ans)}, nil
}

func cliConfirm(i *Interp, args []Val) ([]Val, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("cli.confirm expects a prompt string")
	}
	prompt := SnowStr(args[0])
	defaultYes := false
	if len(args) >= 2 {
		defaultYes = Truthy(args[1])
	}
	hint := "y/N"
	if defaultYes {
		hint = "Y/n"
	}
	fmt.Fprintf(i.out, "%s [%s]: ", prompt, hint)

	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	ans := strings.ToLower(strings.TrimSpace(line))
	if ans == "" {
		return []Val{Bool(defaultYes)}, nil
	}
	if ans == "y" || ans == "yes" || ans == "s" || ans == "si" || ans == "true" {
		return []Val{Bool(true)}, nil
	}
	return []Val{Bool(false)}, nil
}

func cliClear(i *Interp, args []Val) ([]Val, error) {
	fmt.Fprint(i.out, "\033[2J\033[H")
	return []Val{}, nil
}

func cliDivider(i *Interp, args []Val) ([]Val, error) {
	title := ""
	width := 50
	if len(args) >= 1 {
		title = SnowStr(args[0])
	}
	if len(args) >= 2 {
		if n, ok := args[1].(Int); ok && int(n) > 10 {
			width = int(n)
		}
	}
	var sb strings.Builder
	if title != "" {
		tWidth := stringWidth(title)
		fill := width - tWidth - 4
		if fill < 2 {
			fill = 2
		}
		sb.WriteString("── \033[1m" + title + "\033[0m " + strings.Repeat("─", fill))
	} else {
		sb.WriteString(strings.Repeat("─", width))
	}
	divStr := sb.String()
	fmt.Fprintln(i.out, divStr)
	return []Val{Str(divStr)}, nil
}

func cliBox(i *Interp, args []Val) ([]Val, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("cli.box expects content string")
	}
	title := ""
	content := ""
	if len(args) == 1 {
		content = SnowStr(args[0])
	} else {
		title = SnowStr(args[0])
		content = SnowStr(args[1])
	}

	lines := strings.Split(content, "\n")
	titleWidth := 0
	if title != "" {
		titleWidth = stringWidth(title)
	}

	contentWidth := 0
	for _, l := range lines {
		w := stringWidth(l)
		if w > contentWidth {
			contentWidth = w
		}
	}

	// Lateral padding: 2 spaces on the left, 2 spaces on the right
	const padH = 2
	padStr := strings.Repeat(" ", padH)

	// Inner width between the left '│' and right '│'
	// Must accommodate the widest content line + left & right padding.
	totalInner := contentWidth + 2*padH

	// If there is a title, top border has format: "┌── " + title + " " + "─"*fill + "┐"
	// The runes inside top border are: 3 ("── ") + titleWidth + 1 (" ") + fill
	// This must equal totalInner, so fill = totalInner - titleWidth - 4.
	// We require at least 2 dashes after the title ("──"), so totalInner >= titleWidth + 6.
	if title != "" {
		minInnerForTitle := titleWidth + 6
		if minInnerForTitle > totalInner {
			totalInner = minInnerForTitle
		}
	}

	var sb strings.Builder

	// Top border (square: ┌ ┐)
	if title != "" {
		fill := totalInner - titleWidth - 4
		sb.WriteString("┌── " + "\033[1m" + title + "\033[0m" + " " + strings.Repeat("─", fill) + "┐\n")
	} else {
		sb.WriteString("┌" + strings.Repeat("─", totalInner) + "┐\n")
	}

	// Content lines: │ + padStr + line + rem + padStr + │
	for _, l := range lines {
		lWidth := stringWidth(l)
		rem := totalInner - 2*padH - lWidth
		if rem < 0 {
			rem = 0
		}
		sb.WriteString("│" + padStr + l + strings.Repeat(" ", rem) + padStr + "│\n")
	}

	// Bottom border (square: └ ┘)
	sb.WriteString("└" + strings.Repeat("─", totalInner) + "┘")

	boxStr := sb.String()
	fmt.Fprintln(i.out, boxStr)
	return []Val{Str(boxStr)}, nil
}

func cliTable(i *Interp, args []Val) ([]Val, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("cli.table expects headers list and rows list")
	}
	rawHeaders, ok := args[0].(List)
	if !ok {
		return nil, fmt.Errorf("cli.table headers must be a list")
	}
	rawRows, ok := args[1].(List)
	if !ok {
		return nil, fmt.Errorf("cli.table rows must be a list of lists")
	}

	numCols := len(rawHeaders)
	colWidths := make([]int, numCols)

	headers := make([]string, numCols)
	for c, h := range rawHeaders {
		s := SnowStr(h)
		headers[c] = s
		w := stringWidth(s)
		if w > colWidths[c] {
			colWidths[c] = w
		}
	}

	rows := make([][]string, len(rawRows))
	for r, rawRow := range rawRows {
		rowList, ok := rawRow.(List)
		if !ok {
			return nil, fmt.Errorf("cli.table row %d is not a list", r)
		}
		rowStrs := make([]string, numCols)
		for c := 0; c < numCols; c++ {
			if c < len(rowList) {
				rowStrs[c] = SnowStr(rowList[c])
			} else {
				rowStrs[c] = ""
			}
			w := stringWidth(rowStrs[c])
			if w > colWidths[c] {
				colWidths[c] = w
			}
		}
		rows[r] = rowStrs
	}

	var sb strings.Builder

	// Top border: ┌───────┬─────────┐
	sb.WriteString("┌")
	for c, w := range colWidths {
		sb.WriteString(strings.Repeat("─", w+2))
		if c < numCols-1 {
			sb.WriteString("┬")
		}
	}
	sb.WriteString("┐\n")

	// Header row: │ col1  │ col2    │
	sb.WriteString("│")
	for c, h := range headers {
		pad := colWidths[c] - stringWidth(h)
		sb.WriteString(" \033[1m" + h + "\033[0m" + strings.Repeat(" ", pad) + " │")
	}
	sb.WriteString("\n")

	// Header separator: ├───────┼─────────┤
	sb.WriteString("├")
	for c, w := range colWidths {
		sb.WriteString(strings.Repeat("─", w+2))
		if c < numCols-1 {
			sb.WriteString("┼")
		}
	}
	sb.WriteString("┤\n")

	// Data rows
	for _, row := range rows {
		sb.WriteString("│")
		for c, cell := range row {
			pad := colWidths[c] - stringWidth(cell)
			if pad < 0 {
				pad = 0
			}
			sb.WriteString(" " + cell + strings.Repeat(" ", pad) + " │")
		}
		sb.WriteString("\n")
	}

	// Bottom border: └───────┴─────────┘
	sb.WriteString("└")
	for c, w := range colWidths {
		sb.WriteString(strings.Repeat("─", w+2))
		if c < numCols-1 {
			sb.WriteString("┴")
		}
	}
	sb.WriteString("┘")

	tableStr := sb.String()
	fmt.Fprintln(i.out, tableStr)
	return []Val{Str(tableStr)}, nil
}

func cliParse(i *Interp, args []Val) ([]Val, error) {
	rawArgs := i.argv
	boolFlags := map[string]bool{
		"help":    true,
		"h":       true,
		"version": true,
		"v":       true,
		"verbose": true,
		"debug":   true,
		"force":   true,
		"f":       true,
		"quiet":   true,
		"q":       true,
		"all":     true,
		"a":       true,
		"yes":     true,
		"y":       true,
	}

	if len(args) >= 1 {
		if l, ok := args[0].(List); ok {
			rawArgs = make([]string, len(l))
			for k, v := range l {
				rawArgs[k] = SnowStr(v)
			}
		}
	}
	if len(args) >= 2 {
		if boolList, ok := args[1].(List); ok {
			for _, bf := range boolList {
				boolFlags[SnowStr(bf)] = true
			}
		} else if boolDict, ok := args[1].(*Dict); ok {
			for _, k := range boolDict.keys {
				boolFlags[k] = true
			}
		}
	}

	var posArgs List
	flags := NewDict(8)

	for idx := 0; idx < len(rawArgs); idx++ {
		arg := rawArgs[idx]
		if strings.HasPrefix(arg, "--") {
			name := strings.TrimPrefix(arg, "--")
			if strings.Contains(name, "=") {
				parts := strings.SplitN(name, "=", 2)
				flags.Set(parts[0], Str(parts[1]))
			} else if boolFlags[name] {
				flags.Set(name, Bool(true))
			} else if idx+1 < len(rawArgs) && !strings.HasPrefix(rawArgs[idx+1], "-") {
				flags.Set(name, Str(rawArgs[idx+1]))
				idx++
			} else {
				flags.Set(name, Bool(true))
			}
		} else if strings.HasPrefix(arg, "-") && len(arg) > 1 {
			name := strings.TrimPrefix(arg, "-")
			if strings.Contains(name, "=") {
				parts := strings.SplitN(name, "=", 2)
				flags.Set(parts[0], Str(parts[1]))
			} else if boolFlags[name] {
				flags.Set(name, Bool(true))
			} else if idx+1 < len(rawArgs) && !strings.HasPrefix(rawArgs[idx+1], "-") {
				flags.Set(name, Str(rawArgs[idx+1]))
				idx++
			} else {
				flags.Set(name, Bool(true))
			}
		} else {
			posArgs = append(posArgs, Str(arg))
		}
	}

	res := NewDict(2)
	res.Set("args", posArgs)
	res.Set("flags", flags)
	return []Val{res}, nil
}

func stripAnsi(str string) string {
	var sb strings.Builder
	for i := 0; i < len(str); i++ {
		if str[i] == '\033' {
			if i+1 < len(str) && str[i+1] == '[' {
				i += 2
				for i < len(str) && (str[i] < 0x40 || str[i] > 0x7E) {
					i++
				}
				continue
			}
			continue
		}
		sb.WriteByte(str[i])
	}
	return sb.String()
}

func runeWidth(r rune) int {
	// Zero width control characters
	if r < 32 || (r >= 0x7f && r < 0xa0) {
		return 0
	}
	// Variation selectors (e.g. \uFE0F)
	if (r >= 0xFE00 && r <= 0xFE0F) || (r >= 0xE0100 && r <= 0xE01EF) {
		return 0
	}
	// Zero-width spaces, joiners, formatting
	if r == 0x200B || r == 0x200C || r == 0x200D || r == 0x200E || r == 0x200F || r == 0xFEFF || r == 0x00AD {
		return 0
	}
	// Combining diacritical marks
	if (r >= 0x0300 && r <= 0x036F) || (r >= 0x1AB0 && r <= 0x1AFF) ||
		(r >= 0x1DC0 && r <= 0x1DFF) || (r >= 0x20D0 && r <= 0x20FF) ||
		(r >= 0xFE20 && r <= 0xFE2F) {
		return 0
	}
	// Wide characters: CJK, fullwidth
	if (r >= 0x1100 && r <= 0x115F) || // Hangul Jamo
		(r >= 0x2E80 && r <= 0xA4CF && r != 0x303F) || // CJK Radicals, Kangxi, Hiragana, Katakana, CJK Ideographs
		(r >= 0xAC00 && r <= 0xD7A3) || // Hangul Syllables
		(r >= 0xF900 && r <= 0xFAFF) || // CJK Compatibility Ideographs
		(r >= 0xFE10 && r <= 0xFE19) || // Vertical forms
		(r >= 0xFE30 && r <= 0xFE6F) || // CJK Compatibility Forms
		(r >= 0xFF01 && r <= 0xFF60) || // Fullwidth Forms
		(r >= 0xFFE0 && r <= 0xFFE6) ||
		(r >= 0x20000 && r <= 0x2FFFD) ||
		(r >= 0x30000 && r <= 0x3FFFD) {
		return 2
	}
	// Emoji and pictograph ranges
	if (r >= 0x1F300 && r <= 0x1F9FF) || // Miscellaneous Symbols & Pictographs, Emoticons, Supplemental
		(r >= 0x1FA00 && r <= 0x1FAFF) || // Symbols and Pictographs Extended-A
		(r >= 0x2600 && r <= 0x27BF) {   // Miscellaneous Symbols & Dingbats (including ❄ 0x2744, ⚡ 0x26A1, etc.)
		return 2
	}
	return 1
}

func stringWidth(s string) int {
	clean := stripAnsi(s)
	w := 0
	for _, r := range clean {
		w += runeWidth(r)
	}
	return w
}
