package stacks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	portainer "github.com/portainer/portainer/api"
	"github.com/portainer/portainer/api/datastore"
	"github.com/portainer/portainer/api/docker/consts"
	gittypes "github.com/portainer/portainer/api/git/types"
	"github.com/portainer/portainer/api/internal/testhelpers"
	"github.com/portainer/portainer/api/stacks/stackutils"

	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testStackUpdateToken = "test-stack-update-token"

const containerImageCompose = `# Shaped like the yammy stack: several services and a YAML anchor.
x-logging: &default-logging
  driver: json-file
  options:
    max-size: "10m"
    max-file: "3"

services:
  yammy-redirect-service:
    image: solidexpert/yammy-redirect-service.net:8
    logging: *default-logging
  yammy-site:
    image: solidexpert/service.yammy-site:34
    logging: *default-logging
  yammy-ai.service:
    image: solidexpert/yammy-service.net:34
    logging: *default-logging
  redis:
    image: redis:6-alpine
    logging: *default-logging
`

func TestContainerImage_Unauthorized(t *testing.T) {
	t.Setenv(stackUpdateTokenEnv, testStackUpdateToken)
	h := NewHandler(testhelpers.NewTestRequestBouncer(), nil)
	runtime := &fakeContainerRuntime{}
	h.containerRuntime = runtime

	tests := []struct {
		name   string
		header string
		body   map[string]string
	}{
		{
			name: "missing token",
			body: map[string]string{"container": "yammy-site", "version": "35"},
		},
		{
			name:   "wrong header",
			header: "attacker-token",
			body:   map[string]string{"token": testStackUpdateToken, "container": "yammy-site", "version": "35"},
		},
		{
			name: "wrong body token",
			body: map[string]string{"token": "attacker-token", "container": "yammy-site", "version": "35"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := postContainerImage(t, h, tt.header, tt.body)
			assert.Equal(t, http.StatusUnauthorized, rec.Code)
			assert.Contains(t, rec.Body.String(), "Invalid stack update token")
			assert.NotContains(t, rec.Body.String(), testStackUpdateToken)
			assert.NotContains(t, rec.Body.String(), "attacker-token")
			assert.Empty(t, runtime.listed)
			assert.Empty(t, runtime.pulls)
		})
	}
}

func TestContainerImage_TokenUnset(t *testing.T) {
	t.Setenv(stackUpdateTokenEnv, "")
	h := NewHandler(testhelpers.NewTestRequestBouncer(), nil)
	runtime := &fakeContainerRuntime{}
	h.containerRuntime = runtime

	rec := postContainerImage(t, h, testStackUpdateToken, map[string]string{
		"container": "yammy-site",
		"version":   "35",
	})

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Contains(t, rec.Body.String(), "Stack update token is not configured")
	assert.NotContains(t, rec.Body.String(), testStackUpdateToken)
	assert.Empty(t, runtime.listed)
	assert.Empty(t, runtime.pulls)
}

func TestContainerImage_NotFound(t *testing.T) {
	h, _, files, compose, runtime := newContainerImageHarness(t, nil, nil)

	rec := postContainerImage(t, h, testStackUpdateToken, map[string]string{
		"container": "yammy-site",
		"version":   "35",
	})

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "No container matches")
	assert.Empty(t, files.readRoot)
	assert.Zero(t, files.writes)
	assert.Empty(t, runtime.pulls)
	assert.Empty(t, compose.calls)
}

func TestContainerImage_Ambiguous(t *testing.T) {
	first := yammyContainer("solidexpert/service.yammy-site:34")
	second := yammyContainer("solidexpert/service.yammy-site:34")
	second.ID = "def"
	second.Names = []string{"/yammy-yammy-site-2"}
	h, _, files, compose, runtime := newContainerImageHarness(t, nil, []container.Summary{first, second})

	rec := postContainerImage(t, h, testStackUpdateToken, map[string]string{
		"container": "yammy-site",
		"version":   "35",
	})

	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Contains(t, rec.Body.String(), "More than one container matches")
	assert.Empty(t, files.readRoot)
	assert.Zero(t, files.writes)
	assert.Empty(t, runtime.pulls)
	assert.Empty(t, compose.calls)
}

