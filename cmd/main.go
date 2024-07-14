package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

type (
	fn func([]string)
)

type Hist struct {
	user_cmd []string
	cmd_file *os.File
}

var builtin = make(map[string]fn)

func main() {
	initcmd()
	cmd_hist := newHistory()
	signal.Ignore(os.Interrupt)
	for {
		fmt.Fprint(os.Stdout, "$ ")
		input, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			os.Exit(1)
		}
		input = strings.TrimRight(input, "\n")
		cmd_hist.addToHistory(input)
		cmds := strings.Split(strings.TrimSpace(input), "|")
		if len(cmds) != 1 {
			pipedCommand(input)
		} else {
			inputs := strings.Split(strings.TrimSpace(input), " ")
			args := inputs[1:]
			if inputs[0] == "history" {
				history(cmd_hist)
				continue
			}
			cmdfc, avail := builtin[inputs[0]]
			if avail {
				cmdfc(args)
			} else {
				command, err := commandPath(inputs[0])
				if err == nil {
					program := exec.Command(command, inputs[1:]...)

					program.Stdout = os.Stdout
					program.Stderr = os.Stderr

					err = program.Run()
					if err != nil {
						fmt.Fprintf(os.Stdout, "%s: command not found\n", strings.TrimRight(input, "\n"))
					}
				} else {
					fmt.Fprintf(os.Stdout, "%s: command not found\n", strings.TrimRight(input, "\n"))
				}
			}
		}
	}
}

func initcmd() {
	builtin["echo"] = echo
	builtin["exit"] = exit
	builtin["type"] = typecmd
	builtin["pwd"] = pwd
	builtin["cd"] = cd
}

func newHistory() *Hist {
	path, _ := os.UserHomeDir()
	var hist_file = filepath.Join(path, ".msh_history")
	var h Hist
	h.cmd_file, _ = os.OpenFile(hist_file, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0600)
	// defer h.cmd_file.Close()
	scanner := bufio.NewScanner(h.cmd_file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		h.user_cmd = append(h.user_cmd, scanner.Text())
	}
	return &h
}

func (h *Hist) addToHistory(command string) {
	if len(h.user_cmd) == 0 || h.user_cmd[len(h.user_cmd)-1] != command {
		h.user_cmd = append(h.user_cmd, command)
		h.cmd_file.WriteString(command + "\n")
	}
}

func pipedCommand(input string) {
	input = strings.TrimSpace(input)
	commands := strings.Split(input, "|")
	var cmds []*exec.Cmd
	var output io.ReadCloser

	for _, command := range commands {
		args := strings.Split(strings.TrimSpace(command), " ")
		program := exec.Command(args[0], args[1:]...)
		program.Stderr = os.Stderr
		cmds = append(cmds, program)

		if output != nil {
			program.Stdin = output
		}
		output, _ = program.StdoutPipe()
	}
	if len(cmds) > 0 {
		cmds[len(cmds)-1].Stdout = os.Stdout
	}
	for _, cmd := range cmds {
		cmd.Start()
	}
	for _, cmd := range cmds {
		err := cmd.Wait()
		if err != nil {
			if cmd.ProcessState.ExitCode() == -1 {
				fmt.Fprintf(os.Stdout, "%s: Command not found", input)
			}
		}
	}
}

func cd(args []string) {
	if len(args) != 1 {
		fmt.Fprintf(os.Stdout, "Invalid number of Arguments.\n") // add a help
		return
	} else {
		p := path.Clean(args[0])
		if p == "~" {
			p = os.Getenv("HOME")
		}
		if !path.IsAbs(p) {
			dir, err := os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stdout, "Error: ") //handle error message
			}
			p = path.Join(dir, p)
		}
		if err := os.Chdir(p); err != nil {
			fmt.Fprintf(os.Stdout, "%s: No such file or directory\n", p)
		}
	}
}

func pwd(args []string) {
	working_dir, err := os.Getwd()
	if err == nil {
		fmt.Fprintf(os.Stdout, working_dir+"\n")
	}
}

func echo(args []string) {
	mess := strings.Join(args, " ")
	fmt.Fprintf(os.Stdout, "%s\n", mess)
}

func exit(args []string) {
	if len(args) == 0 {
		os.Exit(1)
	}
	if code, err := strconv.Atoi(args[0]); err == nil {
		os.Exit(code)
	}
}

func history(h *Hist) {
	for _, hc := range h.user_cmd {
		if hc != "history" {
			fmt.Printf("- %s\n", hc)
		}
	}
}

func typecmd(args []string) {
	if len(args) == 0 {
		fmt.Printf("")
	}
	if _, res := builtin[args[0]]; res {
		fmt.Fprintf(os.Stdout, "%s is a shell builtin\n", args[0])
		return
	}
	path_env := os.Getenv("PATH")
	paths := strings.Split(path_env, ":")
	for _, path := range paths {
		fpath := filepath.Join(path, args[0])
		if _, err := os.Stat(fpath); err == nil {
			fmt.Println(fpath)
			return
		}
	}
	fmt.Fprintf(os.Stdout, "%s: not found\n", args[0])
}

func commandPath(name string) (string, error) {
	if _, err := os.Stat(name); err == nil {
		return name, nil
	}
	curr_dir, _ := os.Getwd()
	prog_path := filepath.Join(curr_dir, name)
	if _, err := os.Stat(prog_path); err == nil {
		return prog_path, nil
	}
	paths := strings.Split(os.Getenv("PATH"), ":")
	for _, path := range paths {
		prog_path = filepath.Join(path, name)
		if _, err := os.Stat(prog_path); err == nil {
			return prog_path, nil
		}
	}
	return "", errors.New("not found")
}
