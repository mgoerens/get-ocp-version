package main

import "embed"
// import "errors"
import "fmt"
import "os"


import "github.com/spf13/cobra"
import "github.com/Masterminds/semver"
import modSemver "golang.org/x/mod/semver"
import "github.com/Masterminds/sprig/v3"
import "gopkg.in/yaml.v3"

//go:embed kubeOpenShiftVersionMap.yaml
var content embed.FS

var kubeOpenShiftVersionMap map[string]string
var latestKubeVersion *semver.Version

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

	latestKubeVersion, _ = semver.NewVersion("0.0")
	for _, versionMap := range versions.Versions {
		currentVersion, _ := semver.NewVersion(versionMap.KubeVersion)
		if currentVersion.GreaterThan(latestKubeVersion) {
			latestKubeVersion = currentVersion
		}
		kubeOpenShiftVersionMap[versionMap.KubeVersion] = versionMap.OcpVersion
	}
}

// func GetOCPRange(kubeVersionRange string) error {
// 	fmt.Println("enter get OCP range")
// 	fmt.Println("provided Kubernetes Range: " + kubeVersionRange)

// 	err := errors.New("error in getOCP Range")
// 	return err
// }

const KuberVersionProcessingError  = "Error converting kubeVersion to an OCP range"

func GetOCPRange(kubeVersionRange string) (string, error) {
	semverCompare := sprig.GenericFuncMap()["semverCompare"].(func(string, string) (bool, error))
	minOCPVersion := ""
	maxOCPVersion := ""
	for kubeVersion, OCPVersion := range kubeOpenShiftVersionMap {
		match, err := semverCompare(kubeVersionRange, kubeVersion)
		if err != nil {
			return "", fmt.Errorf("%s : %s", KuberVersionProcessingError, err)
		}
		if match {
			testOCPVersion := fmt.Sprintf("v%s", OCPVersion)
			if minOCPVersion == "" || modSemver.Compare(testOCPVersion, fmt.Sprintf("v%s", minOCPVersion)) < 0 {
				minOCPVersion = OCPVersion
			}
			if maxOCPVersion == "" || modSemver.Compare(testOCPVersion, fmt.Sprintf("v%s", maxOCPVersion)) > 0 {
				maxOCPVersion = OCPVersion
			}
		}
	}
	// Check if min ocp range is open ended, for example 1.* or >-=1.20
	// To do this see if 1.999 is valid for the min OCP version range, not perfect but works until kubernetes hits 2.0.
	if minOCPVersion != "" {
		match, _ := semverCompare(kubeVersionRange, "1.999")
		if match {
			return fmt.Sprintf(">=%s", minOCPVersion), nil
		} else {
			if minOCPVersion == maxOCPVersion {
				return minOCPVersion, nil
			} else {
				return fmt.Sprintf("%s - %s", minOCPVersion, maxOCPVersion), nil
			}
		}
	}

	return "", fmt.Errorf("%s : Failed to determine a minimum OCP version", KuberVersionProcessingError)
}


var rootCmd = &cobra.Command{
    Use:  "get-ocp-version",
    Short: "get-ocp-version",
    Long: `get-ocp-version`,
    RunE: func(cmd *cobra.Command, args []string) error {
		resultOCPRange, err := GetOCPRange(kubeVersionRange)
		if err != nil {
			return err
		}
		fmt.Println(resultOCPRange)
		// if err := GetOCPRange(); err != nil {
		// 	return err
		// }
		return nil
    },
}

var kubeVersionRange string

func main() {
	// fmt.Println("Hello, world.")
	// fmt.Println(kubeOpenShiftVersionMap)
	rootCmd.PersistentFlags().StringVar(&kubeVersionRange, "kubeVersionRange", "", "Range of Kubernetes versions)")
    if err := rootCmd.Execute(); err != nil {
        // fmt.Fprintf(os.Stderr, "Error in getting OCP range '%s'", err)
        os.Exit(1)
    }
}
