package output

import (
	"testing"

	"github.com/projectdiscovery/katana/pkg/navigation"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasttemplate"
)

func TestFormatTemplateEmitsEachEmail(t *testing.T) {
	tmpl, err := fasttemplate.NewTemplate("{{email}} {{url}}", "{{", "}}")
	require.NoError(t, err)
	writer := &StandardWriter{outputTemplate: tmpl}
	got, err := writer.formatTemplate(&Result{
		Request: &navigation.Request{
			URL: "https://example.com/contact",
			CustomFields: map[string][]string{
				"email": {"a@example.com", "b@example.com", "A@example.com"},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "a@example.com https://example.com/contact\nb@example.com https://example.com/contact", string(got))
}
