package controller

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"afrioracle/internal/model"
)

const waSecret = "app-secret"

func waBody(wamid, from, text string) string {
	return `{"object":"whatsapp_business_account","entry":[{"changes":[{"value":{"messages":[{"from":"` + from +
		`","id":"` + wamid + `","type":"text","text":{"body":"` + text + `"}}]}}]}]}`
}

func sign(body, secret string) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(body))
	return "sha256=" + hex.EncodeToString(m.Sum(nil))
}

func waServer(t *testing.T) (*httptest.Server, model.Store) {
	t.Helper()
	t.Setenv("WHATSAPP_APP_SECRET", waSecret)
	t.Setenv("WHATSAPP_VERIFY_TOKEN", "vt")
	reg := model.NewRegistry()
	reg.Register("A", "tok-A", 2, 88)
	reg.BindPhone("221770000001", "A")
	st := model.NewInMemoryStore()
	ts := httptest.NewServer(NewRouter(NewControllers(st, reg), 1<<20))
	t.Cleanup(ts.Close)
	return ts, st
}

func waPost(t *testing.T, base, body, sig string) (int, string) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, base+"/webhook/whatsapp", strings.NewReader(body))
	if sig != "" {
		req.Header.Set("X-Hub-Signature-256", sig)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func TestWhatsappUnsignedOrBadlySignedIs401(t *testing.T) {
	ts, st := waServer(t)
	body := waBody("w1", "221770000001", "mil 187 FCFA kilo Kaolack")
	for name, sig := range map[string]string{
		"missing": "", "wrong secret": sign(body, "other"), "no prefix": strings.TrimPrefix(sign(body, waSecret), "sha256="),
		"garbage": "sha256=zz",
	} {
		if code, _ := waPost(t, ts.URL, body, sig); code != 401 {
			t.Errorf("%s: got %d want 401", name, code)
		}
	}
	if n := len(st.Submissions()); n != 0 {
		t.Fatalf("stored %d submissions from unauthenticated posts", n)
	}
}

func TestWhatsappUnsetSecretFailsClosed(t *testing.T) {
	ts, _ := waServer(t)
	t.Setenv("WHATSAPP_APP_SECRET", "")
	body := waBody("w1", "221770000001", "mil 187")
	if code, _ := waPost(t, ts.URL, body, sign(body, "")); code != 401 {
		t.Fatalf("got %d want 401", code)
	}
}

func TestWhatsappVerifyFailsClosedWhenTokenUnset(t *testing.T) {
	ts, _ := waServer(t)
	t.Setenv("WHATSAPP_VERIFY_TOKEN", "")
	code, _ := get(t, ts.URL+"/webhook/whatsapp?hub.mode=subscribe&hub.verify_token=&hub.challenge=abc")
	if code != 403 {
		t.Fatalf("both-empty verify got %d want 403", code)
	}
	t.Setenv("WHATSAPP_VERIFY_TOKEN", "vt")
	resp, _ := http.Get(ts.URL + "/webhook/whatsapp?hub.mode=subscribe&hub.verify_token=vt&hub.challenge=abc")
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || string(b) != "abc" {
		t.Fatalf("valid handshake: %d %q", resp.StatusCode, b)
	}
	resp, _ = http.Get(ts.URL + "/webhook/whatsapp?hub.mode=subscribe&hub.verify_token=bad&hub.challenge=abc")
	if resp.StatusCode != 403 {
		t.Fatalf("bad token got %d", resp.StatusCode)
	}
}

func TestWhatsappSenderMapsToRegisteredNode(t *testing.T) {
	ts, st := waServer(t)
	body := waBody("w1", "221770000001", "mil 187 FCFA kilo Kaolack")
	if code, out := waPost(t, ts.URL, body, sign(body, waSecret)); code != 201 {
		t.Fatalf("got %d %s", code, out)
	}
	subs := st.Submissions()
	if len(subs) != 1 || subs[0].NodeID != "A" || subs[0].Tier != 2 {
		t.Fatalf("want one submission from A tier 2, got %+v", subs)
	}
}

func TestWhatsappUnknownSenderRejected(t *testing.T) {
	ts, st := waServer(t)
	body := waBody("w2", "999", "mil 187 FCFA kilo Kaolack")
	if code, _ := waPost(t, ts.URL, body, sign(body, waSecret)); code != 403 {
		t.Fatalf("got %d want 403", code)
	}
	if len(st.Submissions()) != 0 {
		t.Fatal("unknown sender stored")
	}
}

func TestWhatsappDuplicateWamidIsOneSubmission(t *testing.T) {
	ts, st := waServer(t)
	body := waBody("w3", "221770000001", "mil 187 FCFA kilo Kaolack")
	for i := 0; i < 3; i++ {
		if code, out := waPost(t, ts.URL, body, sign(body, waSecret)); code != 200 && code != 201 {
			t.Fatalf("delivery %d: %d %s", i, code, out)
		}
	}
	if n := len(st.Submissions()); n != 1 {
		t.Fatalf("got %d submissions want 1", n)
	}
}

func TestWhatsappBadJSONIs400AndStatusUpdatesAre200(t *testing.T) {
	ts, st := waServer(t)
	bad := `{not json`
	if code, _ := waPost(t, ts.URL, bad, sign(bad, waSecret)); code != 400 {
		t.Fatalf("bad json got %d want 400", code)
	}
	status := `{"entry":[{"changes":[{"value":{"statuses":[{"id":"x","status":"delivered"}]}}]}]}`
	if code, _ := waPost(t, ts.URL, status, sign(status, waSecret)); code != 200 {
		t.Fatalf("status update got %d want 200", code)
	}
	if len(st.Submissions()) != 0 {
		t.Fatal("stored something")
	}
}
