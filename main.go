package main

import "github.com/may1a/bad-vibes/cmd"

var version = "dev"

func main() {
	cmd.SetVersion(version)
	cmd.Execute()
}