func TestContainerImage_GitBacked(t *testing.T) {
	containers := []container.Summary{yammyContainer("solidexpert/service.yammy-site:34")}
	tests := []struct {
		name  string
		stack *portainer.Stack
	}{
		{
			name:  "git config",
			stack: &portainer.Stack{GitConfig: &gittypes.RepoConfig{URL: "https://example.com/yammy.git"}},
		},
		{
			name:  "workflow",
			stack: &portainer.Stack{WorkflowID: 4},
		},
		{
			name:  "auto update",
			stack: &portainer.Stack{AutoUpdate: &portainer.AutoUpdateSettings{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _, files, compose, runtime := newContainerImageHarness(t, tt.stack, containers)

			rec := postContainerImage(t, h, testStackUpdateToken, map[string]string{
				"container": "yammy-yammy-site-1",
				"version":   "35",
			})

			assert.Equal(t, http.StatusConflict, rec.Code)
			assert.Contains(t, rec.Body.String(), "Git-backed stack")
			assert.Empty(t, files.readRoot)
			assert.Zero(t, files.writes)
			assert.Empty(t, runtime.pulls)
			assert.Empty(t, compose.calls)
		})
	}
}

func TestContainerImage_Deploying(t *testing.T) {
	h, _, files, compose, runtime := newContainerImageHarness(t, &portainer.Stack{
		Status: portainer.StackStatusDeploying,
	}, []container.Summary{yammyContainer("solidexpert/service.yammy-site:34")})

	rec := postContainerImage(t, h, testStackUpdateToken, map[string]string{
		"container": "yammy-site",
		"version":   "35",
	})

	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Contains(t, rec.Body.String(), "Stack deployment is already in progress")
	assert.Empty(t, files.readRoot)
	assert.Zero(t, files.writes)
	assert.Empty(t, runtime.pulls)
	assert.Empty(t, compose.calls)
}

func TestContainerImage_SwarmStack(t *testing.T) {
	h, _, files, compose, runtime := newContainerImageHarness(t, &portainer.Stack{
		Type: portainer.DockerSwarmStack,
	}, []container.Summary{yammyContainer("solidexpert/service.yammy-site:34")})

	rec := postContainerImage(t, h, testStackUpdateToken, map[string]string{
		"container": "yammy-site",
		"version":   "35",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Only a compose stack can be updated")
	assert.Empty(t, files.readRoot)
	assert.Zero(t, files.writes)
	assert.Empty(t, runtime.pulls)
	assert.Empty(t, compose.calls)
}

func TestContainerImage_BadTag(t *testing.T) {
	h, _, files, compose, runtime := newContainerImageHarness(t, nil, []container.Summary{
		yammyContainer("solidexpert/service.yammy-site:34"),
	})

	for _, version := range []string{"foo/bar", "foo:bar", "foo bar", " 35"} {
		rec := postContainerImage(t, h, testStackUpdateToken, map[string]string{
			"container": "yammy-site",
			"version":   version,
		})
		assert.Equal(t, http.StatusBadRequest, rec.Code, version)
		assert.Contains(t, rec.Body.String(), "Invalid image tag")
	}

	assert.Zero(t, files.writes)
	assert.Equal(t, containerImageCompose, string(files.content))
	assert.Empty(t, runtime.pulls)
	assert.Empty(t, compose.calls)
}

func TestContainerImage_ServiceNotInFile(t *testing.T) {
	ctr := yammyContainer("solidexpert/service.yammy-site:34")
	ctr.Labels[composeServiceLabel] = "not-in-file"
	h, _, files, compose, runtime := newContainerImageHarness(t, nil, []container.Summary{ctr})

	rec := postContainerImage(t, h, testStackUpdateToken, map[string]string{
		"container": "yammy-yammy-site-1",
		"version":   "35",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Compose service is not in the stack file")
	assert.Zero(t, files.writes)
	assert.Equal(t, containerImageCompose, string(files.content))
	assert.Empty(t, runtime.pulls)
	assert.Empty(t, compose.calls)
}

func TestContainerImage_RefusedDigestDoesNotWrite(t *testing.T) {
	h, _, files, compose, runtime := newContainerImageHarness(t, nil, []container.Summary{
		yammyContainer("solidexpert/service.yammy-site:34"),
	})
	files.content = []byte("services:\n  yammy-site:\n    image: solidexpert/service.yammy-site:34@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n")

	rec := postContainerImage(t, h, testStackUpdateToken, map[string]string{
		"container": "yammy-site",
		"version":   "35",
	})

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Refused to patch the service image")
	assert.Zero(t, files.writes)
	assert.Contains(t, string(files.content), "@sha256:")
	assert.Empty(t, compose.calls)
	require.Len(t, runtime.pulls, 1)
	assert.Equal(t, "solidexpert/service.yammy-site:35", runtime.pulls[0].ref)
}

func TestContainerImage_PullFailure(t *testing.T) {
	h, _, files, compose, runtime := newContainerImageHarness(t, nil, []container.Summary{
		yammyContainer("solidexpert/service.yammy-site:34"),
	})
	runtime.pullErr = errors.New("manifest unknown")

	rec := postContainerImage(t, h, testStackUpdateToken, map[string]string{
		"container": "yammy-site",
		"version":   "35",
	})

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), "Failed to pull image")
	assert.NotContains(t, rec.Body.String(), testStackUpdateToken)
	assert.Zero(t, files.writes)
	assert.Equal(t, containerImageCompose, string(files.content))
	assert.Empty(t, compose.calls)
	require.Len(t, runtime.pulls, 1)
	assert.Equal(t, "solidexpert/service.yammy-site:35", runtime.pulls[0].ref)
	assert.Equal(t, portainer.EndpointID(1), runtime.pulls[0].endpointID)
}

func TestContainerImage_UpFailureRestoresFile(t *testing.T) {
	h, _, files, compose, runtime := newContainerImageHarness(t, nil, []container.Summary{
		yammyContainer("solidexpert/service.yammy-site:34"),
	})
	compose.err = errors.New("compose up failed")

	rec := postContainerImage(t, h, testStackUpdateToken, map[string]string{
		"container": "yammy-site",
		"version":   "35",
	})

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Failed to update the container")
	assert.Equal(t, containerImageCompose, string(files.content))
	assert.Equal(t, 2, files.writes)
	assert.Equal(t, "1", files.storedID)
	assert.Equal(t, "docker-compose.yml", files.storedName)
	require.Len(t, compose.calls, 1)
	assert.Equal(t, []string{"yammy-site"}, compose.calls[0].Services)
	assert.False(t, compose.calls[0].ForceRecreate)
	assert.False(t, compose.calls[0].Prune)
	require.Len(t, runtime.pulls, 1)
}

func TestContainerImage_Unchanged(t *testing.T) {
	h, _, files, compose, runtime := newContainerImageHarness(t, nil, []container.Summary{
		yammyContainer("solidexpert/service.yammy-site:34"),
	})

	rec := postContainerImage(t, h, "", map[string]string{
		"token":     testStackUpdateToken,
		"container": "yammy-site",
		"version":   "34",
	})

	require.Equal(t, http.StatusOK, rec.Code)
	var got containerImageResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, containerImageStatusUnchanged, got.Status)
	assert.Equal(t, portainer.EndpointID(1), got.EndpointID)
	assert.Equal(t, portainer.StackID(1), got.StackID)
	assert.Equal(t, "yammy", got.StackName)
	assert.Equal(t, "yammy-site", got.Service)
	assert.Equal(t, "yammy-yammy-site-1", got.Container)
	assert.Equal(t, "solidexpert/service.yammy-site:34", got.Image)
	assert.Equal(t, "34", got.PreviousTag)
	assert.NotContains(t, rec.Body.String(), testStackUpdateToken)
	assert.Zero(t, files.writes)
	assert.Empty(t, runtime.pulls)
	assert.Empty(t, compose.calls)
}

func TestContainerImage_FileTagAlreadySetRecreatesService(t *testing.T) {
	h, _, files, compose, runtime := newContainerImageHarness(t, nil, []container.Summary{
		yammyContainer("solidexpert/service.yammy-site:33"),
	})

	rec := postContainerImage(t, h, testStackUpdateToken, map[string]string{
		"container": "yammy-yammy-site-1",
		"version":   "34",
	})

	require.Equal(t, http.StatusOK, rec.Code)
	var got containerImageResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, containerImageStatusUpdated, got.Status)
	assert.Equal(t, "34", got.PreviousTag)
	assert.Equal(t, "solidexpert/service.yammy-site:34", got.Image)
	assert.Zero(t, files.writes)
	assert.Equal(t, containerImageCompose, string(files.content))
	require.Len(t, runtime.pulls, 1)
	assert.Equal(t, "solidexpert/service.yammy-site:34", runtime.pulls[0].ref)
	require.Len(t, compose.calls, 1)
	assert.Equal(t, []string{"yammy-site"}, compose.calls[0].Services)
	assert.False(t, compose.calls[0].ForceRecreate)
	assert.False(t, compose.calls[0].Prune)
}

func TestContainerImage_UpdatesOneService(t *testing.T) {
	h, store, files, compose, runtime := newContainerImageHarness(t, nil, []container.Summary{
		yammyContainer("solidexpert/service.yammy-site:34"),
	})
	require.NoError(t, store.Endpoint().Create(&portainer.Endpoint{
		ID:   2,
		Name: "kubernetes",
		Type: portainer.KubernetesLocalEnvironment,
		URL:  "https://k8s.example",
	}))

	rec := postContainerImage(t, h, testStackUpdateToken, map[string]string{
		"container": "yammy-yammy-site-1",
		"version":   "35",
	})

	require.Equal(t, http.StatusOK, rec.Code)
	var got containerImageResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, containerImageStatusUpdated, got.Status)
	assert.Equal(t, portainer.EndpointID(1), got.EndpointID)
	assert.Equal(t, portainer.StackID(1), got.StackID)
	assert.Equal(t, "yammy", got.StackName)
	assert.Equal(t, "yammy-site", got.Service)
	assert.Equal(t, "yammy-yammy-site-1", got.Container)
	assert.Equal(t, "solidexpert/service.yammy-site:35", got.Image)
	assert.Equal(t, "34", got.PreviousTag)
	assert.NotContains(t, rec.Body.String(), testStackUpdateToken)

	expected, err := stackutils.PatchServiceImageTag([]byte(containerImageCompose), "yammy-site", "35")
	require.NoError(t, err)
	assert.Equal(t, string(expected), string(files.content))
	assert.Contains(t, string(files.content), "logging: *default-logging")
	assert.Contains(t, string(files.content), "image: solidexpert/yammy-redirect-service.net:8")
	assert.Equal(t, 1, files.writes)
	assert.Equal(t, "/data/compose/1", files.readRoot)
	assert.Equal(t, "docker-compose.yml", files.readName)
	assert.Equal(t, "1", files.storedID)
	assert.Equal(t, "docker-compose.yml", files.storedName)

	require.Len(t, runtime.pulls, 1)
	assert.Equal(t, portainer.EndpointID(1), runtime.pulls[0].endpointID)
	assert.Equal(t, "tcp://192.168.3.114:2375", runtime.pulls[0].url)
	assert.Equal(t, "solidexpert/service.yammy-site:35", runtime.pulls[0].ref)
	assert.NotContains(t, runtime.listed, portainer.EndpointID(2))

	require.Len(t, compose.calls, 1)
	assert.Equal(t, []string{"yammy-site"}, compose.calls[0].Services)
	assert.False(t, compose.calls[0].ForceRecreate)
	assert.False(t, compose.calls[0].Prune)
	require.NotNil(t, compose.endpoint)
	assert.Equal(t, portainer.EndpointID(1), compose.endpoint.ID)
	require.NotNil(t, compose.stack)
	assert.Equal(t, "yammy", compose.stack.Name)
}

