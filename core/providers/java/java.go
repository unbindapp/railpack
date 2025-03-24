package java

import (
	"fmt"

	"github.com/unbindapp/railpack/core/generate"
	"github.com/unbindapp/railpack/core/plan"
)

type JavaProvider struct{}

func (p *JavaProvider) Name() string {
	return "java"
}

func (p *JavaProvider) Detect(ctx *generate.GenerateContext) (bool, error) {
	return ctx.App.HasMatch("pom.{xml,atom,clj,groovy,rb,scala,yaml,yml}") || ctx.App.HasMatch("gradlew"), nil
}

func (p *JavaProvider) Initialize(ctx *generate.GenerateContext) error {
	return nil
}

func (p *JavaProvider) StartCommandHelp() string {
	return ""
}

func (p *JavaProvider) Plan(ctx *generate.GenerateContext) error {
	build := ctx.NewCommandStep("build")
	build.AddCommand(plan.NewCopyCommand("."))
	build.Inputs = []plan.Input{plan.NewStepInput(ctx.GetMiseStepBuilder().Name())}

	if p.usesGradle(ctx) {
		ctx.Logger.LogInfo("Using Gradle")

		p.setGradleVersion(ctx)
		p.setJDKVersion(ctx, ctx.GetMiseStepBuilder())

		if ctx.App.HasMatch("gradlew") && !ctx.App.IsFileExecutable("gradlew") {
			build.AddCommand(plan.NewExecCommand("chmod +x gradlew"))
		}

		build.AddCommand(plan.NewExecCommand("./gradlew clean build -x check -x test"))
		build.AddCache(p.gradleCache(ctx))
	} else {
		ctx.Logger.LogInfo("Using Maven")

		ctx.GetMiseStepBuilder().Default("maven", "latest")
		p.setJDKVersion(ctx, ctx.GetMiseStepBuilder())

		if ctx.App.HasMatch("mvnw") && !ctx.App.IsFileExecutable("mvnw") {
			build.AddCommand(plan.NewExecCommand("chmod +x mvnw"))
		}

		build.AddCommand(plan.NewExecCommand(fmt.Sprintf("%s -DoutputFile=target/mvn-dependency-list.log -B -DskipTests clean dependency:list install", p.getMavenExe(ctx))))
		build.AddCache(p.mavenCache(ctx))
	}

	runtimeMiseStep := ctx.NewMiseStepBuilder("packages:mise:runtime")
	p.setJDKVersion(ctx, runtimeMiseStep)

	outPath := "target/."
	if ctx.App.HasMatch("**/build/libs/*.jar") || p.usesGradle(ctx) {
		outPath = "."
	}

	ctx.Deploy.Inputs = []plan.Input{
		ctx.DefaultRuntimeInput(),
		plan.NewStepInput(runtimeMiseStep.Name(), plan.InputOptions{
			Include: runtimeMiseStep.GetOutputPaths(),
		}),
		plan.NewStepInput(build.Name(), plan.InputOptions{
			Include: []string{outPath},
		}),
	}
	ctx.Deploy.StartCmd = p.getStartCmd(ctx)

	p.addMetadata(ctx)

	return nil
}

func (p *JavaProvider) getStartCmd(ctx *generate.GenerateContext) string {
	if p.usesGradle(ctx) {
		buildGradle := p.readBuildGradle(ctx)
		return fmt.Sprintf("java $JAVA_OPTS -jar %s $(ls -1 */build/libs/*jar | grep -v plain)", getGradlePortConfig(buildGradle))
	} else if ctx.App.HasMatch("pom.xml") {
		return fmt.Sprintf("java %s $JAVA_OPTS -jar target/*jar", getMavenPortConfig(ctx))
	} else {
		return "java $JAVA_OPTS -jar target/*jar"
	}

}

func (p *JavaProvider) addMetadata(ctx *generate.GenerateContext) {
	hasGradle := p.usesGradle(ctx)

	if hasGradle {
		ctx.Metadata.Set("javaPackageManager", "gradle")
	} else {
		ctx.Metadata.Set("javaPackageManager", "maven")
	}

	var framework string
	if p.usesSpringBoot(ctx) {
		framework = "spring-boot"
	}

	ctx.Metadata.Set("javaFramework", framework)
}

func (p *JavaProvider) usesSpringBoot(ctx *generate.GenerateContext) bool {
	return ctx.App.HasMatch("**/spring-boot*.jar") ||
		ctx.App.HasMatch("**/spring-boot*.class") ||
		ctx.App.HasMatch("**/org/springframework/boot/**")
}
