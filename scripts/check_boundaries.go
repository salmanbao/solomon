package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type violation struct {
	File   string
	Line   int
	Import string
	Rule   string
}

func main() {
	violations := collectViolations("contexts")
	if len(violations) == 0 {
		fmt.Println("boundary checks passed")
		return
	}

	sort.Slice(violations, func(i, j int) bool {
		if violations[i].File == violations[j].File {
			if violations[i].Line == violations[j].Line {
				return violations[i].Import < violations[j].Import
			}
			return violations[i].Line < violations[j].Line
		}
		return violations[i].File < violations[j].File
	})

	fmt.Println("boundary violations found:")
	for _, v := range violations {
		fmt.Printf("- %s:%d imports %q (%s)\n", v.File, v.Line, v.Import, v.Rule)
	}
	os.Exit(1)
}

func collectViolations(root string) []violation {
	var violations []violation

	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		normalized := filepath.ToSlash(path)
		parts := strings.Split(normalized, "/")
		if len(parts) < 4 || parts[0] != "contexts" {
			return nil
		}

		contextName := parts[1]
		serviceName := parts[2]
		layer := parts[3]
		modulePrefix := fmt.Sprintf("solomon/contexts/%s/%s", contextName, serviceName)

		fileViolations := validateFile(path, normalized, layer, modulePrefix)
		violations = append(violations, fileViolations...)
		return nil
	})

	return violations
}

func validateFile(path string, normalizedPath string, layer string, modulePrefix string) []violation {
	var violations []violation

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		return append(violations, violation{
			File: normalizedPath,
			Line: 1,
			Rule: "file must parse",
		})
	}

	for _, imp := range file.Imports {
		importPath := strings.Trim(imp.Path.Value, "\"")
		line := fset.Position(imp.Pos()).Line

		if strings.HasPrefix(importPath, "solomon/contexts/") && !hasPrefix(importPath, modulePrefix) {
			violations = append(violations, violation{
				File:   normalizedPath,
				Line:   line,
				Import: importPath,
				Rule:   "cross-module imports are forbidden",
			})
		}

		switch layer {
		case "domain":
			violations = append(violations, validateDomainImport(normalizedPath, line, importPath, modulePrefix)...)
		case "application":
			violations = append(violations, validateApplicationImport(normalizedPath, line, importPath, modulePrefix)...)
		}
	}

	return violations
}

func validateDomainImport(file string, line int, importPath string, modulePrefix string) []violation {
	var violations []violation

	if strings.Contains(importPath, "/adapters/") {
		violations = append(violations, violation{
			File:   file,
			Line:   line,
			Import: importPath,
			Rule:   "domain must not import adapters",
		})
	}

	if strings.HasPrefix(importPath, "solomon/internal/") ||
		strings.HasPrefix(importPath, "solomon/integrations/") ||
		strings.HasPrefix(importPath, "solomon/platform/") {
		violations = append(violations, violation{
			File:   file,
			Line:   line,
			Import: importPath,
			Rule:   "domain must not import runtime infrastructure",
		})
	}

	allowed := []string{
		modulePrefix + "/domain",
	}
	if !isStdlib(importPath) && !isAllowed(importPath, allowed) {
		violations = append(violations, violation{
			File:   file,
			Line:   line,
			Import: importPath,
			Rule:   "domain import is outside explicit allowlist",
		})
	}

	return violations
}

func validateApplicationImport(file string, line int, importPath string, modulePrefix string) []violation {
	var violations []violation

	if strings.Contains(importPath, "/adapters/") {
		violations = append(violations, violation{
			File:   file,
			Line:   line,
			Import: importPath,
			Rule:   "application must not import adapters",
		})
	}

	if strings.HasPrefix(importPath, "solomon/internal/") ||
		strings.HasPrefix(importPath, "solomon/integrations/") ||
		strings.HasPrefix(importPath, "solomon/platform/") {
		violations = append(violations, violation{
			File:   file,
			Line:   line,
			Import: importPath,
			Rule:   "application must not import runtime infrastructure",
		})
	}

	allowed := []string{
		modulePrefix + "/application",
		modulePrefix + "/domain",
		modulePrefix + "/ports",
		"solomon/contracts",
	}
	if !isStdlib(importPath) && !isAllowed(importPath, allowed) {
		violations = append(violations, violation{
			File:   file,
			Line:   line,
			Import: importPath,
			Rule:   "application import is outside explicit allowlist",
		})
	}

	return violations
}

func hasPrefix(path string, prefix string) bool {
	return path == prefix || strings.HasPrefix(path, prefix+"/")
}

func isAllowed(importPath string, allowedPrefixes []string) bool {
	for _, p := range allowedPrefixes {
		if hasPrefix(importPath, p) {
			return true
		}
	}
	return false
}

func isStdlib(importPath string) bool {
	if strings.HasPrefix(importPath, "solomon/") {
		return false
	}
	first := importPath
	if idx := strings.Index(first, "/"); idx != -1 {
		first = first[:idx]
	}
	return !strings.Contains(first, ".")
}