func newContainerImageHarness(t *testing.T, stack *portainer.Stack, containers []container.Summary) (*Handler, *datastore.Store, *fakeStackFiles, *fakeComposeUp, *fakeContainerRuntime) {
	t.Helper()
	t.Setenv(stackUpdateTokenEnv, testStackUpdateToken)

	if stack == nil {
		stack = &portainer.Stack{}
	}
	if stack.ID == 0 {
		stack.ID = 1
	}
	if stack.EndpointID == 0 {
		stack.EndpointID = 1
	}
	if stack.Name == "" {
		stack.Name = "yammy"
	}
	if stack.Type == 0 {
		stack.Type = portainer.DockerComposeStack
	}
	if stack.Status == 0 {
		stack.Status = portainer.StackStatusActive
	}
	if stack.EntryPoint == "" {
		stack.EntryPoint = "docker-compose.yml"
	}
	if stack.ProjectPath == "" {
		stack.ProjectPath = "/data/compose/1"
	}

	_, store := datastore.MustNewTestStore(t, true, false)
	require.NoError(t, store.Endpoint().Create(&portainer.Endpoint{
		ID:   stack.EndpointID,
		Name: "114 yammy",
		Type: portainer.DockerEnvironment,
		URL:  "tcp://192.168.3.114:2375",
	}))
	require.NoError(t, store.Stack().Create(stack))

	files := &fakeStackFiles{content: []byte(containerImageCompose)}
	compose := &fakeComposeUp{}
	runtime := &fakeContainerRuntime{
		byEndpoint: map[portainer.EndpointID][]container.Summary{
			stack.EndpointID: containers,
		},
	}

	h := NewHandler(testhelpers.NewTestRequestBouncer(), nil)
	h.DataStore = store
	h.FileService = files
	h.ComposeStackManager = compose
	h.containerRuntime = runtime
	return h, store, files, compose, runtime
}

