package stackutils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// yammyLikeCompose is shaped like the yammy stack file: several services and a
// YAML logging anchor. It does not copy production secrets.
const yammyLikeCompose = `# Shaped like the yammy stack: several services and a YAML anchor.
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

func TestImageTagPatchYammyFixture(t *testing.T) {
	t.Parallel()

	content := []byte(yammyLikeCompose)
	got, err := PatchServiceImageTag(content, "yammy-site", "35")
	require.NoError(t, err)

	oldLine := "    image: solidexpert/service.yammy-site:34"
	newLine := "    image: solidexpert/service.yammy-site:35"
	require.Equal(t, 1, strings.Count(yammyLikeCompose, oldLine))
	expected := strings.Replace(yammyLikeCompose, oldLine, newLine, 1)
	require.Equal(t, expected, string(got))
	require.Contains(t, string(got), "&default-logging")
	require.Contains(t, string(got), "logging: *default-logging")
	require.Contains(t, string(got), "image: solidexpert/yammy-service.net:34")
	require.Contains(t, string(got), "image: redis:6-alpine")
}

func TestImageTagAlreadyPresent(t *testing.T) {
	t.Parallel()

	content := []byte(yammyLikeCompose)
	got, err := PatchServiceImageTag(content, "yammy-site", "34")
	require.NoError(t, err)
	require.Equal(t, content, got)
}

func TestImageTagUnknownService(t *testing.T) {
	t.Parallel()

	content := []byte(yammyLikeCompose)
	got, err := PatchServiceImageTag(content, "no-such-service", "35")
	require.ErrorIs(t, err, ErrServiceNotFound)
	require.Equal(t, content, got)
}

func TestImageTagRejectsBadTag(t *testing.T) {
	t.Parallel()

	content := []byte(yammyLikeCompose)
	bad := []string{"", "foo/bar", "foo:bar", "foo bar", "foo\tbar", " 35", "35\n"}
	for _, tag := range bad {
		got, err := PatchServiceImageTag(content, "yammy-site", tag)
		require.ErrorIs(t, err, ErrInvalidImageTag, tag)
		require.Equal(t, content, got)
	}
}

func TestImageTagDigestOnlyRefused(t *testing.T) {
	t.Parallel()

	content := []byte(`
services:
  app:
    image: solidexpert/app@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
  other:
    image: redis:6
`)
	got, err := PatchServiceImageTag(content, "app", "2")
	require.ErrorIs(t, err, ErrImageTagRefused)
	require.Equal(t, content, got)
}

func TestImageTagDigestPinRefused(t *testing.T) {
	t.Parallel()

	content := []byte(`
services:
  app:
    image: solidexpert/app:1@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
`)
	got, err := PatchServiceImageTag(content, "app", "2")
	require.ErrorIs(t, err, ErrImageTagRefused)
	require.Equal(t, content, got)
}

func TestImageTagDuplicateImageLineRefused(t *testing.T) {
	t.Parallel()

	content := []byte(`
services:
  app:
    image: solidexpert/app:1
    image: solidexpert/app:2
  other:
    image: redis:6
`)
	got, err := PatchServiceImageTag(content, "app", "3")
	require.ErrorIs(t, err, ErrImageTagRefused)
	require.Equal(t, content, got)
}

func TestImageTagMissingImageRefused(t *testing.T) {
	t.Parallel()

	content := []byte(`
services:
  app:
    restart: always
`)
	got, err := PatchServiceImageTag(content, "app", "1")
	require.ErrorIs(t, err, ErrImageTagRefused)
	require.Equal(t, content, got)
}

func TestImageTagSameImageOnTwoServices(t *testing.T) {
	t.Parallel()

	content := []byte(`
services:
  web:
    image: localhost:5000/app:1
  other:
    image: localhost:5000/app:1
`)
	got, err := PatchServiceImageTag(content, "web", "2")
	require.NoError(t, err)
	require.Equal(t, `
services:
  web:
    image: localhost:5000/app:2
  other:
    image: localhost:5000/app:1
`, string(got))
}

func TestImageTagPreservesQuotesCommentsAndCRLF(t *testing.T) {
	t.Parallel()

	content := []byte("services:\r\n  web:\r\n    image: \"nginx:1\" # keep\r\n  db:\r\n    image: redis:6\r\n")
	got, err := PatchServiceImageTag(content, "web", "2")
	require.NoError(t, err)
	require.Equal(t, "services:\r\n  web:\r\n    image: \"nginx:2\" # keep\r\n  db:\r\n    image: redis:6\r\n", string(got))
}

func TestImageTagUntaggedImage(t *testing.T) {
	t.Parallel()

	content := []byte(`
services:
  web:
    image: nginx
  db:
    image: redis:6
`)
	got, err := PatchServiceImageTag(content, "web", "1.25")
	require.NoError(t, err)
	require.Equal(t, `
services:
  web:
    image: nginx:1.25
  db:
    image: redis:6
`, string(got))
}

func TestImageTagNestedLabelIsNotADuplicate(t *testing.T) {
	t.Parallel()

	content := []byte(`
services:
  app:
    image: nginx:1
    labels:
      image: "not-the-service-image"
  db:
    image: redis:6
`)
	got, err := PatchServiceImageTag(content, "app", "2")
	require.NoError(t, err)
	require.Contains(t, string(got), "image: nginx:2")
	require.Contains(t, string(got), "image: \"not-the-service-image\"")
	require.Contains(t, string(got), "image: redis:6")
}
