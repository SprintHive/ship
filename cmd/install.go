// Copyright © 2017 SprintHive (Pty) Ltd (buzz@sprinthive.com)
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
	"os"

	"github.com/SprintHive/ship/pkg/helm"
	"github.com/SprintHive/ship/pkg/kubectl"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// installCmd represents the create command
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Installs the SHIP components into your Kubernetes cluster",
	Long: `Install a bundle of SHIP components into your Kubernetes cluster using helm.
	
	The following components will be installed:
	* Ingress GW (Kong)
	* Ingress Controller (Kong Controller)
	* Ingress GW Database (Cassandra)
	* Logging database (Elasticsearch)
	* Log collector (Fluent-bit)
	* Tracing (Zipkin)
	* Metric database (Prometheus)
	* Metric Visualization (Grafana)
	* Log Viewer (Kibana)
	* CI/CD (Jenkins)
	* Artifact repository (Nexus)`,
	Run: func(cmd *cobra.Command, args []string) {
		domain, err := cmd.Flags().GetString("domain")
		if err != nil {
			fmt.Fprintf(os.Stderr, fmt.Sprintf("Failed to get domain flag"))
			os.Exit(1)
		}
		var components []ShipComponent
		viper.UnmarshalKey("components", &components)

		helm.InstallChartRepo()
		installComponents(&components, domain)
	},
}

func installComponents(components *[]ShipComponent, domain string) {
	releasesToSkip := make(map[string]struct{})
	currentReleases := helm.GetHelmReleases()
	for _, release := range currentReleases {
		releasesToSkip[release] = struct{}{}
	}

	errors := []error{}

	for _, component := range *components {
		if _, found := releasesToSkip[component.Chart.ReleaseName]; found {
			fmt.Printf("Skipping installation of already installed component: %s\n", component.Chart.ChartPath)
		} else {
			for _, preInstallSpec := range component.PreInstallResources {
				if preInstallSpec.PreconditionReady.Resource != (KubernetesResource{}) {
					if err := waitForResourceReady(&preInstallSpec.PreconditionReady.Resource, preInstallSpec.PreconditionReady.MinReplicas); err != nil {
						fmt.Printf("Error encountered: %v\n", err)
						errors = append(errors, err)
						continue
					}
				}

				// TODO: Fix hardcoded infra namespace
				kubectl.Create(preInstallSpec.ManifestPath, "infra")

				if preInstallSpec.WaitForDone != (KubernetesResource{}) {
					if err := waitForResourceCompleted(&preInstallSpec.WaitForDone); err != nil {
						fmt.Printf("Error encountered: %v\n", err)
						errors = append(errors, err)
						continue
					}
				}

				if !preInstallSpec.PersistentAfterWait {
					kubectl.Delete(preInstallSpec.ManifestPath, "infra")
				}
			}
			helm.InstallChart(&component.Chart, domain)

			for _, postInstallSpec := range component.PostInstallResources {
				if postInstallSpec.PreconditionReady.Resource != (KubernetesResource{}) {
					if err := waitForResourceReady(&postInstallSpec.PreconditionReady.Resource, postInstallSpec.PreconditionReady.MinReplicas); err != nil {
						fmt.Printf("Error encountered: %v\n", err)
						errors = append(errors, err)
						continue
					}
				}

				// TODO: Fix hardcoded infra namespace
				kubectl.Create(postInstallSpec.ManifestPath, "infra")

				if postInstallSpec.WaitForDone != (KubernetesResource{}) {
					if err := waitForResourceCompleted(&postInstallSpec.WaitForDone); err != nil {
						fmt.Printf("Error encountered: %v\n", err)
						errors = append(errors, err)
						continue
					}
				}

				if !postInstallSpec.PersistentAfterWait {
					kubectl.Delete(postInstallSpec.ManifestPath, "infra")
				}
			}
		}
	}

	if len(errors) == 0 {
		fmt.Println("Installation was successful!")
	} else {
		fmt.Println("Installation completed with errors:")
		for _, componentError := range errors {
			fmt.Println(componentError)
		}
	}
}

func waitForResourceReady(kubeResource *KubernetesResource, minReplicas int) error {
	if kubeResource.Type == "deployment" {
		kubectl.WaitDeployReady(kubeResource.Name, minReplicas, kubeResource.Namespace)
	} else if kubeResource.Type == "daemonset" {
		kubectl.WaitDaemonSetReady(kubeResource.Name, minReplicas, kubeResource.Namespace)
	} else if kubeResource.Type == "statefulset" {
		kubectl.WaitStatefulSetReady(kubeResource.Name, minReplicas, kubeResource.Namespace)
	} else {
		return fmt.Errorf("unsupported wait precondition type: %s", kubeResource.Type)
	}

	return nil
}

func waitForResourceCompleted(kubeResource *KubernetesResource) error {
	if kubeResource.Type == "pod" {
		kubectl.WaitPodCompleted(kubeResource.Name, kubeResource.Namespace)
	} else {
		fmt.Fprintf(os.Stderr, fmt.Sprintf("Unsupported wait type: %s\n", kubeResource.Type))
		return fmt.Errorf("unsupported wait resource type: %s", kubeResource.Type)
	}

	return nil
}

func init() {
	RootCmd.AddCommand(installCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// installCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// installCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	installCmd.Flags().StringP("domain", "d", "", "Sets the base domain that will be used for ingress. *.<base domain> should resolve to your Kubernetes cluster.")
	installCmd.MarkFlagRequired("domain")
}
