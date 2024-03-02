package curling

const (
	lineContinuationDefault    = "\\"
	lineContinuationWindows    = "^"
	lineContinuationPowerShell = "`"
)

type Option func(curling *Command)

func WithFollowRedirects() Option {
	return func(curling *Command) {
		curling.location = true
	}
}

func WithCompression() Option {
	return func(curling *Command) {
		curling.compressed = true
	}
}

func WithInsecure() Option {
	return func(curling *Command) {
		curling.insecure = true
	}
}

func WithLongForm() Option {
	return func(curling *Command) {
		curling.useLongForm = true
	}
}

func WithSilent() Option {
	return func(curling *Command) {
		curling.silent = true
	}
}

func WithMultiLine() Option {
	return func(curling *Command) {
		curling.useMultiLine = true
		curling.lineContinuation = lineContinuationDefault
	}
}

func WithWindowsMultiLine() Option {
	return func(curling *Command) {
		curling.useMultiLine = true
		curling.lineContinuation = lineContinuationWindows
	}
}

func WithPowerShellMultiLine() Option {
	return func(curling *Command) {
		curling.useMultiLine = true
		curling.lineContinuation = lineContinuationPowerShell
	}
}
