package migrations

import "strings"

var allowedMigrateActions = map[string]struct{}{
	"up":     {},
	"down":   {},
	"status": {},
	"script": {},
}

func ParseMigrateCommand(args []string) (string, bool) {
	cmd, action, extra := parseCommandArgs(args)

	if cmd != "migrate" || extra {
		return "", false
	}

	if _, ok := allowedMigrateActions[action]; !ok {
		return "", false
	}

	return action, true
}

func parseCommandArgs(args []string) (cmd, action string, extra bool) {
	for i := 0; i < len(args); i++ {
		arg := args[i]

		if arg == "--" {
			return firstTwoPositional(args[i+1:])
		}

		if isFlag(arg) {
			if shouldSkipFlagValue(arg, args, i) {
				i++
			}
			continue
		}

		if cmd == "" {
			cmd = arg
			continue
		}

		if action == "" {
			action = arg
			continue
		}

		extra = true
	}

	return cmd, action, extra
}

func firstTwoPositional(args []string) (cmd, action string, extra bool) {
	for _, arg := range args {
		if cmd == "" {
			cmd = arg
			continue
		}

		if action == "" {
			action = arg
			continue
		}

		extra = true
	}

	return cmd, action, extra
}

func isFlag(arg string) bool {
	return strings.HasPrefix(arg, "-")
}

func shouldSkipFlagValue(arg string, args []string, index int) bool {
	return !strings.Contains(arg, "=") &&
		index+1 < len(args) &&
		!isFlag(args[index+1])
}
