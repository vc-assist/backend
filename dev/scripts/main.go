package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/bitfield/script"
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

var scriptMap = map[string]func(){
	"dev:apply_db_schema": migrateDb,
}

func migrateDb() {
	script.Exec(strings.Join(
		[]string{
			"atlas schema apply",
			"-u 'sqlite://cmd/powerschool_api/state.db'",
			"--to 'file://cmd/powerschool_api/db/schema.sql'",
			"--dev-url 'sqlite://dev?mode=memory'",
		},
		" ",
	)).
		Stdout()
}
