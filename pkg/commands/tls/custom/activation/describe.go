package activation

import (
	"fmt"
	"io"

	"github.com/fastly/cli/pkg/cmd"
	fsterr "github.com/fastly/cli/pkg/errors"
	"github.com/fastly/cli/pkg/global"
	"github.com/fastly/cli/pkg/manifest"
	"github.com/fastly/go-fastly/v8/fastly"
)

var include = []string{"tls_certificate", "tls_configuration", "tls_domain"}

// NewDescribeCommand returns a usable command registered under the parent.
func NewDescribeCommand(parent cmd.Registerer, g *global.Data, m manifest.Data) *DescribeCommand {
	var c DescribeCommand
	c.CmdClause = parent.Command("describe", "Show a TLS configuration").Alias("get")
	c.Globals = g
	c.manifest = m

	// required
	c.CmdClause.Flag("id", "Alphanumeric string identifying a TLS activation").Required().StringVar(&c.id)

	// optional
	c.CmdClause.Flag("include", "Include related objects (comma-separated values)").HintOptions(include...).EnumVar(&c.include, include...)
	c.RegisterFlagBool(c.JSONFlag()) // --json

	return &c
}

// DescribeCommand calls the Fastly API to describe an appropriate resource.
type DescribeCommand struct {
	cmd.Base
	cmd.JSONOutput

	id       string
	include  string
	manifest manifest.Data
}

// Exec invokes the application logic for the command.
func (c *DescribeCommand) Exec(_ io.Reader, out io.Writer) error {
	if c.Globals.Verbose() && c.JSONOutput.Enabled {
		return fsterr.ErrInvalidVerboseJSONCombo
	}

	input := c.constructInput()

	o, err := c.Globals.APIClient.GetTLSActivation(input)
	if err != nil {
		c.Globals.ErrLog.AddWithContext(err, map[string]any{
			"TLS Activation ID": c.id,
		})
		return err
	}

	if ok, err := c.WriteJSON(out, o); ok {
		return err
	}

	return c.print(out, o)
}

// constructInput transforms values parsed from CLI flags into an object to be used by the API client library.
func (c *DescribeCommand) constructInput() *fastly.GetTLSActivationInput {
	var input fastly.GetTLSActivationInput

	input.ID = c.id

	if c.include != "" {
		input.Include = &c.include
	}

	return &input
}

// print displays the information returned from the API.
func (c *DescribeCommand) print(out io.Writer, r *fastly.TLSActivation) error {
	fmt.Fprintf(out, "\nID: %s\n", r.ID)

	if r.CreatedAt != nil {
		fmt.Fprintf(out, "Created at: %s\n", r.CreatedAt)
	}

	return nil
}
