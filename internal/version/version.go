package version

import (
	"fmt"
	"strconv"
	"strings"
)

// License is valid for same major version (1.x.x works with 1.y.z, but not 2.x.x)
func IsCompatible(licenseVersion, requestedVersion string) (bool, error) {
	licenseMajor, err := ExtractMajorVersion(licenseVersion)
	if err != nil {
		return false, fmt.Errorf("invalid license version: %v", err)
	}

	requestedMajor, err := ExtractMajorVersion(requestedVersion)
	if err != nil {
		return false, fmt.Errorf("invalid app version: %v", err)
	}

	return licenseMajor == requestedMajor, nil
}

func ExtractMajorVersion(version string) (int, error) {
	if version == "" {
		return 0, fmt.Errorf("empty version string")
	}

	parts := strings.Split(version, ".")
	if len(parts) < 1 {
		return 0, fmt.Errorf("invalid version format")
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid major version: %v", err)
	}

	if major < 0 {
		return 0, fmt.Errorf("major version cannot be negative")
	}

	return major, nil
}
