/*
Copyright 2021 Flant JSC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

var variablesTemplate = `// Code generated by "tools/audit_policy.go" DO NOT EDIT.
package hooks

var auditPolicyBasicNamespaces = []string{
{{- range $value := .Namespace }}
	"{{ $value }}",
{{- end }}
}
var auditPolicyBasicServiceAccounts = []string{
{{- range $value := .ServiceAccount }}
	"{{ $value }}",
{{- end }}
}
`

func cwd() string {
	_, f, _, ok := runtime.Caller(1)
	if !ok {
		panic("cannot get caller")
	}

	dir, err := filepath.Abs(f)
	if err != nil {
		panic(err)
	}

	for i := 0; i < 3; i++ { // ../../
		dir = filepath.Dir(dir)
	}

	// If deckhouse repo directory is symlinked (e.g. to /deckhouse), resolve the real path.
	// Otherwise, filepath.Walk will ignore all subdirectories.
	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		panic(err)
	}

	return dir
}

func walkModules(namespaces, sas *[]string, workDir string) error {
	chartNames := make(map[string]string)
	saNames := make(map[string][]string)

	err := filepath.Walk(workDir, func(path string, f os.FileInfo, err error) error {
		if f != nil && f.IsDir() {
			return nil
		}

		modulePath := filepath.Dir(strings.TrimPrefix(path, workDir))
		// In case of files inside `templates` directory we want only module path
		modulePath = strings.Split(modulePath, "templates")[0]
		modulePath = strings.TrimRight(modulePath, "/")

		if filepath.Base(path) == "Chart.yaml" {
			c, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			var chart map[string]interface{}

			if err := yaml.Unmarshal(c, &chart); err != nil {
				return err
			}

			if name, ok := chart["name"]; ok {
				chartNames[modulePath] = name.(string)
			}

			return nil
		}

		if filepath.Base(path) == ".namespace" {
			ns, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			namespace := strings.Trim(string(ns), "\r\n")

			if !strings.HasPrefix(namespace, "d8-") && namespace != "kube-system" {
				return nil
			}

			*namespaces = append(*namespaces, namespace)
		}

		if filepath.Base(path) == "rbac-for-us.yaml" {
			rbac, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			r := regexp.MustCompile(`apiVersion: v1
kind: ServiceAccount
metadata:
\s*name:\s*(.*)
`)
			match := r.FindAllStringSubmatch(string(rbac), -1)
			for _, m := range match {
				// ServiceAccount could be in double or single quotes
				// Unquote it for versatility
				saName, err := unquote(m[1])
				if err != nil {
					return err
				}

				saNames[modulePath] = append(saNames[modulePath], saName)
			}
		}

		return nil
	})

	for modulePath, saNames := range saNames {
		for _, sa := range saNames {
			chartName := ""
			if name, ok := chartNames[modulePath]; ok {
				chartName = name
			}
			sa = strings.Replace(sa, "{{ .Chart.Name }}", chartName, 1)
			sa = strings.Replace(sa, "{{ $.Chart.Name }}", chartName, 1)
			if len(sa) == 0 && len(chartName) == 0 {
				return fmt.Errorf("empty final SA name, seems chartName didnt resolve for module: %s", modulePath)
			}
			*sas = append(*sas, sa)
		}
	}

	return err
}

func unquote(s string) (string, error) {
	var err error
	if strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) || strings.HasPrefix(s, `'`) && strings.HasSuffix(s, `'`) {
		s, err = strconv.Unquote(s)
		if err != nil {
			return "", err
		}
	}

	return s, err
}

func main() {
	workDir := cwd()

	var (
		output string
		stream = os.Stdout
	)
	flag.StringVar(&output, "output", "", "output file for generated code")
	flag.Parse()

	if output != "" {
		var err error
		stream, err = os.Create(output)
		if err != nil {
			panic(err)
		}

		defer stream.Close()
	}

	var namespaces []string
	var sas []string
	if err := walkModules(&namespaces, &sas, workDir); err != nil {
		panic(err)
	}

	sort.Strings(namespaces)
	sort.Strings(sas)

	t := template.New("variables")
	t, err := t.Parse(variablesTemplate)
	if err != nil {
		panic(err)
	}

	type Data struct {
		Namespace, ServiceAccount []string
	}
	data := Data{
		Namespace:      uniqueNonEmptyElementsOf(namespaces),
		ServiceAccount: uniqueNonEmptyElementsOf(sas),
	}
	err = t.Execute(stream, data)
	if err != nil {
		panic(err)
	}
}

func uniqueNonEmptyElementsOf(s []string) []string {
	unique := make(map[string]bool, len(s))
	us := make([]string, len(unique))
	for _, elem := range s {
		if len(elem) != 0 {
			if !unique[elem] {
				us = append(us, elem)
				unique[elem] = true
			}
		}
	}

	return us
}
