package https_test

import (
	"bytes"
	"testing"

	"github.com/fastly/go-fastly/v8/fastly"

	"github.com/fastly/cli/pkg/argparser"
	"github.com/fastly/cli/pkg/commands/logging/https"
	"github.com/fastly/cli/pkg/config"
	"github.com/fastly/cli/pkg/errors"
	"github.com/fastly/cli/pkg/global"
	"github.com/fastly/cli/pkg/manifest"
	"github.com/fastly/cli/pkg/mock"
	"github.com/fastly/cli/pkg/testutil"
)

func TestCreateHTTPSInput(t *testing.T) {
	for _, testcase := range []struct {
		name      string
		cmd       *https.CreateCommand
		want      *fastly.CreateHTTPSInput
		wantError string
	}{
		{
			name: "required values set flag serviceID",
			cmd:  createCommandRequired(),
			want: &fastly.CreateHTTPSInput{
				ServiceID:      "123",
				ServiceVersion: 4,
				Name:           fastly.String("log"),
				URL:            fastly.String("example.com"),
			},
		},
		{
			name: "all values set flag serviceID",
			cmd:  createCommandAll(),
			want: &fastly.CreateHTTPSInput{
				ServiceID:         "123",
				ServiceVersion:    4,
				Name:              fastly.String("logs"),
				ResponseCondition: fastly.String("Prevent default logging"),
				Format:            fastly.String(`%h %l %u %t "%r" %>s %b`),
				URL:               fastly.String("example.com"),
				RequestMaxEntries: fastly.Int(2),
				RequestMaxBytes:   fastly.Int(2),
				ContentType:       fastly.String("application/json"),
				HeaderName:        fastly.String("name"),
				HeaderValue:       fastly.String("value"),
				Method:            fastly.String("GET"),
				JSONFormat:        fastly.String("1"),
				Placement:         fastly.String("none"),
				TLSCACert:         fastly.String("-----BEGIN CERTIFICATE-----foo"),
				TLSClientCert:     fastly.String("-----BEGIN CERTIFICATE-----bar"),
				TLSClientKey:      fastly.String("-----BEGIN PRIVATE KEY-----bar"),
				TLSHostname:       fastly.String("example.com"),
				MessageType:       fastly.String("classic"),
				FormatVersion:     fastly.Int(2),
			},
		},
		{
			name:      "error missing serviceID",
			cmd:       createCommandMissingServiceID(),
			want:      nil,
			wantError: errors.ErrNoServiceID.Error(),
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			var bs []byte
			out := bytes.NewBuffer(bs)
			verboseMode := true

			serviceID, serviceVersion, err := argparser.ServiceDetails(argparser.ServiceDetailsOpts{
				AutoCloneFlag:      testcase.cmd.AutoClone,
				APIClient:          testcase.cmd.Globals.APIClient,
				Manifest:           testcase.cmd.Manifest,
				Out:                out,
				ServiceVersionFlag: testcase.cmd.ServiceVersion,
				VerboseMode:        verboseMode,
			})

			switch {
			case err != nil && testcase.wantError == "":
				t.Fatalf("unexpected error getting service details: %v", err)
				return
			case err != nil && testcase.wantError != "":
				testutil.AssertErrorContains(t, err, testcase.wantError)
				return
			case err == nil && testcase.wantError != "":
				t.Fatalf("expected error, have nil (service details: %s, %d)", serviceID, serviceVersion.Number)
			case err == nil && testcase.wantError == "":
				have, err := testcase.cmd.ConstructInput(serviceID, serviceVersion.Number)
				testutil.AssertErrorContains(t, err, testcase.wantError)
				testutil.AssertEqual(t, testcase.want, have)
			}
		})
	}
}

