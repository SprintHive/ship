package helm

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var (
	cmdName = "helm"
)

// ValueOverride describes a helm override
type ValueOverride struct {
	Override string
	Type     string
}

// Chart contains the information needed to install a helm chart
type Chart struct {
	ChartPath   string
	Namespace   string
	ReleaseName string
	Overrides   []ValueOverride
	ValuesPath  string
}

// GetHelmReleases returns the list of helm releases in the kubernetes cluster in the active profile
func GetHelmReleases() []string {
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

// RemoveReleases removes the provided releases if they are present
func RemoveReleases(releases []string) {
	currentReleases := GetHelmReleases()
	currentReleasesMap := make(map[string]struct{})
	for _, currentRelease := range currentReleases {
		currentReleasesMap[currentRelease] = struct{}{}
	}

	for _, release := range releases {
		if _, found := currentReleasesMap[release]; found {
			fmt.Println(fmt.Sprintf("Removing release: %v", release))
			removeHelmRelease(release)
		}
	}
}

// InstallChartRepo installs a given helm repository into the repository config
func InstallChartRepo() {
	args := []string{"repo", "add", "sprinthive-dev-charts", "https://s3.eu-west-2.amazonaws.com/sprinthive-dev-charts"}

	if output, err := exec.Command(cmdName, args...).CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, fmt.Sprintf("Failed to install sprinthive charts: %v", string(output)))
		os.Exit(1)
	}

	fmt.Println("Successfully installed sprinthive chart repo")
}

// InstallChart will install the provided chart into the currently configured Kubernetes cluster
func InstallChart(chart *Chart, domain string) {
	fmt.Printf("Installing chart: %s\n", chart.ChartPath)
	args := []string{"install", chart.ChartPath, "-n", chart.ReleaseName, "--namespace", chart.Namespace}

	for _, valueOverride := range chart.Overrides {
		var helmFlag string
		if valueOverride.Type == "string" {
			helmFlag = "--set-string"
		} else {
			helmFlag = "--set"
		}
		args = append(args, helmFlag, strings.Replace(valueOverride.Override, "${domain}", domain, -1))
	}

	if chart.ValuesPath != "" {
		args = append(args, "--values", chart.ValuesPath)
	}

	if output, err := exec.Command(cmdName, args...).CombinedOutput(); err != nil {
		panic(fmt.Sprintf("Failed to install chart: %v", string(output)))
	}
}

func removeHelmRelease(releaseName string) {
	args := []string{"delete", "--purge", releaseName}

	if output, err := exec.Command(cmdName, args...).CombinedOutput(); err != nil {
		panic(fmt.Sprintf("Failed to remove helm release '%s': %v", releaseName, string(output)))
	}
}
