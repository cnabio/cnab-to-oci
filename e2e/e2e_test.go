package e2e

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"text/template"

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

	// Load invocation image from archive
	invocationImageName := registry + "/e2e/hello-world:0.1.0-invoc"
	serviceImageName := registry + "/e2e/http-echo"
	runCmd(t, icmd.Command("docker", "load", "--input", filepath.Join("testdata", "hello-world", "hello-world:0.1.0-invoc.tar")))
	runCmd(t, icmd.Command("docker", "tag", "hello-world:0.1.0-invoc", invocationImageName))

	// Fetch service image
	runCmd(t, icmd.Command("docker", "pull", "hashicorp/http-echo"))
	runCmd(t, icmd.Command("docker", "tag", "hashicorp/http-echo", serviceImageName))

	// Tidy up my room
	defer func() {
		runCmd(t, icmd.Command("docker", "image", "rm", "-f", "hello-world:0.1.0-invoc", invocationImageName, "hashicorp/http-echo", serviceImageName))
	}()

	// Push the images to the registry
	runCmd(t, icmd.Command("docker", "push", invocationImageName))
	runCmd(t, icmd.Command("docker", "push", serviceImageName))

	// Templatize the bundle
	tmpl, err := template.ParseFiles(filepath.Join("testdata", "hello-world", "bundle.json.template"))
	assert.NilError(t, err)
	data := struct {
		InvocationImage string
		ServiceImage    string
	}{
		invocationImageName,
		serviceImageName,
	}
	f, err := os.Create(dir.Join("bundle.json"))
	assert.NilError(t, err)
	defer f.Close()
	err = tmpl.Execute(f, data)
	assert.NilError(t, err)

	// Save the fixed bundle
	runCmd(t, icmd.Command("cnab-to-oci", "fixup", dir.Join("bundle.json"),
		"--target", registry+"/myuser",
		"--insecure-registries", registry,
		"--output", dir.Join("fixed-bundle.json")))

	// Re fix-up, checking it works twice
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
