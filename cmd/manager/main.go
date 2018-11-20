package main

import "github.com/jaegertracing/jaeger-operator/cmd"

func main() {
	// Note that this file should be identical to the main.go at the root of the project
	// It would really be nice if this one here wouldn't be required, but the Operator SDK
	// requires it...
	// https://github.com/operator-framework/operator-sdk/blob/master/doc/migration/v0.1.0-migration-guide.md#copy-changes-from-maingo
	// > operator-sdk now expects cmd/manager/main.go to be present in Go operator projects.
	// > Go project-specific commands, ex. add [api, controller], will error if main.go is not found in its expected path.
	cmd.Execute()
}
