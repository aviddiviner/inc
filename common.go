package main

import (
	"github.com/aviddiviner/inc/file"
	"sort"
)

// De-dupe, clean and sort a list of file paths.
func cleanPaths(paths []string) (out []string) {
	uniq := make(map[string]bool)
	for _, p := range paths {
		uniq[file.CleanPath(p)] = true
	}
	out = make([]string, len(uniq))
	i := 0
	for key := range uniq {
		out[i] = key
		i += 1
	}
	sort.Strings(out)
	return
}

// Scan for files, ensuring we always exclude the config file (contains keys!)
func scanFiles(paths LocalConfigPaths, opt options) *file.PathScanner {
	var incl, excl []string

	// Use the backup paths given on the command line in preference to config.
	if len(opt.includePaths) > 0 {
		incl = opt.includePaths
	} else {
		incl = paths.Include
	}

	// Exclude all paths given on command line as well as in config.
	excl = append(paths.Exclude, opt.excludePaths...)

	scanner := file.NewScanner()
	for _, dir := range cleanPaths(incl) {
		scanner.IncludePath(dir)
	}
	for _, dir := range cleanPaths(excl) {
		scanner.ExcludePath(dir)
	}
	scanner.ExcludePath(opt.configPath)

	return scanner
}
