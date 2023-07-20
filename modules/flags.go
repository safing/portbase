package modules

import (
	"flag"
	"fmt"
)

var (
	// HelpFlag triggers printing flag.Usage. It's exported for custom help handling.
	HelpFlag       bool
	printGraphFlag bool
)

func init() {
	flag.BoolVar(&HelpFlag, "help", false, "print help")
	flag.BoolVar(&printGraphFlag, "print-module-graph", false, "print the module dependency graph")
}

func parseFlags() error {
	// parse flags
	if !flag.Parsed() {
		flag.Parse()
	}

	if HelpFlag {
		flag.Usage()
		return ErrCleanExit
	}

	if printGraphFlag {
		printGraph()
		return ErrCleanExit
	}

	return nil
}

func printGraph() {
	// mark roots
	for _, module := range modules {
		if len(module.depReverse) == 0 {
			// is root, dont print deps in dep tree
			module.stopFlag.Set()
		}
	}
	// print
	for _, module := range modules {
		if module.stopFlag.IsSet() {
			// print from root
			printModuleGraph("", module, true)
		}
	}
}

func printModuleGraph(prefix string, module *Module, root bool) {
	fmt.Printf("%s├── %s\n", prefix, module.Name)
	if root || !module.stopFlag.IsSet() {
		for _, dep := range module.Dependencies() {
			printModuleGraph(fmt.Sprintf("│   %s", prefix), dep, false)
		}
	}
}
