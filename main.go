/*
Copyright © 2021 Sentry

*/
package main

import (
	"math/rand"
	"time"

	"github.com/getsentry/go-load-tester/cmd"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	cmd.Execute()
}
