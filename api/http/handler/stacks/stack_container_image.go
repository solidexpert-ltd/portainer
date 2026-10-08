package stacks

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	portainer "github.com/portainer/portainer/api"
	dockerclient "github.com/portainer/portainer/api/docker/client"
	"github.com/portainer/portainer/api/docker/consts"
	"github.com/portainer/portainer/api/internal/endpointutils"
	"github.com/portainer/portainer/api/logs"
	"github.com/portainer/portainer/api/stacks/stackutils"
	httperror "github.com/portainer/portainer/pkg/libhttp/error"
	"github.com/portainer/portainer/pkg/libhttp/request"
	"github.com/portainer/portainer/pkg/libhttp/response"

	composeloader "github.com/compose-spec/compose-go/v2/loader"
	composetypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	stackUpdateTokenHeader = "X-Stack-Update-Token"
	stackUpdateTokenEnv    = "PORTAINER_STACK_UPDATE_TOKEN"
	composeServiceLabel    = "com.docker.compose.service"

	containerImageStatusUpdated   = "updated"
	containerImageStatusUnchanged = "unchanged"
)

// stackContainerRuntime lists containers and pulls an image on one Portainer
// environment. Production uses the Docker API of that environment. Tests
// substitute a fake so the handler does not open a Docker socket.
type stackContainerRuntime interface {
	ListContainers(ctx context.Context, endpoint *portainer.Endpoint) ([]container.Summary, error)
	PullImage(ctx context.Context, endpoint *portainer.Endpoint, ref string) error
}

type dockerContainerRuntime struct {
	factory *dockerclient.ClientFactory
}

var _ stackContainerRuntime = dockerContainerRuntime{}

type containerImagePayload struct {
	Token     string `json:"token"`
	Container string `json:"container"`
	Version   string `json:"version"`
}

func (payload *containerImagePayload) Validate(r *http.Request) error {
	if strings.TrimSpace(payload.Container) == "" {
		return errors.New("container is required")
	}
	if payload.Version == "" {
		return errors.New("version is required")
	}
	return nil
}

type containerImageResponse struct {
	Status      string               `json:"status"`
	EndpointID  portainer.EndpointID `json:"endpointId"`
	StackID     portainer.StackID    `json:"stackId"`
	StackName   string               `json:"stackName"`
	Service     string               `json:"service"`
	Container   string               `json:"container"`
	Image       string               `json:"image"`
	PreviousTag string               `json:"previousTag"`
}

type containerImageTarget struct {
	endpoint  portainer.Endpoint
	container container.Summary
	stack     portainer.Stack
	service   string
}

// @id StackContainerImage
// @summary Update one compose service image and recreate that service
// @description Git or CI calls this route. It is not a Portainer JWT route.
// @description The token is the X-Stack-Update-Token header or the JSON token field.
// @description **Access policy**: public wrapper, token required by the handler
// @tags stacks
// @accept json
// @produce json
// @param body body containerImagePayload true "Container name and image tag"
// @success 200 {object} containerImageResponse "Success"
// @failure 400 "Invalid request"
// @failure 401 "Invalid token"
// @failure 404 "Container not found"
// @failure 409 "Ambiguous, git-backed, or already deploying"
// @failure 422 "Image pull failed"
// @failure 500 "Server error"
// @failure 503 "Token is not configured"
// @router /stacks/container-image [post]
func (handler *Handler) stackContainerImage(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
	payload, httpErr := readContainerImageRequest(r)
	if httpErr != nil {
		return httpErr
	}

	target, httpErr := handler.findContainerImageTarget(r.Context(), payload.Container)
	if httpErr != nil {
		return httpErr
	}

	if target.stack.Type != portainer.DockerComposeStack {
		logContainerImage(target.endpoint.ID, target.stack.ID, target.stack.Name, target.service, payload.Container, "", payload.Version, "error", errors.New("stack is not a compose stack"))
		return httperror.BadRequest("Only a compose stack can be updated", errors.New("swarm and kubernetes stacks are not supported"))
	}

	return handler.withStackImageLock(target.stack.ID, func() *httperror.HandlerError {
		return handler.applyContainerImage(w, r, payload, target)
	})
}

