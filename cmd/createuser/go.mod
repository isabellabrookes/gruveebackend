module github.com/pixelogicdev/gruveebackend/cmd/createuser

go 1.13

require (
	cloud.google.com/go/firestore v1.1.1
	github.com/pixelogicdev/gruveebackend/pkg/firebase v0.0.0-20200308213401-073e9c1ba1b9
)

// ENABLE WHEN IN DEBUG
replace github.com/pixelogicdev/gruveebackend/pkg/firebase => ../../pkg/firebase
