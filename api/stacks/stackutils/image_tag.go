package stackutils

import (
	"context"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	composeloader "github.com/compose-spec/compose-go/v2/loader"
	composetypes "github.com/compose-spec/compose-go/v2/types"
	"go.yaml.in/yaml/v3"
)

// ErrServiceNotFound means the compose service key is not in the file.
var ErrServiceNotFound = fmt.Errorf("compose service not found")

// ErrInvalidImageTag means the requested tag is not a single Docker tag.
var ErrInvalidImageTag = fmt.Errorf("invalid image tag")

// ErrImageTagRefused means the compose file was left unchanged because the
// image cannot be patched safely: duplicate image keys, a digest pin, a
// missing image, or the post-edit check failed.
var ErrImageTagRefused = fmt.Errorf("image tag patch refused")

// PatchServiceImageTag replaces the image tag of one compose service.
//
// content is the stack compose file. service is the compose service key
// (not a Docker container name). tag is the new tag only.
//
// The write is a line edit of that service's image scalar. The file is not
// remarshalled, so YAML anchors stay as written. After the edit the file is
// parsed again with compose-go. The result is refused when any other
// service's image string changes or when the target repository (everything
// before the tag) changes.
//
// When the service already uses tag, the original bytes are returned with a
// nil error. On every error the original bytes are returned unchanged.
func PatchServiceImageTag(content []byte, service, tag string) ([]byte, error) {
	if err := validateImageTag(tag); err != nil {
		return content, err
	}
	if service == "" {
		return content, fmt.Errorf("%w: service name is empty", ErrServiceNotFound)
	}

	located, err := locateServiceImage(content, service)
	if err != nil {
		return content, err
	}

	project, err := loadComposeProject(content)
	if err != nil {
		return content, fmt.Errorf("%w: parse compose: %v", ErrImageTagRefused, err)
	}

	resolved, ok := project.Services[service]
	if !ok {
		return content, fmt.Errorf("%w: service %q is not in the loaded compose project", ErrImageTagRefused, service)
	}
	if resolved.Image != located.value {
		return content, fmt.Errorf("%w: service %q image is not a literal value", ErrImageTagRefused, service)
	}

	repo, currentTag, digest, ok := splitImageRef(located.value)
	if !ok || repo == "" {
		return content, fmt.Errorf("%w: service %q image %q is not a repository reference", ErrImageTagRefused, service, located.value)
	}
	if digest != "" {
		return content, fmt.Errorf("%w: service %q image is pinned by digest", ErrImageTagRefused, service)
	}
	if currentTag == tag {
		return content, nil
	}

	patched, err := replaceImageValue(content, located, repo+":"+tag)
	if err != nil {
		return content, err
	}

	after, err := loadComposeProject(patched)
	if err != nil {
		return content, fmt.Errorf("%w: patched compose failed to parse: %v", ErrImageTagRefused, err)
	}
	if err := verifyImagePatch(project, after, service, repo, tag); err != nil {
		return content, err
	}

	return patched, nil
}

// validateImageTag accepts one Docker tag: non-empty, no slash, no colon, no whitespace.
func validateImageTag(tag string) error {
	if tag == "" || strings.ContainsAny(tag, "/:") || strings.ContainsFunc(tag, unicode.IsSpace) {
		return fmt.Errorf("%w: tag must be a single Docker tag without '/', ':', or whitespace", ErrInvalidImageTag)
	}
	return nil
}

type locatedImage struct {
	line   int
	column int
	style  yaml.Style
	value  string
}

