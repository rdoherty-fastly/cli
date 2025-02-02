package gcs

import (
	"fmt"
	"io"

	"github.com/fastly/go-fastly/v8/fastly"

	"github.com/fastly/cli/pkg/argparser"
	fsterr "github.com/fastly/cli/pkg/errors"
	"github.com/fastly/cli/pkg/global"
	"github.com/fastly/cli/pkg/text"
)

// ListCommand calls the Fastly API to list GCS logging endpoints.
type ListCommand struct {
	argparser.Base
	argparser.JSONOutput

	Input          fastly.ListGCSsInput
	serviceName    argparser.OptionalServiceNameID
	serviceVersion argparser.OptionalServiceVersion
}

// NewListCommand returns a usable command registered under the parent.
func NewListCommand(parent argparser.Registerer, g *global.Data) *ListCommand {
	c := ListCommand{
		Base: argparser.Base{
			Globals: g,
		},
	}
	c.CmdClause = parent.Command("list", "List GCS endpoints on a Fastly service version")

	// Required.
	c.RegisterFlag(argparser.StringFlagOpts{
		Name:        argparser.FlagVersionName,
		Description: argparser.FlagVersionDesc,
		Dst:         &c.serviceVersion.Value,
		Required:    true,
	})

	// Optional.
	c.RegisterFlagBool(c.JSONFlag()) // --json
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
func (c *ListCommand) Exec(_ io.Reader, out io.Writer) error {
	if c.Globals.Verbose() && c.JSONOutput.Enabled {
		return fsterr.ErrInvalidVerboseJSONCombo
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
			"Service Version": fsterr.ServiceVersion(serviceVersion),
		})
		return err
	}

	c.Input.ServiceID = serviceID
	c.Input.ServiceVersion = serviceVersion.Number

	o, err := c.Globals.APIClient.ListGCSs(&c.Input)
	if err != nil {
		c.Globals.ErrLog.Add(err)
		return err
	}

	if ok, err := c.WriteJSON(out, o); ok {
		return err
	}

	if !c.Globals.Verbose() {
		tw := text.NewTable(out)
		tw.AddHeader("SERVICE", "VERSION", "NAME")
		for _, gcs := range o {
			tw.AddLine(gcs.ServiceID, gcs.ServiceVersion, gcs.Name)
		}
		tw.Print()
		return nil
	}

	fmt.Fprintf(out, "Version: %d\n", c.Input.ServiceVersion)
	for i, gcs := range o {
		fmt.Fprintf(out, "\tGCS %d/%d\n", i+1, len(o))
		fmt.Fprintf(out, "\t\tService ID: %s\n", gcs.ServiceID)
		fmt.Fprintf(out, "\t\tVersion: %d\n", gcs.ServiceVersion)
		fmt.Fprintf(out, "\t\tName: %s\n", gcs.Name)
		fmt.Fprintf(out, "\t\tBucket: %s\n", gcs.Bucket)
		fmt.Fprintf(out, "\t\tUser: %s\n", gcs.User)
		fmt.Fprintf(out, "\t\tAccount name: %s\n", gcs.AccountName)
		fmt.Fprintf(out, "\t\tSecret key: %s\n", gcs.SecretKey)
		fmt.Fprintf(out, "\t\tPath: %s\n", gcs.Path)
		fmt.Fprintf(out, "\t\tPeriod: %d\n", gcs.Period)
		fmt.Fprintf(out, "\t\tGZip level: %d\n", gcs.GzipLevel)
		fmt.Fprintf(out, "\t\tFormat: %s\n", gcs.Format)
		fmt.Fprintf(out, "\t\tFormat version: %d\n", gcs.FormatVersion)
		fmt.Fprintf(out, "\t\tResponse condition: %s\n", gcs.ResponseCondition)
		fmt.Fprintf(out, "\t\tMessage type: %s\n", gcs.MessageType)
		fmt.Fprintf(out, "\t\tTimestamp format: %s\n", gcs.TimestampFormat)
		fmt.Fprintf(out, "\t\tPlacement: %s\n", gcs.Placement)
		fmt.Fprintf(out, "\t\tCompression codec: %s\n", gcs.CompressionCodec)
	}
	fmt.Fprintln(out)

	return nil
}