func TestUpdateHTTPSInput(t *testing.T) {
	scenarios := []struct {
		name      string
		cmd       *https.UpdateCommand
		api       mock.API
		want      *fastly.UpdateHTTPSInput
		wantError string
	}{
		{
			name: "all values set flag serviceID",
			cmd:  updateCommandAll(),
			api: mock.API{
				ListVersionsFn: testutil.ListVersions,
				CloneVersionFn: testutil.CloneVersionResult(4),
				GetHTTPSFn:     getHTTPSOK,
			},
			want: &fastly.UpdateHTTPSInput{
				ServiceID:         "123",
				ServiceVersion:    4,
				Name:              "log",
				NewName:           fastly.String("new1"),
				ResponseCondition: fastly.String("new2"),
				Format:            fastly.String("new3"),
				URL:               fastly.String("new4"),
				RequestMaxEntries: fastly.Int(3),
				RequestMaxBytes:   fastly.Int(3),
				ContentType:       fastly.String("new5"),
				HeaderName:        fastly.String("new6"),
				HeaderValue:       fastly.String("new7"),
				Method:            fastly.String("new8"),
				JSONFormat:        fastly.String("new9"),
				Placement:         fastly.String("new10"),
				TLSCACert:         fastly.String("new11"),
				TLSClientCert:     fastly.String("new12"),
				TLSClientKey:      fastly.String("new13"),
				TLSHostname:       fastly.String("new14"),
				MessageType:       fastly.String("new15"),
				FormatVersion:     fastly.Int(3),
			},
		},
		{
			name: "no updates",
			cmd:  updateCommandNoUpdates(),
			api: mock.API{
				ListVersionsFn: testutil.ListVersions,
				CloneVersionFn: testutil.CloneVersionResult(4),
				GetHTTPSFn:     getHTTPSOK,
			},
			want: &fastly.UpdateHTTPSInput{
				ServiceID:      "123",
				ServiceVersion: 4,
				Name:           "log",
			},
		},
		{
			name:      "error missing serviceID",
			cmd:       updateCommandMissingServiceID(),
			want:      nil,
			wantError: errors.ErrNoServiceID.Error(),
		},
	}
	for testcaseIdx := range scenarios {
		testcase := &scenarios[testcaseIdx]
		t.Run(testcase.name, func(t *testing.T) {
			testcase.cmd.Globals.APIClient = testcase.api

			var bs []byte
			out := bytes.NewBuffer(bs)
			verboseMode := true

			serviceID, serviceVersion, err := argparser.ServiceDetails(argparser.ServiceDetailsOpts{
				AutoCloneFlag:      testcase.cmd.AutoClone,
				APIClient:          testcase.api,
				Manifest:           testcase.cmd.Manifest,
				Out:                out,
				ServiceVersionFlag: testcase.cmd.ServiceVersion,
				VerboseMode:        verboseMode,
			})

			switch {
			case err != nil && testcase.wantError == "":
				t.Fatalf("unexpected error getting service details: %v", err)
				return
			case err != nil && testcase.wantError != "":
				testutil.AssertErrorContains(t, err, testcase.wantError)
				return
			case err == nil && testcase.wantError != "":
				t.Fatalf("expected error, have nil (service details: %s, %d)", serviceID, serviceVersion.Number)
			case err == nil && testcase.wantError == "":
				have, err := testcase.cmd.ConstructInput(serviceID, serviceVersion.Number)
				testutil.AssertErrorContains(t, err, testcase.wantError)
				testutil.AssertEqual(t, testcase.want, have)
			}
		})
	}
}

