package blizzard

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func newSysModule(i *Interp, name string) *Module {
	m := &Module{Name: name, Dict: NewDict(16)}
	m.Dict.Set("exec", Native(sysExec))
	m.Dict.Set("sh", Native(sysSh))
	m.Dict.Set("env", Native(bEnv))
	m.Dict.Set("set_env", Native(sysSetEnv))
	m.Dict.Set("envs", Native(sysEnvs))
	m.Dict.Set("args", Native(bArgs))
	m.Dict.Set("cwd", Native(sysCwd))
	m.Dict.Set("cd", Native(sysCd))
	m.Dict.Set("hostname", Native(sysHostname))
	m.Dict.Set("pid", Int(os.Getpid()))
	m.Dict.Set("platform", Str(runtime.GOOS))
	m.Dict.Set("arch", Str(runtime.GOARCH))
	m.Dict.Set("exit", Native(bExit))
	m.Dict.Set("now", Native(bNow))
	m.Dict.Set("time", Native(bNow))
	m.Dict.Set("sleep", Native(bSleep))

	// Convenient fs shortcuts on sys
	m.Dict.Set("read_file", Native(fsRead))
	m.Dict.Set("write_file", Native(fsWrite))
	m.Dict.Set("exists", Native(fsExists))
	m.Dict.Set("list_dir", Native(fsList))
	return m
}

func newFSModule(i *Interp, name string) *Module {
	m := &Module{Name: name, Dict: NewDict(12)}
	m.Dict.Set("read", Native(fsRead))
	m.Dict.Set("write", Native(fsWrite))
	m.Dict.Set("append", Native(fsAppend))
	m.Dict.Set("exists", Native(fsExists))
	m.Dict.Set("remove", Native(fsRemove))
	m.Dict.Set("mkdir", Native(fsMkdir))
	m.Dict.Set("list", Native(fsList))
	m.Dict.Set("is_file", Native(fsIsFile))
	m.Dict.Set("is_dir", Native(fsIsDir))
	m.Dict.Set("stat", Native(fsStat))
	m.Dict.Set("copy", Native(fsCopy))
	return m
}

func sysExec(i *Interp, args []Val) ([]Val, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("sys.exec expects at least a command name string")
	}
	cmdName, err := strArg(args[0], "sys.exec")
	if err != nil {
		return nil, err
	}
	var cmdArgs []string
	if len(args) >= 2 {
		if l, ok := args[1].(List); ok {
			for _, item := range l {
				cmdArgs = append(cmdArgs, BlizzardStr(item))
			}
		} else {
			for _, item := range args[1:] {
				cmdArgs = append(cmdArgs, BlizzardStr(item))
			}
		}
	}

	cmd := exec.Command(string(cmdName), cmdArgs...)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	exitCode := 0
	execErr := cmd.Run()
	if execErr != nil {
		if exitErr, ok := execErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	res := NewDict(4)
	res.Set("code", Int(exitCode))
	res.Set("ok", Bool(exitCode == 0))
	res.Set("stdout", Str(stdoutBuf.String()))
	res.Set("stderr", Str(stderrBuf.String()))
	return []Val{res}, nil
}

func sysSh(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "sys.sh", 1); err != nil {
		return nil, err
	}
	cmdStr, err := strArg(args[0], "sys.sh")
	if err != nil {
		return nil, err
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", string(cmdStr))
	} else {
		cmd = exec.Command("/bin/sh", "-c", string(cmdStr))
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	exitCode := 0
	execErr := cmd.Run()
	if execErr != nil {
		if exitErr, ok := execErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	res := NewDict(4)
	res.Set("code", Int(exitCode))
	res.Set("ok", Bool(exitCode == 0))
	res.Set("stdout", Str(stdoutBuf.String()))
	res.Set("stderr", Str(stderrBuf.String()))
	return []Val{res}, nil
}

func sysSetEnv(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "sys.set_env", 2); err != nil {
		return nil, err
	}
	key, err := strArg(args[0], "sys.set_env")
	if err != nil {
		return nil, err
	}
	val, err := strArg(args[1], "sys.set_env")
	if err != nil {
		return nil, err
	}
	if err := os.Setenv(string(key), string(val)); err != nil {
		return nil, err
	}
	return []Val{}, nil
}

func sysEnvs(i *Interp, args []Val) ([]Val, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("sys.envs expects no arguments")
	}
	raw := os.Environ()
	d := NewDict(len(raw))
	for _, env := range raw {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			d.Set(parts[0], Str(parts[1]))
		}
	}
	return []Val{d}, nil
}

func sysCwd(i *Interp, args []Val) ([]Val, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("sys.cwd expects no arguments")
	}
	pwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return []Val{Str(pwd)}, nil
}