// readContainerImageRequest checks the token before any stack or Docker lookup.
// The header is used when it is set. A wrong header is rejected even if the
// JSON token matches. The token is never written to the log or the error text.
func readContainerImageRequest(r *http.Request) (*containerImagePayload, *httperror.HandlerError) {
	expected := os.Getenv(stackUpdateTokenEnv)
	if expected == "" {
		log.Info().Str("status", "unavailable").Msg("stack container image update")
		return nil, httperror.NewError(http.StatusServiceUnavailable, "Stack update token is not configured", errors.New("PORTAINER_STACK_UPDATE_TOKEN is unset"))
	}

	headerToken := r.Header.Get(stackUpdateTokenHeader)
	if headerToken != "" {
		if subtle.ConstantTimeCompare([]byte(headerToken), []byte(expected)) != 1 {
			log.Info().Str("status", "unauthorized").Msg("stack container image update")
			return nil, httperror.Unauthorized("Invalid stack update token", errors.New("missing or invalid stack update token"))
		}

		payload, err := request.GetPayload[containerImagePayload](r)
		if err != nil {
			return nil, httperror.BadRequest("Invalid request payload", err)
		}
		return payload, nil
	}

	if r.Body == nil {
		log.Info().Str("status", "unauthorized").Msg("stack container image update")
		return nil, httperror.Unauthorized("Invalid stack update token", errors.New("missing or invalid stack update token"))
	}
	var payload containerImagePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Info().Str("status", "unauthorized").Msg("stack container image update")
		return nil, httperror.Unauthorized("Invalid stack update token", errors.New("missing or invalid stack update token"))
	}
	if subtle.ConstantTimeCompare([]byte(payload.Token), []byte(expected)) != 1 {
		log.Info().Str("status", "unauthorized").Msg("stack container image update")
		return nil, httperror.Unauthorized("Invalid stack update token", errors.New("missing or invalid stack update token"))
	}
	if err := payload.Validate(r); err != nil {
		return nil, httperror.BadRequest("Invalid request payload", err)
	}
	return &payload, nil
}

func (handler *Handler) findContainerImageTarget(ctx context.Context, query string) (containerImageTarget, *httperror.HandlerError) {
	endpoints, err := handler.DataStore.Endpoint().Endpoints()
	if err != nil {
		return containerImageTarget{}, httperror.InternalServerError("Unable to list environments", err)
	}

	runtime := handler.containersRuntime()
	hits := make([]containerImageTarget, 0)
	var listErr error
	for i := range endpoints {
		endpoint := endpoints[i]
		if !endpointutils.IsDockerEndpoint(&endpoint) {
			continue
		}

		containers, err := runtime.ListContainers(ctx, &endpoint)
		if err != nil {
			listErr = err
			log.Error().Err(err).Int("endpointId", int(endpoint.ID)).Str("status", "error").Msg("stack container image update")
			continue
		}

		for _, ctr := range containers {
			if containerMatchesQuery(ctr, query) {
				hits = append(hits, containerImageTarget{endpoint: endpoint, container: ctr})
			}
		}
	}

	if listErr != nil {
		return containerImageTarget{}, httperror.InternalServerError("Unable to list containers on every environment", listErr)
	}
	if len(hits) == 0 {
		log.Info().Str("container", query).Str("status", "not-found").Msg("stack container image update")
		return containerImageTarget{}, httperror.NotFound("No container matches the requested name", errors.New("no container matches"))
	}
	if len(hits) > 1 {
		log.Info().Str("container", query).Int("matches", len(hits)).Str("status", "ambiguous").Msg("stack container image update")
		return containerImageTarget{}, httperror.Conflict("More than one container matches", fmt.Errorf("%d containers match", len(hits)))
	}

	hit := hits[0]
	project := hit.container.Labels[consts.ComposeStackNameLabel]
	service := hit.container.Labels[composeServiceLabel]
	if project == "" || service == "" {
		return containerImageTarget{}, httperror.BadRequest("Container is not a compose service", errors.New("compose project or service label is missing"))
	}

	stacks, err := handler.DataStore.Stack().StacksByName(project)
	if err != nil {
		return containerImageTarget{}, httperror.InternalServerError("Unable to find the stack for the container", err)
	}

	matched := make([]portainer.Stack, 0, 1)
	for _, stack := range stacks {
		if stack.EndpointID == hit.endpoint.ID {
			matched = append(matched, stack)
		}
	}
	if len(matched) == 0 {
		log.Info().Str("container", query).Str("project", project).Str("status", "not-found").Msg("stack container image update")
		return containerImageTarget{}, httperror.NotFound("No compose stack matches the container", fmt.Errorf("no stack named %q on this environment", project))
	}
	if len(matched) > 1 {
		return containerImageTarget{}, httperror.Conflict("More than one stack matches the container", fmt.Errorf("%d stacks named %q", len(matched), project))
	}

	hit.stack = matched[0]
	hit.service = service
	return hit, nil
}