func createCommandRequired() *https.CreateCommand {
	var b bytes.Buffer

	g := global.Data{
		Config: config.File{},
		Env:    config.Environment{},
		Output: &b,
	}
	g.APIClient, _ = mock.APIClient(mock.API{
		ListVersionsFn: testutil.ListVersions,
		CloneVersionFn: testutil.CloneVersionResult(4),
	})("token", "endpoint", false)

	return &https.CreateCommand{
		Base: argparser.Base{
			Globals: &g,
		},
		Manifest: manifest.Data{
			Flag: manifest.Flag{
				ServiceID: "123",
			},
		},
		ServiceVersion: argparser.OptionalServiceVersion{
			OptionalString: argparser.OptionalString{Value: "1"},
		},
		AutoClone: argparser.OptionalAutoClone{
			OptionalBool: argparser.OptionalBool{
				Optional: argparser.Optional{
					WasSet: true,
				},
				Value: true,
			},
		},
		EndpointName: argparser.OptionalString{Optional: argparser.Optional{WasSet: true}, Value: "log"},
		URL:          argparser.OptionalString{Optional: argparser.Optional{WasSet: true}, Value: "example.com"},
	}
}

func createCommandAll() *https.CreateCommand {
	var b bytes.Buffer

	g := global.Data{
		Config: config.File{},
		Env:    config.Environment{},
		Output: &b,
	}
	g.APIClient, _ = mock.APIClient(mock.API{
		ListVersionsFn: testutil.ListVersions,
		CloneVersionFn: testutil.CloneVersionResult(4),
	})("token", "endpoint", false)

	return &https.CreateCommand{
		Base: argparser.Base{
			Globals: &g,
		},
		Manifest: manifest.Data{
			Flag: manifest.Flag{
				ServiceID: "123",
			},
		},
		ServiceVersion: argparser.OptionalServiceVersion{
			OptionalString: argparser.OptionalString{Value: "1"},
		},
		AutoClone: argparser.OptionalAutoClone{
			OptionalBool: argparser.OptionalBool{
				Optional: argparser.Optional{
					WasSet: true,
				},
				Value: true,
			},
		},
		EndpointName:      argparser.OptionalString{Optional: argparser.Optional{WasSet: true}, Value: "logs"},
		URL:               argparser.OptionalString{Optional: argparser.Optional{WasSet: true}, Value: "example.com"},
		ContentType:       argparser.OptionalString{Optional: argparser.Optional{WasSet: true}, Value: "application/json"},
		HeaderName:        argparser.OptionalString{Optional: argparser.Optional{WasSet: true}, Value: "name"},
		HeaderValue:       argparser.OptionalString{Optional: argparser.Optional{WasSet: true}, Value: "value"},
		Method:            argparser.OptionalString{Optional: argparser.Optional{WasSet: true}, Value: "GET"},
		JSONFormat:        argparser.OptionalString{Optional: argparser.Optional{WasSet: true}, Value: "1"},
		MessageType:       argparser.OptionalString{Optional: argparser.Optional{WasSet: true}, Value: "classic"},
		RequestMaxEntries: argparser.OptionalInt{Optional: argparser.Optional{WasSet: true}, Value: 2},
		RequestMaxBytes:   argparser.OptionalInt{Optional: argparser.Optional{WasSet: true}, Value: 2},
		Format:            argparser.OptionalString{Optional: argparser.Optional{WasSet: true}, Value: `%h %l %u %t "%r" %>s %b`},
		FormatVersion:     argparser.OptionalInt{Optional: argparser.Optional{WasSet: true}, Value: 2},
		ResponseCondition: argparser.OptionalString{Optional: argparser.Optional{WasSet: true}, Value: "Prevent default logging"},
		Placement:         argparser.OptionalString{Optional: argparser.Optional{WasSet: true}, Value: "none"},
		TLSCACert:         argparser.OptionalString{Optional: argparser.Optional{WasSet: true}, Value: "-----BEGIN CERTIFICATE-----foo"},
		TLSHostname:       argparser.OptionalString{Optional: argparser.Optional{WasSet: true}, Value: "example.com"},
		TLSClientCert:     argparser.OptionalString{Optional: argparser.Optional{WasSet: true}, Value: "-----BEGIN CERTIFICATE-----bar"},
		TLSClientKey:      argparser.OptionalString{Optional: argparser.Optional{WasSet: true}, Value: "-----BEGIN PRIVATE KEY-----bar"},
	}
}

