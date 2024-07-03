package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

type (
	fn func([]string)
)

var builtin = make(map[string]fn)

func main() {
	initcmd()
	sigchnl := make(chan os.Signal, 1)
	signal.Notify(sigchnl)

	for {
		fmt.Fprint(os.Stdout, "$ ")
		go func() {
			for {
				sig := <-sigchnl
				handler(sig)
			}
		}()
		input, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			os.Exit(1)
		}
		input = strings.TrimRight(input, "\n")
		cmds := strings.Split(strings.TrimSpace(input), "|")
		if len(cmds) != 1 {
			pipedCommand(input)
		} else {
			inputs := strings.Split(strings.TrimSpace(input), " ")
			args := inputs[1:]
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

func handler(signal os.Signal) {
	if signal == syscall.SIGINT {
		fmt.Println("\nType Exit to quit the shell")
	}
}

func initcmd() {
	builtin["echo"] = echo
	builtin["exit"] = exit
	builtin["type"] = typecmd
	builtin["pwd"] = pwd
	builtin["cd"] = cd
}

func pipedCommand(input string) {
	input = strings.TrimSpace(input)
	program := exec.Command("bash", "-c", input)
	program.Stdout = os.Stdout
	program.Stderr = os.Stderr
	if err := program.Run(); err != nil {
		fmt.Fprintf(os.Stdout, "%s: command not found\n", input)
	}
}

func cd(args []string) {
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
