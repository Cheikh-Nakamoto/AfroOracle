package controller

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const maxWhatsAppAudioBytes = 16 << 20

var (
	graphVersionPattern    = regexp.MustCompile(`^v[0-9]+\.[0-9]+$`)
	whatsAppMediaIDPattern = regexp.MustCompile(`^[0-9]+$`)
)

type cloudWhatsAppMediaDownloader struct {
	token   string
	version string
	client  *http.Client
}

type whatsAppMediaMetadata struct {
	URL      string `json:"url"`
	MIMEType string `json:"mime_type"`
	SHA256   string `json:"sha256"`
	FileSize int64  `json:"file_size"`
}

func NewWhatsAppMediaDownloader(token, version string, client *http.Client) WhatsAppMediaDownloader {
	if client == nil {
		client = &http.Client{
			Timeout: 20 * time.Second,
			CheckRedirect: func(req *http.Request, _ []*http.Request) error {
				if req.URL.Scheme != "https" || (!strings.EqualFold(req.URL.Hostname(), "graph.facebook.com") && !strings.EqualFold(req.URL.Hostname(), "lookaside.fbsbx.com")) {
					return fmt.Errorf("refusing redirect to untrusted media host")
				}
				return nil
			},
		}
	}
	return &cloudWhatsAppMediaDownloader{token: token, version: version, client: client}
}

func (d *cloudWhatsAppMediaDownloader) Download(ctx context.Context, mediaID string) ([]byte, string, error) {
	if d.token == "" || !graphVersionPattern.MatchString(d.version) {
		return nil, "", fmt.Errorf("WhatsApp media credentials or Graph API version are not configured")
	}
	if !whatsAppMediaIDPattern.MatchString(mediaID) {
		return nil, "", fmt.Errorf("invalid WhatsApp media ID")
	}
	metadataURL := "https://graph.facebook.com/" + d.version + "/" + url.PathEscape(mediaID)
	metadataReq, err := http.NewRequestWithContext(ctx, http.MethodGet, metadataURL, nil)
	if err != nil {
		return nil, "", err
	}
	metadataReq.Header.Set("Authorization", "Bearer "+d.token)
	metadataResp, err := d.client.Do(metadataReq)
	if err != nil {
		return nil, "", fmt.Errorf("retrieve WhatsApp media metadata: %w", err)
	}
	defer metadataResp.Body.Close()
	if metadataResp.StatusCode < 200 || metadataResp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("retrieve WhatsApp media metadata: HTTP %d", metadataResp.StatusCode)
	}
	var metadata whatsAppMediaMetadata
	if err := json.NewDecoder(io.LimitReader(metadataResp.Body, 32<<10)).Decode(&metadata); err != nil {
		return nil, "", fmt.Errorf("decode WhatsApp media metadata: %w", err)
	}
	if metadata.FileSize <= 0 || metadata.FileSize > maxWhatsAppAudioBytes {
		return nil, "", fmt.Errorf("WhatsApp audio exceeds the allowed size")
	}
	mediaURL, err := url.Parse(metadata.URL)
	if err != nil || mediaURL.Scheme != "https" || mediaURL.User != nil || mediaURL.Port() != "" || !strings.EqualFold(mediaURL.Hostname(), "lookaside.fbsbx.com") {
		return nil, "", fmt.Errorf("WhatsApp returned an untrusted media URL")
	}
	mediaReq, err := http.NewRequestWithContext(ctx, http.MethodGet, mediaURL.String(), nil)
	if err != nil {
		return nil, "", err
	}
	mediaReq.Header.Set("Authorization", "Bearer "+d.token)
	mediaResp, err := d.client.Do(mediaReq)
	if err != nil {
		return nil, "", fmt.Errorf("download WhatsApp media: %w", err)
	}
	defer mediaResp.Body.Close()
	if mediaResp.StatusCode < 200 || mediaResp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("download WhatsApp media: HTTP %d", mediaResp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(mediaResp.Body, maxWhatsAppAudioBytes+1))
	if err != nil {
		return nil, "", fmt.Errorf("read WhatsApp media: %w", err)
	}
	if len(data) == 0 || len(data) > maxWhatsAppAudioBytes || int64(len(data)) != metadata.FileSize {
		return nil, "", fmt.Errorf("WhatsApp media size does not match metadata")
	}
	if metadata.SHA256 != "" {
		want, err := hex.DecodeString(metadata.SHA256)
		if err != nil || len(want) != sha256.Size {
			return nil, "", fmt.Errorf("invalid WhatsApp media checksum")
		}
		got := sha256.Sum256(data)
		if subtle.ConstantTimeCompare(want, got[:]) != 1 {
			return nil, "", fmt.Errorf("WhatsApp media checksum mismatch")
		}
	}
	mediaType, _, err := mime.ParseMediaType(metadata.MIMEType)
	if err != nil || !strings.HasPrefix(mediaType, "audio/") {
		return nil, "", fmt.Errorf("unsupported WhatsApp media type")
	}
	return data, mediaType, nil
}
