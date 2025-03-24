package java

import (
	"strconv"

	"github.com/unbindapp/railpack/core/generate"
)

const DEFAULT_JDK_VERSION = "21"

func (p *JavaProvider) setJDKVersion(ctx *generate.GenerateContext, miseStep *generate.MiseStepBuilder) {
	jdk := miseStep.Default("java", DEFAULT_JDK_VERSION)
	if jdkVersion, envName := ctx.Env.GetConfigVariable("JDK_VERSION"); jdkVersion != "" {
		miseStep.Version(jdk, jdkVersion, envName)
	}

	if p.usesGradle(ctx) {
		gradleVersion, err := strconv.Atoi(miseStep.Resolver.Get("gradle").Version)
		if err == nil && gradleVersion <= 5 {
			miseStep.Version(jdk, "8", "Gradle <= 5")
		}
	}
}
