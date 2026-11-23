package parser

import (
	"encoding/hex"
	"regexp"
	"strings"
)

var (
	emailLike   = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)
	cfEmailAttr = regexp.MustCompile(`(?i)data-cfemail=["']([0-9a-f]+)["']`)
	obfuscated  = regexp.MustCompile(`(?i)([a-z0-9._%+\-]+)\s*(?:\[at\]|\(at\))\s*([a-z0-9.\-]+)\s*(?:\[dot\]|\(dot\))\s*([a-z]{2,})`)
)

// keepEmails normalises matches from the email field and drops token-shaped
// local parts. The raw page regex also matches Cloudflare and analytics
// tokens that happen to contain an @.
func keepEmails(matches []string) []string {
	seen := make(map[string]struct{}, len(matches))
	kept := make([]string, 0, len(matches))
	for _, match := range matches {
		email, ok := acceptEmail(match)
		if !ok {
			continue
		}
		if _, exists := seen[email]; exists {
			continue
		}
		seen[email] = struct{}{}
		kept = append(kept, email)
	}
	return kept
}

// extraEmailCandidates finds addresses the field regex misses when they are
// obfuscated rather than written as user@host.
func extraEmailCandidates(body string) []string {
	var found []string
	for _, match := range cfEmailAttr.FindAllStringSubmatch(body, -1) {
		if len(match) < 2 {
			continue
		}
		if decoded := decodeCloudflareEmail(match[1]); decoded != "" {
			found = append(found, decoded)
		}
	}
	for _, match := range obfuscated.FindAllStringSubmatch(body, -1) {
		if len(match) < 4 {
			continue
		}
		found = append(found, match[1]+"@"+match[2]+"."+match[3])
	}
	return found
}

func acceptEmail(raw string) (string, bool) {
	raw = strings.Trim(strings.TrimSpace(raw), "<>\"'()[]")
	raw = strings.TrimRight(raw, ".,;:")
	found := emailLike.FindString(raw)
	if found == "" {
		return "", false
	}
	local, domain, ok := strings.Cut(strings.ToLower(found), "@")
	if !ok || local == "" || domain == "" || strings.HasPrefix(domain, ".") || strings.Contains(domain, "..") {
		return "", false
	}
	if !containsLetter(local) || tokenLocal(local) {
		return "", false
	}
	return local + "@" + domain, true
}

func containsLetter(value string) bool {
	for _, r := range value {
		if r >= 'a' && r <= 'z' {
			return true
		}
	}
	return false
}

// tokenLocal reports a local part that is long and almost entirely hex, which
// is how site analytics and email-protection tokens show up.
func tokenLocal(local string) bool {
	if len(local) < 24 {
		return false
	}
	hexDigits := 0
	for _, r := range local {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'f':
			hexDigits++
		}
	}
	return hexDigits*100/len(local) >= 85
}

func decodeCloudflareEmail(encoded string) string {
	raw, err := hex.DecodeString(encoded)
	if err != nil || len(raw) < 2 {
		return ""
	}
	key := raw[0]
	out := make([]byte, len(raw)-1)
	for i := 1; i < len(raw); i++ {
		out[i-1] = raw[i] ^ key
	}
	return string(out)
}