func sysCd(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "sys.cd", 1); err != nil {
		return nil, err
	}
	dir, err := strArg(args[0], "sys.cd")
	if err != nil {
		return nil, err
	}
	if err := os.Chdir(string(dir)); err != nil {
		return nil, err
	}
	return []Val{}, nil
}

func sysHostname(i *Interp, args []Val) ([]Val, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("sys.hostname expects no arguments")
	}
	host, err := os.Hostname()
	if err != nil {
		return []Val{Str("localhost")}, nil
	}
	return []Val{Str(host)}, nil
}

// Filesystem implementations

func fsRead(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "fs.read", 1); err != nil {
		return nil, err
	}
	path, err := strArg(args[0], "fs.read")
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(string(path))
	if err != nil {
		return nil, err
	}
	return []Val{Str(string(data))}, nil
}

func fsWrite(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "fs.write", 2); err != nil {
		return nil, err
	}
	path, err := strArg(args[0], "fs.write")
	if err != nil {
		return nil, err
	}
	content := BlizzardStr(args[1])
	if err := os.WriteFile(string(path), []byte(content), 0644); err != nil {
		return nil, err
	}
	return []Val{}, nil
}

func fsAppend(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "fs.append", 2); err != nil {
		return nil, err
	}
	path, err := strArg(args[0], "fs.append")
	if err != nil {
		return nil, err
	}
	content := BlizzardStr(args[1])
	f, err := os.OpenFile(string(path), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if _, err := f.WriteString(content); err != nil {
		return nil, err
	}
	return []Val{}, nil
}

func fsExists(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "fs.exists", 1); err != nil {
		return nil, err
	}
	path, err := strArg(args[0], "fs.exists")
	if err != nil {
		return nil, err
	}
	_, err = os.Stat(string(path))
	return []Val{Bool(err == nil)}, nil
}

func fsRemove(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "fs.remove", 1); err != nil {
		return nil, err
	}
	path, err := strArg(args[0], "fs.remove")
	if err != nil {
		return nil, err
	}
	err = os.RemoveAll(string(path))
	return []Val{Bool(err == nil)}, nil
}

func fsMkdir(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "fs.mkdir", 1); err != nil {
		return nil, err
	}
	path, err := strArg(args[0], "fs.mkdir")
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(string(path), 0755); err != nil {
		return nil, err
	}
	return []Val{}, nil
}

func fsList(i *Interp, args []Val) ([]Val, error) {
	path := "."
	if len(args) == 1 {
		s, err := strArg(args[0], "fs.list")
		if err != nil {
			return nil, err
		}
		path = string(s)
	} else if len(args) > 1 {
		return nil, fmt.Errorf("fs.list expects at most 1 argument")
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	res := make(List, len(entries))
	for k, e := range entries {
		res[k] = Str(e.Name())
	}
	return []Val{res}, nil
}

func fsIsFile(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "fs.is_file", 1); err != nil {
		return nil, err
	}
	path, err := strArg(args[0], "fs.is_file")
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(string(path))
	if err != nil {
		return []Val{Bool(false)}, nil
	}
	return []Val{Bool(!info.IsDir())}, nil
}

func fsIsDir(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "fs.is_dir", 1); err != nil {
		return nil, err
	}
	path, err := strArg(args[0], "fs.is_dir")
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(string(path))
	if err != nil {
		return []Val{Bool(false)}, nil
	}
	return []Val{Bool(info.IsDir())}, nil
}

func fsStat(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "fs.stat", 1); err != nil {
		return nil, err
	}
	path, err := strArg(args[0], "fs.stat")
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(string(path))
	if err != nil {
		return []Val{Nil}, nil
	}
	d := NewDict(4)
	d.Set("name", Str(info.Name()))
	d.Set("size", Int(info.Size()))
	d.Set("is_dir", Bool(info.IsDir()))
	d.Set("mod_time", Float(float64(info.ModTime().UnixNano())/1e9))
	return []Val{d}, nil
}

func fsCopy(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "fs.copy", 2); err != nil {
		return nil, err
	}
	src, err := strArg(args[0], "fs.copy")
	if err != nil {
		return nil, err
	}
	dst, err := strArg(args[1], "fs.copy")
	if err != nil {
		return nil, err
	}
	inF, err := os.Open(string(src))
	if err != nil {
		return nil, err
	}
	defer inF.Close()
	outF, err := os.Create(string(dst))
	if err != nil {
		return nil, err
	}
	defer outF.Close()
	if _, err := io.Copy(outF, inF); err != nil {
		return nil, err
	}
	return []Val{}, nil
}
