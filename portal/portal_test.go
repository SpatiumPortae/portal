//nolint:errcheck
package portal_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/SpatiumPortae/portal/portal"
	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type rendezvousContainer struct {
	testcontainers.Container
	URI string
}

func TestE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test...")
	}
	ctx := context.Background()
	rendezvousC, err := setupRendezvous(ctx)
	if err != nil {
		t.Fatalf("unable to setup rendezvous server: %s", err)
	}
	config := portal.Config{
		RendezvousAddr: rendezvousC.URI,
	}
	t.Cleanup(func() {
		if err := rendezvousC.Terminate(ctx); err != nil {
			t.Fatal(err)
		}
	})
	oracle := "A frog walks into a bank..."

	in := bytes.NewBufferString(oracle)
	out := &bytes.Buffer{}

	password, err, errC := portal.Send(in, int64(in.Len()), &config)
	assert.Nil(t, err)

	err = portal.Receive(out, password, &config)
	assert.Nil(t, err)
	assert.Nil(t, <-errC)
	assert.Equal(t, oracle, out.String())
}

func setupRendezvous(ctx context.Context) (*rendezvousContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        "rendezvous:latest", // FIXME: ideally we want to run from dockerfile, not from prebuilt image.
		ExposedPorts: []string{"8080/tcp"},
		WaitingFor: wait.ForHTTP("/ping").WithPort(nat.Port("8080/tcp")).WithStatusCodeMatcher(
			func(status int) bool { return status == http.StatusOK }),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})

	if err != nil {
		return nil, err
	}
	ip, err := container.Host(ctx)
	if err != nil {
		return nil, err
	}
	mappedPort, err := container.MappedPort(ctx, "8080")
	if err != nil {
		return nil, err
	}
	uri := fmt.Sprintf("%s:%d", ip, mappedPort.Int())

	return &rendezvousContainer{Container: container, URI: uri}, nil
}
