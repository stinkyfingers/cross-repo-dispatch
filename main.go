package main

import "github.com/sethvargo/go-githubactions"

func main() {
	owner := githubactions.GetInput("owner")
	if owner == "" {
		githubactions.Fatalf("missing input 'owner'")
	}
	githubactions.AddMask(owner)
	a := githubactions.New()
	a.Setoutput("status", "BEST")
	a.Setoutput("response", "RESP")
}
