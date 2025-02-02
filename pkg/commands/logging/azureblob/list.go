package azureblob

import (
	"fmt"
	"io"

	"github.com/fastly/go-fastly/v8/fastly"

	"github.com/fastly/cli/pkg/argparser"
	fsterr "github.com/fastly/cli/pkg/errors"
	"github.com/fastly/cli/pkg/global"
	"github.com/fastly/cli/pkg/text"
)

// ListCommand calls the Fastly API to list Azure Blob Storage logging endpoints.
type ListCommand struct {
	argparser.Base
	argparser.JSONOutput

	Input          fastly.ListBlobStoragesInput
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
	c.CmdClause = parent.Command("list", "List Azure Blob Storage logging endpoints on a Fastly service version")

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

	o, err := c.Globals.APIClient.ListBlobStorages(&c.Input)
	if err != nil {
		c.Globals.ErrLog.AddWithContext(err, map[string]any{
			"Service ID":      serviceID,
			"Service Version": serviceVersion.Number,
		})
		return err
	}

	if ok, err := c.WriteJSON(out, o); ok {
		return err
	}

	if !c.Globals.Verbose() {
		tw := text.NewTable(out)
		tw.AddHeader("SERVICE", "VERSION", "NAME")
		for _, azureblob := range o {
			tw.AddLine(azureblob.ServiceID, azureblob.ServiceVersion, azureblob.Name)
		}
		tw.Print()
		return nil
	}

	fmt.Fprintf(out, "Version: %d\n", c.Input.ServiceVersion)
	for i, azureblob := range o {
		fmt.Fprintf(out, "\tBlobStorage %d/%d\n", i+1, len(o))
		fmt.Fprintf(out, "\t\tService ID: %s\n", azureblob.ServiceID)
		fmt.Fprintf(out, "\t\tVersion: %d\n", azureblob.ServiceVersion)
		fmt.Fprintf(out, "\t\tName: %s\n", azureblob.Name)
		fmt.Fprintf(out, "\t\tContainer: %s\n", azureblob.Container)
		fmt.Fprintf(out, "\t\tAccount name: %s\n", azureblob.AccountName)
		fmt.Fprintf(out, "\t\tSAS token: %s\n", azureblob.SASToken)
		fmt.Fprintf(out, "\t\tPath: %s\n", azureblob.Path)
		fmt.Fprintf(out, "\t\tPeriod: %d\n", azureblob.Period)
		fmt.Fprintf(out, "\t\tGZip level: %d\n", azureblob.GzipLevel)
		fmt.Fprintf(out, "\t\tFormat: %s\n", azureblob.Format)
		fmt.Fprintf(out, "\t\tFormat version: %d\n", azureblob.FormatVersion)
		fmt.Fprintf(out, "\t\tResponse condition: %s\n", azureblob.ResponseCondition)
		fmt.Fprintf(out, "\t\tMessage type: %s\n", azureblob.MessageType)
		fmt.Fprintf(out, "\t\tTimestamp format: %s\n", azureblob.TimestampFormat)
		fmt.Fprintf(out, "\t\tPlacement: %s\n", azureblob.Placement)
		fmt.Fprintf(out, "\t\tPublic key: %s\n", azureblob.PublicKey)
		fmt.Fprintf(out, "\t\tFile max bytes: %d\n", azureblob.FileMaxBytes)
		fmt.Fprintf(out, "\t\tCompression codec: %s\n", azureblob.CompressionCodec)
	}
	fmt.Fprintln(out)

	return nil
}