func locateServiceImage(content []byte, service string) (locatedImage, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return locatedImage{}, fmt.Errorf("%w: parse yaml: %v", ErrImageTagRefused, err)
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) != 1 || doc.Content[0].Kind != yaml.MappingNode {
		return locatedImage{}, fmt.Errorf("%w: compose file must be one YAML mapping", ErrImageTagRefused)
	}
	root := doc.Content[0]

	_, services, ok := mappingChild(root, "services")
	if !ok || services.Kind != yaml.MappingNode {
		return locatedImage{}, fmt.Errorf("%w: %s", ErrServiceNotFound, service)
	}

	var serviceNode *yaml.Node
	startLine := 0
	endLine := 0
	for i := 0; i+1 < len(services.Content); i += 2 {
		key := services.Content[i]
		if key.Kind != yaml.ScalarNode || key.Value != service {
			continue
		}
		serviceNode = services.Content[i+1]
		startLine = key.Line
		endLine = -1
		if i+2 < len(services.Content) {
			endLine = services.Content[i+2].Line
		} else if next := nextMappingKeyLine(root, "services"); next > 0 {
			endLine = next
		}
		break
	}
	if serviceNode == nil {
		return locatedImage{}, fmt.Errorf("%w: %s", ErrServiceNotFound, service)
	}
	if serviceNode.Kind != yaml.MappingNode {
		return locatedImage{}, fmt.Errorf("%w: service %q is not a mapping", ErrImageTagRefused, service)
	}

	var imageValues []*yaml.Node
	var imageColumn int
	for i := 0; i+1 < len(serviceNode.Content); i += 2 {
		key := serviceNode.Content[i]
		if key.Kind == yaml.ScalarNode && key.Value == "image" {
			imageValues = append(imageValues, serviceNode.Content[i+1])
			if imageColumn == 0 {
				imageColumn = key.Column
			}
		}
	}
	if len(imageValues) == 0 {
		return locatedImage{}, fmt.Errorf("%w: service %q has no image", ErrImageTagRefused, service)
	}
	duplicates := len(imageValues)
	if imageColumn > 0 {
		if n := countKeyAtColumn(content, startLine, endLine, imageColumn, "image"); n > duplicates {
			duplicates = n
		}
	}
	if duplicates > 1 {
		return locatedImage{}, fmt.Errorf("%w: service %q has duplicate image lines", ErrImageTagRefused, service)
	}

	value := imageValues[0]
	if value.Kind == yaml.AliasNode {
		return locatedImage{}, fmt.Errorf("%w: service %q image is a YAML alias", ErrImageTagRefused, service)
	}
	if value.Kind != yaml.ScalarNode || value.Value == "" {
		return locatedImage{}, fmt.Errorf("%w: service %q image is empty", ErrImageTagRefused, service)
	}
	if value.Style&yaml.LiteralStyle != 0 || value.Style&yaml.FoldedStyle != 0 {
		return locatedImage{}, fmt.Errorf("%w: service %q image is a multiline scalar", ErrImageTagRefused, service)
	}

	return locatedImage{
		line:   value.Line,
		column: value.Column,
		style:  value.Style,
		value:  value.Value,
	}, nil
}

func mappingChild(mapping *yaml.Node, key string) (keyNode, valueNode *yaml.Node, found bool) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		k := mapping.Content[i]
		if k.Kind == yaml.ScalarNode && k.Value == key {
			return k, mapping.Content[i+1], true
		}
	}
	return nil, nil, false
}

func nextMappingKeyLine(mapping *yaml.Node, key string) int {
	seen := false
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if seen {
			return mapping.Content[i].Line
		}
		k := mapping.Content[i]
		if k.Kind == yaml.ScalarNode && k.Value == key {
			seen = true
		}
	}
	return 0
}

func countKeyAtColumn(content []byte, startLine, endLine, column int, key string) int {
	count := 0
	lineNo := 1
	lineStart := 0
	flush := func(lineEnd int) {
		if lineNo < startLine || (endLine > 0 && lineNo >= endLine) {
			return
		}
		line := content[lineStart:lineEnd]
		if lineHasKeyAtColumn(line, column, key) {
			count++
		}
	}
	for i := 0; i < len(content); i++ {
		if content[i] != '\n' {
			continue
		}
		flush(i)
		lineNo++
		lineStart = i + 1
	}
	flush(len(content))
	return count
}

func lineHasKeyAtColumn(line []byte, column int, key string) bool {
	line = trimCR(line)
	offset, ok := columnByteOffset(line, column)
	if !ok {
		return false
	}
	return strings.HasPrefix(string(line[offset:]), key+":")
}

func loadComposeProject(content []byte) (*composetypes.Project, error) {
	details := composetypes.ConfigDetails{
		ConfigFiles: []composetypes.ConfigFile{{Content: content}},
		Environment: map[string]string{},
	}
	return composeloader.LoadWithContext(context.Background(), details, func(o *composeloader.Options) {
		o.SkipValidation = true
		o.ResolvePaths = false
		o.SkipInclude = true
		o.SetProjectName("stack", true)
	})
}

func verifyImagePatch(before, after *composetypes.Project, service, repo, tag string) error {
	beforeImages := serviceImages(before)
	afterImages := serviceImages(after)
	if len(beforeImages) != len(afterImages) {
		return fmt.Errorf("%w: service set changed during patch", ErrImageTagRefused)
	}
	for name, image := range beforeImages {
		got, ok := afterImages[name]
		if !ok {
			return fmt.Errorf("%w: service %q missing after patch", ErrImageTagRefused, name)
		}
		if name == service {
			continue
		}
		if got != image {
			return fmt.Errorf("%w: service %q image changed", ErrImageTagRefused, name)
		}
	}
	updated, ok := afterImages[service]
	if !ok {
		return fmt.Errorf("%w: service %q missing after patch", ErrImageTagRefused, service)
	}
	gotRepo, gotTag, gotDigest, ok := splitImageRef(updated)
	if !ok || gotDigest != "" || gotRepo != repo || gotTag != tag {
		return fmt.Errorf("%w: patched image %q failed the repository check", ErrImageTagRefused, updated)
	}
	return nil
}