func createCommandMissingServiceID() *https.CreateCommand {
	res := createCommandAll()
	res.Manifest = manifest.Data{}
	return res
}

func updateCommandNoUpdates() *https.UpdateCommand {
	var b bytes.Buffer

	g := global.Data{
		Config: config.File{},
		Env:    config.Environment{},
		Output: &b,
	}

	return &https.UpdateCommand{
		Base: argparser.Base{
			Globals: &g,
		},
		Manifest: manifest.Data{
			Flag: manifest.Flag{
				ServiceID: "123",
			},
		},
		EndpointName: "log",
		ServiceVersion: argparser.OptionalServiceVersion{
			OptionalString: argparser.OptionalString{Value: "1"},
		},
		AutoClone: argparser.OptionalAutoClone{
			OptionalBool: argparser.OptionalBool{
				Optional: argparser.Optional{
					WasSet: true,
				},
				Value: true,
			},
		},
	}
}

func updateCommandAll() *https.UpdateCommand {
	var b bytes.Buffer

	g := global.Data{
		Config: config.File{},
		Env:    config.Environment{},
		Output: &b,
	}

	return &https.UpdateCommand{
		Base: argparser.Base{
			Globals: &g,
		},
		Manifest: manifest.Data{
			Flag: manifest.Flag{
				ServiceID: "123",
			},
		},
		EndpointName: "log",
		ServiceVersion: argparser.OptionalServiceVersion{
			OptionalString: argparser.OptionalString{Value: "1"},
		},
		AutoClone: argparser.OptionalAutoClone{
			OptionalBool: argparser.OptionalBool{
				Optional: argparser.Optional{
					WasSet: true,
				},
				Value: true,
			},
		},
		NewName:           argparser.OptionalString{Optional: argparser.Optional{WasSet: true}, Value: "new1"},
		ResponseCondition: argparser.OptionalString{Optional: argparser.Optional{WasSet: true}, Value: "new2"},
		Format:            argparser.OptionalString{Optional: argparser.Optional{WasSet: true}, Value: "new3"},
		URL:               argparser.OptionalString{Optional: argparser.Optional{WasSet: true}, Value: "new4"},
		ContentType:       argparser.OptionalString{Optional: argparser.Optional{WasSet: true}, Value: "new5"},
		HeaderName:        argparser.OptionalString{Optional: argparser.Optional{WasSet: true}, Value: "new6"},
		HeaderValue:       argparser.OptionalString{Optional: argparser.Optional{WasSet: true}, Value: "new7"},
		Method:            argparser.OptionalString{Optional: argparser.Optional{WasSet: true}, Value: "new8"},
		JSONFormat:        argparser.OptionalString{Optional: argparser.Optional{WasSet: true}, Value: "new9"},
		Placement:         argparser.OptionalString{Optional: argparser.Optional{WasSet: true}, Value: "new10"},
		RequestMaxEntries: argparser.OptionalInt{Optional: argparser.Optional{WasSet: true}, Value: 3},
		RequestMaxBytes:   argparser.OptionalInt{Optional: argparser.Optional{WasSet: true}, Value: 3},
		TLSCACert:         argparser.OptionalString{Optional: argparser.Optional{WasSet: true}, Value: "new11"},
		TLSClientCert:     argparser.OptionalString{Optional: argparser.Optional{WasSet: true}, Value: "new12"},
		TLSClientKey:      argparser.OptionalString{Optional: argparser.Optional{WasSet: true}, Value: "new13"},
		TLSHostname:       argparser.OptionalString{Optional: argparser.Optional{WasSet: true}, Value: "new14"},
		MessageType:       argparser.OptionalString{Optional: argparser.Optional{WasSet: true}, Value: "new15"},
		FormatVersion:     argparser.OptionalInt{Optional: argparser.Optional{WasSet: true}, Value: 3},
	}
}

func updateCommandMissingServiceID() *https.UpdateCommand {
	res := updateCommandAll()
	res.Manifest = manifest.Data{}
	return res
}
