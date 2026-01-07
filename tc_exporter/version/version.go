package version

import "runtime"

var (
	Version   = "1.0.0"
	Revision  string
	Branch    string
	GoVersion = runtime.Version()
)
