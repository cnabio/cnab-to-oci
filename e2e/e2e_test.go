package e2e

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"text/template"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"
	"gotest.tools/v3/golden"
	"gotest.tools/v3/icmd"
)

func TestPushAndPullCNAB(t *testing.T) {
	dir := fs.NewDir(t, t.Name())
	defer dir.Remove()
	r := startRegistry(t)
	defer r.Stop(t)
	registry := r.GetAddress(t)

	invocationImageName := registry + "/e2e/hello-world:0.1.0-invoc"
	serviceImageName := registry + "/e2e/http-echo"
	appImageName := registry + "/myuser"

	// Build invocation image
	cmd := icmd.Command("docker", "build", "-f", filepath.Join("testdata", "hello-world", "invocation-image", "Dockerfile"),
		"-t", invocationImageName, filepath.Join("testdata", "hello-world", "invocation-image"))
	cmd.Env = append(os.Environ(), "DOCKER_BUILDKIT=1")
	runCmd(t, cmd)

	// Fetch service image
	runCmd(t, icmd.Command("docker", "pull", "hashicorp/http-echo"))
	runCmd(t, icmd.Command("docker", "tag", "hashicorp/http-echo", serviceImageName))

	// Tidy up my room
	defer func() {
		runCmd(t, icmd.Command("docker", "image", "rm", "-f", invocationImageName, "hashicorp/http-echo", serviceImageName))
	}()

	// Push the images to the registry
	output := runCmd(t, icmd.Command("docker", "push", invocationImageName))
	invocDigest := getDigest(t, output)

	runCmd(t, icmd.Command("docker", "push", serviceImageName))

	// Templatize the bundle
	applyTemplate(t, serviceImageName, invocationImageName, invocDigest, filepath.Join("testdata", "hello-world", "bundle.json.template"), dir.Join("bundle.json"))

	// Save the fixed bundle
	runCmd(t, icmd.Command("cnab-to-oci", "fixup", dir.Join("bundle.json"),
		"--target", appImageName,
		"--insecure-registries", registry,
		"--bundle", dir.Join("fixed-bundle.json"),
		"--relocation-map", dir.Join("relocation.json"),
		"--auto-update-bundle"))

	// Check the fixed bundle
	applyTemplate(t, serviceImageName, invocationImageName, invocDigest, filepath.Join("testdata", "bundle.json.golden.template"), filepath.Join("testdata", "bundle.json.golden"))
	buf, err := ioutil.ReadFile(dir.Join("fixed-bundle.json"))
	assert.NilError(t, err)
	golden.Assert(t, string(buf), "bundle.json.golden")

	// Check the relocation map
	checkRelocationMap(t, serviceImageName, invocationImageName, appImageName, dir.Join("relocation.json"))

	// Re fix-up, checking it works twice
	runCmd(t, icmd.Command("cnab-to-oci", "fixup", dir.Join("bundle.json"),
		"--target", appImageName,
		"--insecure-registries", registry,
		"--bundle", dir.Join("fixed-bundle.json"),
		"--auto-update-bundle"))

	// Push the CNAB to the registry and get the digest
	out := runCmd(t, icmd.Command("cnab-to-oci", "push", dir.Join("bundle.json"),
		"--target", appImageName,
		"--insecure-registries", registry,
		"--auto-update-bundle"))
	re := regexp.MustCompile(`"(.*)"`)
	digest := re.FindAllStringSubmatch(out, -1)[0][1]

	// Pull the CNAB from the registry
	runCmd(t, icmd.Command("cnab-to-oci", "pull", fmt.Sprintf("%s@%s", appImageName, digest),
		"--bundle", dir.Join("pulled-bundle.json"),
		"--relocation-map", dir.Join("pulled-relocation.json"),
		"--insecure-registries", registry))
	pulledBundle, err := ioutil.ReadFile(dir.Join("pulled-bundle.json"))
	assert.NilError(t, err)
	pulledRelocation, err := ioutil.ReadFile(dir.Join("pulled-relocation.json"))
	assert.NilError(t, err)

	// Check the fixed bundle.json is equal to the pulled bundle.json
	golden.Assert(t, string(pulledBundle), dir.Join("fixed-bundle.json"))
	golden.Assert(t, string(pulledRelocation), dir.Join("relocation.json"))
}

func runCmd(t *testing.T, cmd icmd.Cmd) string {
	fmt.Println("#", strings.Join(cmd.Command, " "))
	result := icmd.RunCmd(cmd)
	fmt.Println(result.Combined())
	result.Assert(t, icmd.Success)
	return result.Stdout()
}

func applyTemplate(t *testing.T, serviceImageName, invocationImageName, invocationDigest, templateFile, resultFile string) {
	tmpl, err := template.ParseFiles(templateFile)
	assert.NilError(t, err)
	data := struct {
		InvocationImage  string
		InvocationDigest string
		ServiceImage     string
	}{
		invocationImageName,
		invocationDigest,
		serviceImageName,
	}
	f, err := os.Create(resultFile)
	assert.NilError(t, err)
	defer f.Close()
	err = tmpl.Execute(f, data)
	assert.NilError(t, err)
}

func checkRelocationMap(t *testing.T, serviceImageName, invocationImageName, appImageName, relocationMapFile string) {
	data, err := ioutil.ReadFile(relocationMapFile)
	assert.NilError(t, err)
	relocationMap := map[string]string{}
	err = json.Unmarshal(data, &relocationMap)
	assert.NilError(t, err)

	// Check the relocated images are in the app repository
	assert.Assert(t, strings.HasPrefix(relocationMap[serviceImageName], appImageName))
	assert.Assert(t, strings.HasPrefix(relocationMap[invocationImageName], appImageName))
}

func getDigest(t *testing.T, output string) string {
	re := regexp.MustCompile(`digest: (.+) size:`)
	result := re.FindStringSubmatch(output)
	assert.Equal(t, len(result), 2)
	digest := result[1]
	assert.Assert(t, digest != "")
	return digest
}
