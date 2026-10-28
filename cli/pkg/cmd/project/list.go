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

	"github.com/spf13/cobra"

	amsvc "github.com/wso2/agent-manager/cli/pkg/clients/amsvc/gen"
	"github.com/wso2/agent-manager/cli/pkg/clierr"
	"github.com/wso2/agent-manager/cli/pkg/cmdutil"
	"github.com/wso2/agent-manager/cli/pkg/iostreams"
	"github.com/wso2/agent-manager/cli/pkg/render"
	"github.com/wso2/agent-manager/cli/pkg/tableprinter"
)

type ListOptions struct {
	IO           *iostreams.IOStreams
	Client       func(context.Context) (*amsvc.ClientWithResponses, error)
	ResolveScope func(*cobra.Command, bool, bool) (string, string, error)
	MakeScope    func(org, proj string) render.Scope

	Org    string
	Scope  render.Scope
	Limit  *int
	Offset *int
}

func NewListCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &ListOptions{
		IO:           f.IOStreams,
		Client:       f.AgentManager,
		ResolveScope: f.ResolveOrgProject,
		MakeScope:    f.Scope,
	}
	var limit, offset int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List projects in an organization",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			org, _, err := opts.ResolveScope(cmd, true, false)
			scope := opts.MakeScope(org, "")
			if err != nil {
				return render.Error(opts.IO, scope, err)
			}
			opts.Org, opts.Scope = org, scope

			if cmd.Flags().Changed("limit") {
				if limit < 1 {
					return render.Error(opts.IO, scope, cmdutil.FlagErrorf("--limit must be at least 1"))
				}
				opts.Limit = &limit
			}
			if cmd.Flags().Changed("offset") {
				if offset < 0 {
					return render.Error(opts.IO, scope, cmdutil.FlagErrorf("--offset must be at least 0"))
				}
				opts.Offset = &offset
			}
			return runList(cmd.Context(), opts)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of results to return")
	cmd.Flags().IntVar(&offset, "offset", 0, "Number of results to skip")
	return cmd
}

func runList(ctx context.Context, o *ListOptions) error {
	client, err := o.Client(ctx)
	if err != nil {
		return render.Error(o.IO, o.Scope, err)
	}

	resp, err := client.ListProjectsWithResponse(ctx, o.Org, &amsvc.ListProjectsParams{
		Limit:  o.Limit,
		Offset: o.Offset,
	})
	if err != nil {
		return render.Error(o.IO, o.Scope, clierr.Newf(clierr.Transport, "%v", err))
	}
	if resp.JSON200 == nil {
		return render.Error(o.IO, o.Scope, cmdutil.ErrorFromServer(resp.HTTPResponse, cmdutil.FirstNonNil(resp.JSON400, resp.JSON404, resp.JSON500)))
	}

	if o.IO.JSON {
		return render.JSONSuccess(o.IO, o.Scope, resp.JSON200)
	}

	tp := tableprinter.New(o.IO, "name", "display name", "created")
	cs := o.IO.ColorScheme()
	for _, p := range resp.JSON200.Projects {
		tp.AddField(p.Name, tableprinter.WithColor(cs.Bold))
		tp.AddField(p.DisplayName)
		tp.AddField(p.CreatedAt.Format("2006-01-02"), tableprinter.WithColor(cs.Gray))
		tp.EndRow()
	}
	if err := tp.Render(); err != nil {
		return err
	}
	return render.TruncationHint(o.IO, resp.JSON200.Offset+len(resp.JSON200.Projects), resp.JSON200.Total, "projects")
}