func serviceImages(project *composetypes.Project) map[string]string {
	images := make(map[string]string, len(project.Services))
	for name, svc := range project.Services {
		images[name] = svc.Image
	}
	return images
}

// splitImageRef splits an image reference into repository, tag, and digest.
// The repository is everything before the tag. A registry port stays in the
// repository: the tag colon is the last colon after the last slash.
func splitImageRef(image string) (repo, tag, digest string, ok bool) {
	image = strings.TrimSpace(image)
	if image == "" {
		return "", "", "", false
	}
	name := image
	if i := strings.LastIndex(image, "@"); i >= 0 {
		name = image[:i]
		digest = image[i+1:]
		if name == "" || digest == "" {
			return "", "", "", false
		}
	}
	slash := strings.LastIndex(name, "/")
	colon := strings.LastIndex(name, ":")
	if colon > slash {
		repo = name[:colon]
		tag = name[colon+1:]
		if repo == "" || tag == "" {
			return "", "", "", false
		}
		return repo, tag, digest, true
	}
	if name == "" {
		return "", "", "", false
	}
	return name, "", digest, true
}

func replaceImageValue(content []byte, located locatedImage, newValue string) ([]byte, error) {
	start, end, ok := lineBounds(content, located.line)
	if !ok {
		return nil, fmt.Errorf("%w: image line %d not found", ErrImageTagRefused, located.line)
	}
	updated, err := spliceImageScalar(content[start:end], located.column, located.value, newValue, located.style)
	if err != nil {
		return nil, err
	}
	out := make([]byte, 0, len(content)-(end-start)+len(updated))
	out = append(out, content[:start]...)
	out = append(out, updated...)
	out = append(out, content[end:]...)
	return out, nil
}

func spliceImageScalar(line []byte, column int, oldValue, newValue string, style yaml.Style) ([]byte, error) {
	offset, ok := columnByteOffset(trimCR(line), column)
	if !ok {
		return nil, fmt.Errorf("%w: image scalar is outside the line", ErrImageTagRefused)
	}
	// Keep a trailing CR on the line so CRLF files stay CRLF.
	cr := []byte(nil)
	body := line
	if len(line) > 0 && line[len(line)-1] == '\r' {
		cr = line[len(line)-1:]
		body = line[:len(line)-1]
	}
	if offset > len(body) {
		return nil, fmt.Errorf("%w: image scalar is outside the line", ErrImageTagRefused)
	}
	rest := body[offset:]

	var rawOld, rawNew string
	switch {
	case style&yaml.DoubleQuotedStyle != 0:
		rawOld = `"` + oldValue + `"`
		rawNew = `"` + newValue + `"`
	case style&yaml.SingleQuotedStyle != 0:
		rawOld = "'" + oldValue + "'"
		rawNew = "'" + newValue + "'"
	default:
		rawOld = oldValue
		rawNew = newValue
	}
	if !strings.HasPrefix(string(rest), rawOld) {
		return nil, fmt.Errorf("%w: image scalar does not match the parsed value", ErrImageTagRefused)
	}
	if style&yaml.DoubleQuotedStyle == 0 && style&yaml.SingleQuotedStyle == 0 {
		end := len(rawOld)
		if end < len(rest) {
			c := rest[end]
			if c != ' ' && c != '\t' && c != '#' {
				return nil, fmt.Errorf("%w: image scalar is not a single token", ErrImageTagRefused)
			}
		}
	}

	out := make([]byte, 0, len(body)-len(rawOld)+len(rawNew)+len(cr))
	out = append(out, body[:offset]...)
	out = append(out, rawNew...)
	out = append(out, rest[len(rawOld):]...)
	out = append(out, cr...)
	return out, nil
}

func lineBounds(content []byte, line int) (start, end int, ok bool) {
	if line < 1 {
		return 0, 0, false
	}
	current := 1
	start = 0
	for i := 0; i < len(content); i++ {
		if content[i] != '\n' {
			continue
		}
		if current == line {
			return start, i, true
		}
		current++
		start = i + 1
	}
	if current == line {
		return start, len(content), true
	}
	return 0, 0, false
}

func columnByteOffset(line []byte, column int) (int, bool) {
	if column < 1 {
		return 0, false
	}
	runeIdx := 0
	for i := 0; i < len(line); {
		if runeIdx == column-1 {
			return i, true
		}
		_, size := utf8.DecodeRune(line[i:])
		if size <= 0 {
			return 0, false
		}
		i += size
		runeIdx++
	}
	return 0, false
}

func trimCR(line []byte) []byte {
	if len(line) > 0 && line[len(line)-1] == '\r' {
		return line[:len(line)-1]
	}
	return line
}
