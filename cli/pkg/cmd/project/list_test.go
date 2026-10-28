// Copyright (c) 2026, WSO2 LLC. (https://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package project

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	amsvc "github.com/wso2/agent-manager/cli/pkg/clients/amsvc/gen"
)

func TestList_Success(t *testing.T) {
	io, out, _ := newTestIO(true)
	clientFn, captured, closeFn := newTestClient(t, http.StatusOK, amsvc.ProjectListResponse{
		Limit:  20,
		Offset: 0,
		Total:  2,
		Projects: []amsvc.ProjectListItem{
			{Name: "alpha", DisplayName: "Alpha"},
			{Name: "beta", DisplayName: "Beta"},
		},
	})
	defer closeFn()

	err := runList(context.Background(), &ListOptions{
		IO: io, Client: clientFn, Org: "acme", Scope: baseScope(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !captured.called {
		t.Fatal("server should have been called")
	}
	if captured.path != "/orgs/acme/projects" {
		t.Errorf("path = %q, want /orgs/acme/projects", captured.path)
	}
	env := decodeEnvelope(t, out.String())
	data := env["data"].(map[string]any)
	projects := data["projects"].([]any)
	if len(projects) != 2 {
		t.Errorf("len(projects) = %d, want 2", len(projects))
	}
}

func TestList_ServerError(t *testing.T) {
	io, out, _ := newTestIO(true)
	clientFn, _, closeFn := newTestClient(t, http.StatusInternalServerError, amsvc.ErrorResponse{
		Code:    "INTERNAL",
		Message: "something broke",
	})
	defer closeFn()

	err := runList(context.Background(), &ListOptions{
		IO: io, Client: clientFn, Org: "acme", Scope: baseScope(),
	})
	if err == nil {
		t.Fatal("expected error for 500")
	}
	env := decodeEnvelope(t, out.String())
	errBody := env["error"].(map[string]any)
	if errBody["code"] != "INTERNAL" {
		t.Errorf("code = %v, want INTERNAL", errBody["code"])
	}
}

func TestList_SendsPagination(t *testing.T) {
	io, _, _ := newTestIO(true)
	clientFn, captured, closeFn := newTestClient(t, http.StatusOK, amsvc.ProjectListResponse{
		Limit: 5, Offset: 10, Total: 15,
		Projects: []amsvc.ProjectListItem{{Name: "alpha"}},
	})
	defer closeFn()

	limit, offset := 5, 10
	if err := runList(context.Background(), &ListOptions{
		IO: io, Client: clientFn, Org: "acme", Scope: baseScope(),
		Limit: &limit, Offset: &offset,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(captured.rawQuery, "limit=5") || !strings.Contains(captured.rawQuery, "offset=10") {
		t.Errorf("rawQuery = %q, want limit=5 and offset=10", captured.rawQuery)
	}
}

func TestList_TruncationHintOnTTY(t *testing.T) {
	io, _, errOut := newTextTestIO(true)
	projects := make([]amsvc.ProjectListItem, 50)
	for i := range projects {
		projects[i] = amsvc.ProjectListItem{Name: "p", CreatedAt: time.Unix(0, 0).UTC()}
	}
	clientFn, _, closeFn := newTestClient(t, http.StatusOK, amsvc.ProjectListResponse{
		Limit: 50, Offset: 0, Total: 63, Projects: projects,
	})
	defer closeFn()

	if err := runList(context.Background(), &ListOptions{
		IO: io, Client: clientFn, Org: "acme", Scope: baseScope(),
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(errOut.String(), "Showing 50 of 63 projects") {
		t.Errorf("errOut = %q, want truncation hint", errOut.String())
	}
}

func TestListCmd_RejectsZeroLimit(t *testing.T) {
	io, _, _ := newTestIO(true)
	clientFn, captured, closeFn := newTestClient(t, http.StatusOK, amsvc.ProjectListResponse{})
	defer closeFn()

	root := testProjectCmd(t, io, clientFn)
	root.SetArgs([]string{"project", "list", "--limit", "0"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected an error for --limit 0")
	}
	if captured.called {
		t.Error("server was called despite an invalid --limit")
	}
}

func TestListCmd_RejectsNegativeOffset(t *testing.T) {
	io, _, _ := newTestIO(true)
	clientFn, captured, closeFn := newTestClient(t, http.StatusOK, amsvc.ProjectListResponse{})
	defer closeFn()

	root := testProjectCmd(t, io, clientFn)
	root.SetArgs([]string{"project", "list", "--offset", "-1"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected an error for --offset -1")
	}
	if captured.called {
		t.Error("server was called despite an invalid --offset")
	}
}