func postContainerImage(t *testing.T, h *Handler, header string, body map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/stacks/container-image", bytes.NewReader(raw))
	if header != "" {
		req.Header.Set(stackUpdateTokenHeader, header)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func yammyContainer(imageRef string) container.Summary {
	return container.Summary{
		ID:    "abc",
		Names: []string{"/yammy-yammy-site-1"},
		Image: imageRef,
		Labels: map[string]string{
			consts.ComposeStackNameLabel: "yammy",
			composeServiceLabel:          "yammy-site",
		},
	}
}

type recordedPull struct {
	endpointID portainer.EndpointID
	url        string
	ref        string
}

type fakeContainerRuntime struct {
	byEndpoint map[portainer.EndpointID][]container.Summary
	pullErr    error
	listed     []portainer.EndpointID
	pulls      []recordedPull
}

func (f *fakeContainerRuntime) ListContainers(_ context.Context, endpoint *portainer.Endpoint) ([]container.Summary, error) {
	f.listed = append(f.listed, endpoint.ID)
	if f.byEndpoint == nil {
		return nil, nil
	}
	return f.byEndpoint[endpoint.ID], nil
}

func (f *fakeContainerRuntime) PullImage(_ context.Context, endpoint *portainer.Endpoint, ref string) error {
	f.pulls = append(f.pulls, recordedPull{endpointID: endpoint.ID, url: endpoint.URL, ref: ref})
	return f.pullErr
}

type fakeStackFiles struct {
	portainer.FileService
	content    []byte
	writes     int
	readRoot   string
	readName   string
	storedID   string
	storedName string
}

func (f *fakeStackFiles) GetFileContent(root, name string) ([]byte, error) {
	f.readRoot = root
	f.readName = name
	return append([]byte(nil), f.content...), nil
}

func (f *fakeStackFiles) StoreStackFileFromBytes(stackIdentifier, fileName string, data []byte) (string, error) {
	f.writes++
	f.storedID = stackIdentifier
	f.storedName = fileName
	f.content = append([]byte(nil), data...)
	return "/data/compose/" + stackIdentifier, nil
}

type fakeComposeUp struct {
	portainer.ComposeStackManager
	calls    []portainer.ComposeUpOptions
	stack    *portainer.Stack
	endpoint *portainer.Endpoint
	err      error
}

func (f *fakeComposeUp) Up(_ context.Context, stack *portainer.Stack, endpoint *portainer.Endpoint, options portainer.ComposeUpOptions) error {
	f.calls = append(f.calls, options)
	f.stack = stack
	f.endpoint = endpoint
	return f.err
}
