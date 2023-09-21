// This package creates a CLI that allows to derive a range of OCP versions from a given range of
// Kubernetes versions. It uses Mastermind/semver/v3 under the hood.

package main

import "embed"
import "fmt"
import "os"
import "strings"

import "github.com/spf13/cobra"
import "github.com/Masterminds/semver/v3"
import "gopkg.in/yaml.v3"

//go:embed kubeOpenShiftVersionMap.yaml
var content embed.FS

type versionMap struct {
	Versions []*versionMapping `yaml:"versions"`
}

type versionMapping struct {
	KubeVersion string `yaml:"kube-version"`
	OcpVersion  string `yaml:"ocp-version"`
}

// Maps a Kubernetes version to its corresponding OCP version.
var kubeOpenShiftVersionMap map[string]string

// Contains the highest known Kubernetes version. Used to determine if the provided Kubernetes
// range is open-ended.
var upperKubeVersion *semver.Version

// Contains the content of the CLI argument "kubeVersionRange".
var inputKubeVersionRange string

// Upon initialization of this package, the kubeOpenShiftVersionMap variable is populated with the
// content of kubeOpenShiftVersionMap.yaml.
func init() {
	kubeOpenShiftVersionMap = make(map[string]string)

	yamlFile, err := content.ReadFile("kubeOpenShiftVersionMap.yaml")
	if err != nil {
		panic(fmt.Sprintf("Error reading content of kubeOpenShiftVersionMap.yaml: %v", err))
	}

	versions := versionMap{}
	err = yaml.Unmarshal(yamlFile, &versions)
	if err != nil {
		panic(fmt.Sprintf("Error reading content of kubeOpenShiftVersionMap.yaml: %v", err))
	}

	upperKubeVersion, _ = semver.NewVersion("0.0")
	for _, versionMap := range versions.Versions {
		// Register the highest known Kubernetes version.
		kubeVersion, err := semver.NewVersion(versionMap.KubeVersion)
		if err != nil {
			panic(fmt.Sprintf("Error reading content of kubeOpenShiftVersionMap.yaml: %v", err))
		}
		if kubeVersion.GreaterThan(upperKubeVersion) {
			upperKubeVersion = kubeVersion
		}

		// Build Kubernetes version to OCP version mapping.
		kubeOpenShiftVersionMap[versionMap.KubeVersion] = versionMap.OcpVersion
	}
}

// GetOCPRange derives a range of OCP versions given a range of Kubernetes versions.
//
// To do so, it first ensures that the provided range of Kubernetes versions is a valid SemVer
// constraint using Mastermind/semver/v3. It then checks which of the known Kubernetes versions
// validate the Constraint, and registers the corresponding minimum and maximum OCP versions.
// Finally, it builds the resulting range of OCP versions.
//
// This function currently doesn't support the || operator for the provided range of Kubernetes
// versions.
func GetOCPRange(kubeVersionRange string) (string, error) {
	// Return an error if the provided range of Kubernetes versions contains unsupported operators.
	if strings.Contains(kubeVersionRange, "||") {
		return "", fmt.Errorf("Range %s contains unsupported operator ||", kubeVersionRange)
	}

	minOCPRange, _ := semver.NewVersion("9.9")
	maxOCPRange, _ := semver.NewVersion("0.0")

	// Ensure that the provided range of Kubernetes versions is a valid SemVer constraint.
	kubeVersionRangeConstraint, err := semver.NewConstraint(kubeVersionRange)
	if err != nil {
		return "", fmt.Errorf("Error converting %s to Constraint: %v", kubeVersionRange, err)
	}

	for kubeVersion, OCPVersion := range kubeOpenShiftVersionMap {
		// Check which of the known Kubernetes versions validate the Constraint.
		kubeVersionVersion, err := semver.NewVersion(kubeVersion)
		if err != nil {
			return "", fmt.Errorf("Error converting %s to Version: %v", kubeVersion, err)
		}
		isInRange, _ := kubeVersionRangeConstraint.Validate(kubeVersionVersion)
		if isInRange {
			// Register the corresponding minimum and maximum OCP versions.
			OCPVersionVersion, err := semver.NewVersion(OCPVersion)
			if err != nil {
				return "", fmt.Errorf("Error converting %s to Version: %v", OCPVersion, err)
			}
			if OCPVersionVersion.LessThan(minOCPRange) {
				minOCPRange = OCPVersionVersion
			}
			if OCPVersionVersion.GreaterThan(maxOCPRange) {
				maxOCPRange = OCPVersionVersion
			}
		}
	}

	// Build the resulting range of OCP versions.
	if minOCPRange.Original() == "9.9" {
		// If the minimum was never set, it means we didn't match any known Kubernetes version.
		return "", fmt.Errorf("Failed to match any known Kubernetes version to the provided range %s", kubeVersionRange)
	}
	if isRangeOpenEnded(kubeVersionRangeConstraint) {
		// If the provided range is open-ended, the result range should also be open-ended.
		return ">=" + minOCPRange.Original(), nil
	}
	if minOCPRange.Equal(maxOCPRange) {
		return minOCPRange.Original(), nil
	}
	return ">=" + minOCPRange.Original() + " <=" + maxOCPRange.Original(), nil
}

// isRangeOpenEnded returns a boolean set to True if the provided range of Kubernetes versions is
// open-ended (e.g. ">=1.13").
//
// To do so, we incremente the Patch value of the highest known Kubernetes Version, and check if it
// belongs to the range.
//
// TODO: This also returns True on ">=1.13 <=1.30" when upperKubeVersion=1.26.
func isRangeOpenEnded(kubeVersionRangeConstraint *semver.Constraints) bool {
	nextUpperKubeVersion := upperKubeVersion.IncMinor()
	isOpenEnded, _ := kubeVersionRangeConstraint.Validate(&nextUpperKubeVersion)
	return isOpenEnded
}

var rootCmd = &cobra.Command{
    Use:  "get-ocp-version",
    Short: "get-ocp-version",
    Long: `get-ocp-version derives a range of OCP versions from a given range of Kubernetes Version. It uses Mastermind/semver/v3 under the hood.`,
    RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		resultOCPRange, err := GetOCPRange(inputKubeVersionRange)
		if err != nil {
			return err
		}
		fmt.Println(resultOCPRange)
		return nil
    },
}

func main() {
	rootCmd.PersistentFlags().StringVar(&inputKubeVersionRange, "kubeVersionRange", "", "SemVer compatible range of Kubernetes versions")
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
