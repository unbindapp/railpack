package buildkit

import (
	"fmt"
	"runtime"
)

type BuildPlatform struct {
	OS           string
	Architecture string
	Variant      string
}

var (
	PlatformLinuxAMD64 = BuildPlatform{
		OS:           "linux",
		Architecture: "amd64",
	}
	PlatformLinuxARM64 = BuildPlatform{
		OS:           "linux",
		Architecture: "arm64",
		Variant:      "v8",
	}
)

func determineBuildPlatformFromHost() BuildPlatform {
	if runtime.GOARCH == "arm64" {
		return PlatformLinuxARM64
	}
	return PlatformLinuxAMD64
}

func (p BuildPlatform) String() string {
	return fmt.Sprintf("%s/%s", p.OS, p.Architecture)
}