func (handler *Handler) applyContainerImage(w http.ResponseWriter, r *http.Request, payload *containerImagePayload, target containerImageTarget) *httperror.HandlerError {
	stack, err := handler.DataStore.Stack().Read(target.stack.ID)
	if handler.DataStore.IsErrObjectNotFound(err) {
		return httperror.NotFound("No compose stack matches the container", err)
	} else if err != nil {
		return httperror.InternalServerError("Unable to find the stack for the container", err)
	}

	containerName := containerDisplayName(target.container, payload.Container)
	if stack.Type != portainer.DockerComposeStack {
		logContainerImage(target.endpoint.ID, stack.ID, stack.Name, target.service, containerName, "", payload.Version, "error", errors.New("stack is not a compose stack"))
		return httperror.BadRequest("Only a compose stack can be updated", errors.New("swarm and kubernetes stacks are not supported"))
	}
	if stackIsGitBacked(stack) {
		logContainerImage(target.endpoint.ID, stack.ID, stack.Name, target.service, containerName, "", payload.Version, "error", errors.New("git-backed stack"))
		return httperror.Conflict("Git-backed stack cannot be updated in place", errors.New("a later git sync would overwrite the patched stack file"))
	}
	if stack.Status == portainer.StackStatusDeploying {
		logContainerImage(target.endpoint.ID, stack.ID, stack.Name, target.service, containerName, "", payload.Version, "error", errors.New("stack deployment is already in progress"))
		return httperror.Conflict("Stack deployment is already in progress", errors.New("stack deployment is already in progress"))
	}
	if stack.EntryPoint == "" || stack.ProjectPath == "" {
		return httperror.InternalServerError("Stack file path is missing", errors.New("stack project path or entry point is empty"))
	}

	original, err := handler.FileService.GetFileContent(stack.ProjectPath, stack.EntryPoint)
	if err != nil {
		logContainerImage(target.endpoint.ID, stack.ID, stack.Name, target.service, containerName, "", payload.Version, "error", err)
		return httperror.InternalServerError("Unable to read the stack file", err)
	}
	original = bytes.Clone(original)

	patched, patchErr := stackutils.PatchServiceImageTag(original, target.service, payload.Version)
	if errors.Is(patchErr, stackutils.ErrInvalidImageTag) {
		logContainerImage(target.endpoint.ID, stack.ID, stack.Name, target.service, containerName, "", payload.Version, "error", patchErr)
		return httperror.BadRequest("Invalid image tag", patchErr)
	}
	if errors.Is(patchErr, stackutils.ErrServiceNotFound) {
		logContainerImage(target.endpoint.ID, stack.ID, stack.Name, target.service, containerName, "", payload.Version, "error", patchErr)
		return httperror.BadRequest("Compose service is not in the stack file", patchErr)
	}
	if patchErr != nil && !errors.Is(patchErr, stackutils.ErrImageTagRefused) {
		logContainerImage(target.endpoint.ID, stack.ID, stack.Name, target.service, containerName, "", payload.Version, "error", patchErr)
		return httperror.InternalServerError("Unable to patch the service image", patchErr)
	}

	currentImage, imageErr := composeServiceImage(original, target.service)
	if imageErr != nil {
		logContainerImage(target.endpoint.ID, stack.ID, stack.Name, target.service, containerName, "", payload.Version, "error", imageErr)
		return httperror.InternalServerError("Unable to read the service image", imageErr)
	}
	repo, previousTag, ok := imageRepoAndTag(currentImage)
	if !ok || repo == "" {
		err := fmt.Errorf("service %q image %q has no repository", target.service, currentImage)
		logContainerImage(target.endpoint.ID, stack.ID, stack.Name, target.service, containerName, previousTag, payload.Version, "error", err)
		return httperror.InternalServerError("Unable to read the service image", err)
	}

	imageRef := repo + ":" + payload.Version
	if patchErr == nil && previousTag == payload.Version && runningImageHasTag(target.container.Image, repo, payload.Version) {
		logContainerImage(target.endpoint.ID, stack.ID, stack.Name, target.service, containerName, previousTag, payload.Version, containerImageStatusUnchanged, nil)
		return response.JSON(w, &containerImageResponse{
			Status:      containerImageStatusUnchanged,
			EndpointID:  target.endpoint.ID,
			StackID:     stack.ID,
			StackName:   stack.Name,
			Service:     target.service,
			Container:   containerName,
			Image:       imageRef,
			PreviousTag: previousTag,
		})
	}

	endpoint := target.endpoint
	if err := handler.containersRuntime().PullImage(r.Context(), &endpoint, imageRef); err != nil {
		logContainerImage(target.endpoint.ID, stack.ID, stack.Name, target.service, containerName, previousTag, payload.Version, "error", err)
		return httperror.NewError(http.StatusUnprocessableEntity, "Failed to pull image", err)
	}

	if patchErr != nil {
		logContainerImage(target.endpoint.ID, stack.ID, stack.Name, target.service, containerName, previousTag, payload.Version, "error", patchErr)
		return httperror.InternalServerError("Refused to patch the service image", patchErr)
	}

	if handler.ComposeStackManager == nil {
		return httperror.InternalServerError("Compose stack manager is not configured", errors.New("compose stack manager is nil"))
	}

	wrote := false
	stackFolder := strconv.Itoa(int(stack.ID))
	if !bytes.Equal(patched, original) {
		if _, err := handler.FileService.StoreStackFileFromBytes(stackFolder, stack.EntryPoint, patched); err != nil {
			logContainerImage(target.endpoint.ID, stack.ID, stack.Name, target.service, containerName, previousTag, payload.Version, "error", err)
			return httperror.InternalServerError("Unable to persist the stack file", err)
		}
		wrote = true
	}

	upErr := handler.ComposeStackManager.Up(r.Context(), stack, &endpoint, portainer.ComposeUpOptions{
		ForceRecreate: false,
		Prune:         false,
		Services:      []string{target.service},
	})
	if upErr != nil {
		if wrote {
			if _, restoreErr := handler.FileService.StoreStackFileFromBytes(stackFolder, stack.EntryPoint, original); restoreErr != nil {
				joined := errors.Join(upErr, restoreErr)
				logContainerImage(target.endpoint.ID, stack.ID, stack.Name, target.service, containerName, previousTag, payload.Version, "error", joined)
				return httperror.InternalServerError("Failed to update the container and restore the stack file", joined)
			}
		}
		logContainerImage(target.endpoint.ID, stack.ID, stack.Name, target.service, containerName, previousTag, payload.Version, "error", upErr)
		return httperror.InternalServerError("Failed to update the container", upErr)
	}

	logContainerImage(target.endpoint.ID, stack.ID, stack.Name, target.service, containerName, previousTag, payload.Version, containerImageStatusUpdated, nil)
	return response.JSON(w, &containerImageResponse{
		Status:      containerImageStatusUpdated,
		EndpointID:  target.endpoint.ID,
		StackID:     stack.ID,
		StackName:   stack.Name,
		Service:     target.service,
		Container:   containerName,
		Image:       imageRef,
		PreviousTag: previousTag,
	})
}

