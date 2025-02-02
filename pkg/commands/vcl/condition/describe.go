package condition

import (
	"fmt"
	"io"

	"github.com/fastly/go-fastly/v8/fastly"

	"github.com/fastly/cli/pkg/argparser"
	"github.com/fastly/cli/pkg/errors"
	"github.com/fastly/cli/pkg/global"
)

// DescribeCommand calls the Fastly API to describe an appropriate resource.
type DescribeCommand struct {
	argparser.Base
	argparser.JSONOutput
	name           string
	serviceName    argparser.OptionalServiceNameID
	serviceVersion argparser.OptionalServiceVersion
}

// NewDescribeCommand returns a usable command registered under the parent.
func NewDescribeCommand(parent argparser.Registerer, g *global.Data) *DescribeCommand {
	var c DescribeCommand
	c.CmdClause = parent.Command("describe", "Show detailed information about a condition on a Fastly service version").Alias("get")
	c.Globals = g

	// Required flags
	c.CmdClause.Flag("name", "Name of condition").Short('n').Required().StringVar(&c.name)

	c.RegisterFlag(argparser.StringFlagOpts{
		Name:        argparser.FlagVersionName,
		Description: argparser.FlagVersionDesc,
		Dst:         &c.serviceVersion.Value,
		Required:    true,
	})

	// Optional flags
	c.RegisterFlagBool(c.JSONFlag())
	c.RegisterFlag(argparser.StringFlagOpts{
		Name:        argparser.FlagServiceIDName,
		Description: argparser.FlagServiceIDDesc,
		Dst:         &g.Manifest.Flag.ServiceID,
		Short:       's',
	})
	c.RegisterFlag(argparser.StringFlagOpts{
		Action:      c.serviceName.Set,
		Name:        argparser.FlagServiceName,
		Description: argparser.FlagServiceDesc,
		Dst:         &c.serviceName.Value,
	})

	return &c
}

// Exec invokes the application logic for the command.
func (c *DescribeCommand) Exec(_ io.Reader, out io.Writer) error {
	if c.Globals.Verbose() && c.JSONOutput.Enabled {
		return errors.ErrInvalidVerboseJSONCombo
	}

	serviceID, serviceVersion, err := argparser.ServiceDetails(argparser.ServiceDetailsOpts{
		AllowActiveLocked:  true,
		APIClient:          c.Globals.APIClient,
		Manifest:           *c.Globals.Manifest,
		Out:                out,
		ServiceNameFlag:    c.serviceName,
		ServiceVersionFlag: c.serviceVersion,
		VerboseMode:        c.Globals.Flags.Verbose,
	})
	if err != nil {
		c.Globals.ErrLog.AddWithContext(err, map[string]any{
			"Service ID":      serviceID,
			"Service Version": errors.ServiceVersion(serviceVersion),
		})
		return err
	}

	var input fastly.GetConditionInput
	input.ServiceID = serviceID
	input.ServiceVersion = serviceVersion.Number
	input.Name = c.name

	r, err := c.Globals.APIClient.GetCondition(&input)
	if err != nil {
		c.Globals.ErrLog.AddWithContext(err, map[string]any{
			"Service ID":      serviceID,
			"Service Version": serviceVersion.Number,
		})
		return err
	}

	if ok, err := c.WriteJSON(out, r); ok {
		return err
	}

	if !c.Globals.Verbose() {
		fmt.Fprintf(out, "\nService ID: %s\n", r.ServiceID)
	}
	fmt.Fprintf(out, "Version: %d\n", r.ServiceVersion)
	fmt.Fprintf(out, "Name: %s\n", r.Name)
	fmt.Fprintf(out, "Statement: %s\n", r.Statement)
	fmt.Fprintf(out, "Type: %s\n", r.Type)
	fmt.Fprintf(out, "Priority: %d\n", r.Priority)

	return nil
}
