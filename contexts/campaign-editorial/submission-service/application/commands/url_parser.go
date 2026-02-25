package commands

import (
	"net/url"
	"strings"

	domainerrors "solomon/contexts/campaign-editorial/submission-service/domain/errors"
)

func extractPostReference(platform string, rawURL string) (string, string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", "", domainerrors.ErrInvalidSubmissionURL
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Host))
	pathSegments := splitPathSegments(parsed.Path)

	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "tiktok":
		return parseTikTok(host, pathSegments)
	case "instagram":
		return parseInstagram(host, pathSegments)
	case "youtube":
		return parseYouTube(host, parsed.Query().Get("v"), pathSegments)
	case "x":
		return parseX(host, pathSegments)
	default:
		return "", "", domainerrors.ErrUnsupportedPlatform
	}
}

func parseTikTok(host string, segments []string) (string, string, error) {
	if strings.Contains(host, "tiktok.com") {
		if len(segments) >= 3 && strings.HasPrefix(segments[0], "@") && segments[1] == "video" {
			return segments[2], segments[0], nil
		}
		if len(segments) >= 1 && strings.Contains(host, "vm.tiktok.com") {
			return segments[0], "", nil
		}
	}
	return "", "", domainerrors.ErrInvalidSubmissionURL
}

func parseInstagram(host string, segments []string) (string, string, error) {
	if !strings.Contains(host, "instagram.com") {
		return "", "", domainerrors.ErrInvalidSubmissionURL
	}
	if len(segments) >= 2 && (segments[0] == "p" || segments[0] == "reel") {
		return segments[1], "", nil
	}
	return "", "", domainerrors.ErrInvalidSubmissionURL
}

func parseYouTube(host string, queryVideoID string, segments []string) (string, string, error) {
	if strings.Contains(host, "youtu.be") && len(segments) >= 1 {
		return segments[0], "", nil
	}
	if strings.Contains(host, "youtube.com") {
		if strings.TrimSpace(queryVideoID) != "" {
			return strings.TrimSpace(queryVideoID), "", nil
		}
		if len(segments) >= 2 && segments[0] == "shorts" {
			return segments[1], "", nil
		}
	}
	return "", "", domainerrors.ErrInvalidSubmissionURL
}

func parseX(host string, segments []string) (string, string, error) {
	if !strings.Contains(host, "x.com") && !strings.Contains(host, "twitter.com") {
		return "", "", domainerrors.ErrInvalidSubmissionURL
	}
	if len(segments) >= 3 && segments[1] == "status" {
		return segments[2], segments[0], nil
	}
	return "", "", domainerrors.ErrInvalidSubmissionURL
}

func splitPathSegments(rawPath string) []string {
	parts := strings.Split(strings.Trim(strings.TrimSpace(rawPath), "/"), "/")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			items = append(items, value)
		}
	}
	return items
}
