package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
)

func printScripts() {
	fmt.Println("Scripts:")
	for key := range scriptMap {
		fmt.Println("\t" + key)
	}
}

func main() {
	flag.Parse()

	script := flag.Arg(0)
	fn, ok := scriptMap[script]
	if !ok {
		fmt.Printf(
			"you must specify a valid script, '%s' is not a valid script.\n",
			script,
		)
		printScripts()
		os.Exit(1)
	}

	fn()
}

func cmd(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fullCmd := name
	for _, a := range args {
		fullCmd += " "
		fullCmd += a
	}

	fmt.Printf("$ %s\n", fullCmd)
	err := cmd.Run()
	if err != nil {
		os.Exit(1)
	}
}

var scriptMap = map[string]func(){
	"dev:apply_db_schema": migrateDb,
}

func migrateDb() {
	cmd(
		"atlas", "schema", "apply",
		"-u", "sqlite://cmd/powerschool_api/state.db",
		"--to", "file://cmd/powerschool_api/db/schema.sql",
		"--dev-url", "sqlite://dev?mode=memory",
	)
}
