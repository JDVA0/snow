package snow

import (
	"fmt"
	"sync"
	"time"
)

func newTaskModule(i *Interp, name string) *Module {
	m := &Module{Name: name, Dict: NewDict(4)}
	m.Dict.Set("every", Native(taskEvery(i)))
	m.Dict.Set("after", Native(taskAfter(i)))
	return m
}

func parseTaskDuration(arg Val, fnName string) (time.Duration, error) {
	switch v := arg.(type) {
	case Int:
		return time.Duration(v) * time.Second, nil
	case Float:
		return time.Duration(float64(v) * float64(time.Second)), nil
	case Str:
		d, err := time.ParseDuration(string(v))
		if err != nil {
			return 0, fmt.Errorf("%s: invalid duration '%s': %v", fnName, v, err)
		}
		return d, nil
	default:
		return 0, fmt.Errorf("%s: expects duration as string (e.g. '5s', '1m') or number of seconds", fnName)
	}
}

func taskEvery(i *Interp) Native {
	return func(caller *Interp, args []Val) ([]Val, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("task.every expects (interval, fn)")
		}
		dur, err := parseTaskDuration(args[0], "task.every")
		if err != nil {
			return nil, err
		}
		fn := args[1]
		if _, ok := fn.(*Fn); !ok {
			if _, ok := fn.(Native); !ok {
				return nil, fmt.Errorf("task.every: second argument must be a function")
			}
		}

		ticker := time.NewTicker(dur)
		stopCh := make(chan struct{})
		var once sync.Once
		active := true

		go func() {
			for {
				select {
				case <-ticker.C:
					_, _ = i.InvokeSafe(fn, nil)
				case <-stopCh:
					ticker.Stop()
					return
				}
			}
		}()

		handle := NewDict(4)
		handle.Set("stop", Native(func(subI *Interp, a []Val) ([]Val, error) {
			once.Do(func() {
				active = false
				close(stopCh)
			})
			return []Val{Nil}, nil
		}))
		handle.Set("active", Native(func(subI *Interp, a []Val) ([]Val, error) {
			return []Val{Bool(active)}, nil
		}))

		return []Val{handle}, nil
	}
}

func taskAfter(i *Interp) Native {
	return func(caller *Interp, args []Val) ([]Val, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("task.after expects (delay, fn)")
		}
		dur, err := parseTaskDuration(args[0], "task.after")
		if err != nil {
			return nil, err
		}
		fn := args[1]
		if _, ok := fn.(*Fn); !ok {
			if _, ok := fn.(Native); !ok {
				return nil, fmt.Errorf("task.after: second argument must be a function")
			}
		}

		timer := time.NewTimer(dur)
		active := true
		var once sync.Once

		go func() {
			select {
			case <-timer.C:
				active = false
				_, _ = i.InvokeSafe(fn, nil)
			}
		}()

		handle := NewDict(4)
		handle.Set("stop", Native(func(subI *Interp, a []Val) ([]Val, error) {
			once.Do(func() {
				active = false
				timer.Stop()
			})
			return []Val{Nil}, nil
		}))
		handle.Set("active", Native(func(subI *Interp, a []Val) ([]Val, error) {
			return []Val{Bool(active)}, nil
		}))

		return []Val{handle}, nil
	}
}
