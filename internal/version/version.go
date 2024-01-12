// Copyright 2021 Chaos Mesh Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package version

import (
	"fmt"
	"runtime"
)

var (
	GitUrl    string
	GitBranch string
	GitCommit string
	BuildDate string
)

// PrintVersionInfo show version info to Stdout
func PrintVersionInfo(name string) {
	fmt.Printf("%s %v", name, Get().String())
}

// Get returns the overall codebase version. It's for detecting
// what code a binary was built from.
func Get() Info {
	// These variables typically come from -ldflags settings
	return Info{
		GitUrl:    GitUrl,
		GitBranch: GitBranch,
		GitCommit: GitCommit,
		GoVersion: runtime.Version(),
		Compiler:  runtime.Compiler,
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		BuildDate: BuildDate,
	}
}
