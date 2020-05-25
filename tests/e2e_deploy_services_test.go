// +build e2e

package tests

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/docker/ecs-plugin/pkg/amazon"
	"github.com/docker/ecs-plugin/pkg/docker"
	"gotest.tools/assert"
	"gotest.tools/v3/fs"
	"gotest.tools/v3/golden"
	"gotest.tools/v3/icmd"
)

const (
	composeFileName = "compose.yaml"
)

func TestE2eDeployServices(t *testing.T) {
	cmd, cleanup, awsContext := dockerCli.createTestCmd()
	defer cleanup()

	composeUpSimpleService(t, cmd, awsContext)
}

func composeUpSimpleService(t *testing.T, cmd icmd.Cmd, awsContext docker.AwsContext) {
	bgContext := context.Background()
	composeYAML := golden.Get(t, "input/simple-single-service.yaml")
	tmpDir := fs.NewDir(t, t.Name(),
		fs.WithFile(composeFileName, "", fs.WithBytes(composeYAML)),
	)
	// We can't use the file added in the tmp directory because it will drop if an assertion fails
	defer composeDown(t, cmd, golden.Path("input/simple-single-service.yaml"))
	defer tmpDir.Remove()

	cmd.Command = dockerCli.Command("ecs", "compose", "--file="+tmpDir.Join(composeFileName), "--project-name", t.Name(), "up")
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	session, err := session.NewSessionWithOptions(session.Options{
		Profile: awsContext.Profile,
		Config: aws.Config{
			Region: aws.String(awsContext.Region),
		},
	})
	assert.NilError(t, err)
	sdk := amazon.NewAPI(session)
	arns, err := sdk.GetTasks(bgContext, t.Name(), "simple")
	assert.NilError(t, err)
	networkInterfaces, err := sdk.GetNetworkInterfaces(bgContext, t.Name(), arns...)
	publicIps, err := sdk.GetPublicIPs(context.Background(), networkInterfaces...)
	assert.NilError(t, err)
	for _, ip := range publicIps {
		icmd.RunCommand("curl", "-I", "http://"+ip).Assert(t, icmd.Success)
	}

}

func composeDown(t *testing.T, cmd icmd.Cmd, composeFile string) {
	cmd.Command = dockerCli.Command("ecs", "compose", "--file="+composeFile, "--project-name", t.Name(), "down")
	icmd.RunCmd(cmd).Assert(t, icmd.Success)
}