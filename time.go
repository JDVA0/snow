package snow

import (
	"fmt"
	"strings"
	"time"
)

func newTimeModule(i *Interp, name string) *Module {
	m := &Module{Name: name, Dict: NewDict(8)}
	m.Dict.Set("sleep", Native(timeSleep))
	m.Dict.Set("now", Native(timeNow))
	m.Dict.Set("unix", Native(timeUnix))
	m.Dict.Set("unix_ms", Native(timeUnixMs))
	m.Dict.Set("iso", Native(timeISO))
	m.Dict.Set("since", Native(timeSince))
	return m
}

func timeSleep(i *Interp, args []Val) ([]Val, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("time.sleep expects 1 argument (seconds or duration string)")
	}
	switch v := args[0].(type) {
	case Int:
		time.Sleep(time.Duration(v) * time.Second)
	case Float:
		time.Sleep(time.Duration(float64(v) * float64(time.Second)))
	case Str:
		d, err := time.ParseDuration(string(v))
		if err != nil {
			return nil, fmt.Errorf("time.sleep: invalid duration '%s': %v", v, err)
		}
		time.Sleep(d)
	default:
		return nil, fmt.Errorf("time.sleep expects a number of seconds or duration string like '500ms', '2s'")
	}
	return []Val{Nil}, nil
}

// time.now([format]) — returns formatted current time.
// Supports strftime syntax (%Y, %m, %d, %H, %M, %S) or standard Go layout.
func timeNow(i *Interp, args []Val) ([]Val, error) {
	now := time.Now()
	if len(args) == 0 {
		return []Val{Str(now.Format("2006-01-02 15:04:05"))}, nil
	}
	fmtStr, err := strArg(args[0], "time.now")
	if err != nil {
		return nil, err
	}
	layout := convertStrftime(string(fmtStr))
	return []Val{Str(now.Format(layout))}, nil
}

func timeUnix(i *Interp, args []Val) ([]Val, error) {
	return []Val{Int(time.Now().Unix())}, nil
}

func timeUnixMs(i *Interp, args []Val) ([]Val, error) {
	return []Val{Int(time.Now().UnixMilli())}, nil
}

func timeISO(i *Interp, args []Val) ([]Val, error) {
	return []Val{Str(time.Now().Format(time.RFC3339))}, nil
}

func timeSince(i *Interp, args []Val) ([]Val, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("time.since expects a starting timestamp")
	}
	var t0 float64
	switch v := args[0].(type) {
	case Int:
		t0 = float64(v)
	case Float:
		t0 = float64(v)
	default:
		return nil, fmt.Errorf("time.since expects a numeric timestamp")
	}
	now := float64(time.Now().UnixNano()) / 1e9
	return []Val{Float(now - t0)}, nil
}

func convertStrftime(f string) string {
	repl := strings.NewReplacer(
		"%Y", "2006",
		"%y", "06",
		"%m", "01",
		"%d", "02",
		"%H", "15",
		"%I", "03",
		"%M", "04",
		"%S", "05",
		"%p", "PM",
		"%a", "Mon",
		"%A", "Monday",
		"%b", "Jan",
		"%B", "January",
		"%z", "-0700",
		"%Z", "MST",
	)
	return repl.Replace(f)
}
