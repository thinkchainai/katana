package output

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/valyala/fasttemplate"
)

func (w *StandardWriter) formatTemplate(output *Result) ([]byte, error) {
	emails := distinctCustomField(output, "email")
	if len(emails) <= 1 {
		return w.renderTemplate(output)
	}
	lines := make([]string, 0, len(emails))
	for _, email := range emails {
		clone := *output
		req := *output.Request
		fields := make(map[string][]string, len(req.CustomFields))
		for key, values := range req.CustomFields {
			fields[key] = values
		}
		fields["email"] = []string{email}
		req.CustomFields = fields
		clone.Request = &req
		line, err := w.renderTemplate(&clone)
		if err != nil {
			return nil, err
		}
		if len(line) > 0 {
			lines = append(lines, string(line))
		}
	}
	if len(lines) == 0 {
		return nil, nil
	}
	return []byte(strings.Join(lines, "\n")), nil
}

func distinctCustomField(output *Result, name string) []string {
	if output == nil || output.Request == nil {
		return nil
	}
	seen := map[string]struct{}{}
	var values []string
	for _, value := range output.Request.CustomFields[name] {
		key := strings.ToLower(strings.TrimSpace(value))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		values = append(values, value)
	}
	return values
}

func (w *StandardWriter) renderTemplate(output *Result) ([]byte, error) {
	var fieldOutputs []fieldOutput
	fieldNames := strings.Join(FieldNames, ",")
	fieldOutputs = formatField(output, fieldNames)
	fieldOutputs = append(fieldOutputs, getValueForCustomField(output)...)

	fieldsMap := make(map[string]string)
	for _, fo := range fieldOutputs {
		fieldsMap[fo.field] = fo.value
	}

	errUnknownTag := errors.New("unknown tag")

	tagFn := fasttemplate.TagFunc(func(w io.Writer, tag string) (int, error) {
		value, ok := fieldsMap[tag]
		if !ok {
			return 0, fmt.Errorf("%w %q", errUnknownTag, tag)
		}
		return w.Write([]byte(value))
	})

	out, err := w.outputTemplate.ExecuteFuncStringWithErr(tagFn)
	if err != nil {
		if errors.Is(err, errUnknownTag) {
			// If there is an unknown tag, we just ignore it.
			return nil, nil
		}
		return nil, err
	}

	return []byte(out), nil
}