func (handler *Handler) withStackImageLock(id portainer.StackID, fn func() *httperror.HandlerError) *httperror.HandlerError {
	handler.stackImageMu.Lock()
	if handler.stackImageLocks == nil {
		handler.stackImageLocks = make(map[portainer.StackID]*sync.Mutex)
	}
	mu, ok := handler.stackImageLocks[id]
	if !ok {
		mu = &sync.Mutex{}
		handler.stackImageLocks[id] = mu
	}
	handler.stackImageMu.Unlock()

	mu.Lock()
	defer mu.Unlock()
	return fn()
}

func (handler *Handler) containersRuntime() stackContainerRuntime {
	if handler.containerRuntime != nil {
		return handler.containerRuntime
	}
	return dockerContainerRuntime{factory: handler.DockerClientFactory}
}

func containerMatchesQuery(ctr container.Summary, query string) bool {
	query = strings.TrimPrefix(query, "/")
	if query == "" {
		return false
	}
	for _, name := range ctr.Names {
		if strings.TrimPrefix(name, "/") == query {
			return true
		}
	}
	if ctr.Labels[composeServiceLabel] == query || ctr.Labels[consts.ComposeStackNameLabel] == query {
		return true
	}
	return false
}

func containerDisplayName(ctr container.Summary, fallback string) string {
	for _, name := range ctr.Names {
		trimmed := strings.TrimPrefix(name, "/")
		if trimmed != "" {
			return trimmed
		}
	}
	return fallback
}

