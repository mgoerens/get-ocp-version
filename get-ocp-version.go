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

var kubeOpenShiftVersionMap map[string]string
var upperKubeVersion *semver.Version

type versionMap struct {
	Versions []*versionMapping `yaml:"versions"`
}

type versionMapping struct {
	KubeVersion string `yaml:"kube-version"`
	OcpVersion  string `yaml:"ocp-version"`
}

func init() {
	kubeOpenShiftVersionMap = make(map[string]string)

	yamlFile, err := content.ReadFile("kubeOpenShiftVersionMap.yaml")
	if err != nil {
		fmt.Sprintf("Error reading content of kubeOpenShiftVersionMap.yaml: %v", err)
		return
	}

	versions := versionMap{}
	err = yaml.Unmarshal(yamlFile, &versions)
	if err != nil {
		fmt.Sprintf("Error reading content of kubeOpenShiftVersionMap.yaml: %v", err)
		return
	}

	upperKubeVersion, _ = semver.NewVersion("0.0")
	for _, versionMap := range versions.Versions {
		// Register then upper value of the known Kubernetes versions
		kubeVersion, _ := semver.NewVersion(versionMap.KubeVersion)
		if kubeVersion.GreaterThan(upperKubeVersion) {
			upperKubeVersion = kubeVersion
		}
		kubeOpenShiftVersionMap[versionMap.KubeVersion] = versionMap.OcpVersion
	}
}

func GetOCPRange(kubeVersionRange string) (string, error) {
	if strings.Contains(kubeVersionRange, "||") {
		return "", fmt.Errorf("Range contains unsupported constraint ||")
	}

	minOCPRange, _ := semver.NewVersion("9.9")
	maxOCPRange, _ := semver.NewVersion("0.0")

	kubeVersionRangeConstraint, err := semver.NewConstraint(kubeVersionRange)
	if err != nil {
		return "", fmt.Errorf("Error converting %s to Constraint: %s", kubeVersionRange, err)
	}

	for kubeVersion, OCPVersion := range kubeOpenShiftVersionMap {
		kubeVersionVersion, err := semver.NewVersion(kubeVersion)
		if err != nil {
			return "", fmt.Errorf("Error converting %s to Version: %s", kubeVersion, err)
		}
		isInRange, _ := kubeVersionRangeConstraint.Validate(kubeVersionVersion)
		if isInRange {
			OCPVersionVersion, err := semver.NewVersion(OCPVersion)
			if err != nil {
				return "", fmt.Errorf("Error converting %s to Version: %s", OCPVersion, err)
			}
			if OCPVersionVersion.LessThan(minOCPRange) {
				minOCPRange = OCPVersionVersion
			}
			if OCPVersionVersion.GreaterThan(maxOCPRange) {
				maxOCPRange = OCPVersionVersion
			}
		}
	}

	// kubeVersionRange as Constraint
	// For each kubeVersion in kubeOpenShiftMap
	// 		Check if kubeVersion in kubeVersionRange
	//		if Yes, register minOCP, maxOCP:
	//			if min > corresponding OCP Version
	//			if max < corresponding OCP Version
	// Done

	// Craft OCPRange from min / max
	// if min not set

	if minOCPRange.Original() == "9.9" {
		return "", fmt.Errorf("Failed to match any known Kubernetes version to the provided range %s", kubeVersionRange)
	}
	if isRangeOpenEnded(kubeVersionRangeConstraint) {
		return ">=" + minOCPRange.Original(), nil
	}
	if minOCPRange.Equal(maxOCPRange) {
		return minOCPRange.Original(), nil
	}
	return ">=" + minOCPRange.Original() + " <=" + maxOCPRange.Original(), nil
}

func isRangeOpenEnded(kubeVersionRangeConstraint *semver.Constraints) bool {
	nextUpperKubeVersion := upperKubeVersion.IncMinor()
	isOpenEnded, _ := kubeVersionRangeConstraint.Validate(&nextUpperKubeVersion)
	return isOpenEnded
}

var rootCmd = &cobra.Command{
    Use:  "get-ocp-version",
    Short: "get-ocp-version",
    Long: `get-ocp-version`,
    RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		resultOCPRange, err := GetOCPRange(kubeVersionRange)
		if err != nil {
			return err
		}
		fmt.Println(resultOCPRange)
		return nil
    },
}

var kubeVersionRange string

func main() {
	rootCmd.PersistentFlags().StringVar(&kubeVersionRange, "kubeVersionRange", "", "Range of Kubernetes versions)")
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
