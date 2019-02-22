package e2e

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
	"testing"

	"gotest.tools/assert"
	"gotest.tools/fs"
	"gotest.tools/golden"
	"gotest.tools/icmd"
)

func TestPushAndPullCNAB(t *testing.T) {
	dir := fs.NewDir(t, t.Name())
	defer dir.Remove()
	r := startRegistry(t)
	defer r.Stop(t)
	registry := r.GetAddress(t)

	// Create a CNAB bundle from a Docker Application Package
	runCmd(t, icmd.Command("docker-app", "bundle", "/examples/hello-world/hello-world.dockerapp",
		"--invocation-image", "hello-world:0.1.0-invoc",
		"--namespace", registry+"/e2e",
		"--out", dir.Join("bundle.json")))

	// Push the invocation image to the registry
	runCmd(t, icmd.Command("docker", "push", registry+"/e2e/hello-world:0.1.0-invoc"))

	// Save the fixed bundle
	runCmd(t, icmd.Command("cnab-to-oci", "fixup", dir.Join("bundle.json"),
		"--target", registry+"/myuser",
		"--insecure-registries", registry,
		"--output", dir.Join("fixed-bundle.json")))

	// Push the CNAB to the registry and get the digest
	out := runCmd(t, icmd.Command("cnab-to-oci", "push", dir.Join("bundle.json"),
		"--target", registry+"/myuser",
		"--insecure-registries", registry))
	re := regexp.MustCompile(`"(.*)"`)
	digest := re.FindAllStringSubmatch(out, -1)[0][1]

	// Pull the CNAB from the registry
	runCmd(t, icmd.Command("cnab-to-oci", "pull", registry+"/myuser@"+digest,
		"--output", dir.Join("pulled-bundle.json"),
		"--insecure-registries", registry))
	pulledBundle, err := ioutil.ReadFile(dir.Join("pulled-bundle.json"))
	assert.NilError(t, err)

	// Check the fixed bundle.json is equal to the pulled bundle.json
	golden.Assert(t, string(pulledBundle), dir.Join("fixed-bundle.json"))
}

func runCmd(t *testing.T, cmd icmd.Cmd) string {
	fmt.Println("#", strings.Join(cmd.Command, " "))
	result := icmd.RunCmd(cmd)
	fmt.Println(result.Combined())
	result.Assert(t, icmd.Success)
	return result.Stdout()
}