func stackIsGitBacked(stack *portainer.Stack) bool {
	return stack.GitConfig != nil || stack.WorkflowID != 0 || stack.AutoUpdate != nil
}

func composeServiceImage(content []byte, service string) (string, error) {
	details := composetypes.ConfigDetails{
		ConfigFiles: []composetypes.ConfigFile{{Content: content}},
		Environment: map[string]string{},
	}
	project, err := composeloader.LoadWithContext(context.Background(), details, func(options *composeloader.Options) {
		options.SkipValidation = true
		options.ResolvePaths = false
		options.SkipInclude = true
		options.SetProjectName("stack", true)
	})
	if err != nil {
		return "", err
	}
	svc, ok := project.Services[service]
	if !ok {
		return "", fmt.Errorf("%w: %s", stackutils.ErrServiceNotFound, service)
	}
	return svc.Image, nil
}

// imageRepoAndTag splits an image reference. The tag colon is the last colon
// after the last slash. A digest after @ is ignored so a running container
// reported as repo:tag@sha256 still counts as that tag.
func imageRepoAndTag(imageRef string) (repo, tag string, ok bool) {
	imageRef = strings.TrimSpace(imageRef)
	if imageRef == "" {
		return "", "", false
	}
	name := imageRef
	if i := strings.LastIndex(imageRef, "@"); i >= 0 {
		name = imageRef[:i]
		if name == "" || imageRef[i+1:] == "" {
			return "", "", false
		}
	}
	slash := strings.LastIndex(name, "/")
	colon := strings.LastIndex(name, ":")
	if colon > slash {
		repo = name[:colon]
		tag = name[colon+1:]
		if repo == "" || tag == "" {
			return "", "", false
		}
		return repo, tag, true
	}
	if name == "" {
		return "", "", false
	}
	return name, "", true
}

func runningImageHasTag(imageRef, repo, tag string) bool {
	gotRepo, gotTag, ok := imageRepoAndTag(imageRef)
	return ok && gotRepo == repo && gotTag == tag
}

func logContainerImage(endpointID portainer.EndpointID, stackID portainer.StackID, stackName, service, containerName, previousTag, newTag, status string, err error) {
	var event *zerolog.Event
	if err != nil {
		event = log.Error().Err(err)
	} else {
		event = log.Info()
	}
	event.
		Int("endpointId", int(endpointID)).
		Int("stackId", int(stackID)).
		Str("stackName", stackName).
		Str("service", service).
		Str("container", containerName).
		Str("previousTag", previousTag).
		Str("newTag", newTag).
		Str("status", status).
		Msg("stack container image update")
}

func (d dockerContainerRuntime) ListContainers(ctx context.Context, endpoint *portainer.Endpoint) ([]container.Summary, error) {
	if d.factory == nil {
		return nil, errors.New("docker client factory is not configured")
	}
	cli, err := d.factory.CreateClient(endpoint, "", nil)
	if err != nil {
		return nil, err
	}
	defer logs.CloseAndLogErr(cli)

	return cli.ContainerList(ctx, container.ListOptions{All: true})
}

func (d dockerContainerRuntime) PullImage(ctx context.Context, endpoint *portainer.Endpoint, ref string) error {
	if d.factory == nil {
		return errors.New("docker client factory is not configured")
	}
	cli, err := d.factory.CreateClient(endpoint, "", nil)
	if err != nil {
		return err
	}
	defer logs.CloseAndLogErr(cli)

	body, err := cli.ImagePull(ctx, ref, image.PullOptions{})
	if err != nil {
		return err
	}
	defer logs.CloseAndLogErr(body)

	return consumeImagePull(body)
}

func consumeImagePull(body io.Reader) error {
	decoder := json.NewDecoder(body)
	for {
		var msg struct {
			Error string `json:"error"`
		}
		err := decoder.Decode(&msg)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read image pull response: %w", err)
		}
		if msg.Error != "" {
			return errors.New(msg.Error)
		}
	}
}
