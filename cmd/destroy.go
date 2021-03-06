// Copyright © 2017 NAME HERE <EMAIL ADDRESS>
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// destroyCmd represents the destroy command
var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Removes components installed by ship",
	Long:  `This will remove all helm releases with release names that match the release names used by the ship installation`,
	Run: func(cmd *cobra.Command, args []string) {
		var charts []HelmChart
		viper.UnmarshalKey("charts", &charts)

		getHelmReleases()
		removeReleases(&charts)
	},
}

func removeReleases(sourceCharts *[]HelmChart) {
	currentReleases := getHelmReleases()
	currentReleasesMap := make(map[string]struct{})
	for _, currentRelease := range currentReleases {
		currentReleasesMap[currentRelease] = struct{}{}
	}

	for _, sourceChart := range *sourceCharts {
		if _, found := currentReleasesMap[sourceChart.ReleaseName]; found {
			fmt.Println(fmt.Sprintf("Removing release: %v", sourceChart.ReleaseName))
			removeHelmRelease(sourceChart.ReleaseName)
		}
	}
}

func removeHelmRelease(releaseName string) {
	cmdName := "helm"
	args := []string{"delete", "--purge", releaseName}

	if output, err := exec.Command(cmdName, args...).CombinedOutput(); err != nil {
		panic(fmt.Sprintf("Failed to remove helm release '%s': %v", releaseName, string(output)))
	}
}

func getHelmReleases() []string {
	cmdName := "helm"

	args := []string{"list", "-q"}

	output, err := exec.Command(cmdName, args...).CombinedOutput()
	if err != nil {
		panic(fmt.Sprintf("Failed to remove charts: %v", string(output)))
	}

	releases := strings.Split(strings.Trim(string(output), "\" "), "\n")
	// The last line is always empty, so pop it
	releases = releases[:len(releases)-1]

	return releases
}

func init() {
	RootCmd.AddCommand(destroyCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// destroyCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// destroyCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
