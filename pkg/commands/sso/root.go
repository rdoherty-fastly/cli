package sso

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/fastly/cli/pkg/argparser"
	"github.com/fastly/cli/pkg/auth"
	"github.com/fastly/cli/pkg/config"
	fsterr "github.com/fastly/cli/pkg/errors"
	"github.com/fastly/cli/pkg/global"
	"github.com/fastly/cli/pkg/profile"
	"github.com/fastly/cli/pkg/text"
)

// RootCommand is the parent command for all subcommands in this package.
// It should be installed under the primary root command.
type RootCommand struct {
	argparser.Base
	profile string

	// IMPORTANT: The following fields are public to the `profile` subcommands.

	// InvokedFromProfileCreate indicates if we should create a new profile.
	InvokedFromProfileCreate bool
	// ProfileCreateName indicates the new profile name.
	ProfileCreateName string
	// ProfileDefault indicates if the affected profile should become the default.
	ProfileDefault bool
	// InvokedFromProfileUpdate indicates if we should update a profile.
	InvokedFromProfileUpdate bool
	// ProfileUpdateName indicates the profile name to update.
	ProfileUpdateName string
}

// NewRootCommand returns a new command registered in the parent.
func NewRootCommand(parent argparser.Registerer, g *global.Data) *RootCommand {
	var c RootCommand
	c.Globals = g
	// FIXME: Unhide this command once SSO is GA.
	c.CmdClause = parent.Command("sso", "Single Sign-On authentication").Hidden()
	c.CmdClause.Arg("profile", "Profile to authenticate (i.e. create/update a token for)").Short('p').StringVar(&c.profile)
	return &c
}

// Exec implements the command interface.
func (c *RootCommand) Exec(in io.Reader, out io.Writer) error {
	// We need to prompt the user, so they know we're about to open their web
	// browser, but we also need to handle the scenario where the `sso` command is
	// invoked indirectly via ../../app/run.go as that package will have its own
	// (similar) prompt before invoking this command. So to avoid a double prompt,
	// the app package will set `SkipAuthPrompt: true`.
	if !c.Globals.SkipAuthPrompt && !c.Globals.Flags.AutoYes && !c.Globals.Flags.NonInteractive {
		profileName, _ := c.identifyProfileAndFlow()
		msg := fmt.Sprintf("We're going to authenticate the '%s' profile", profileName)
		text.Important(out, "%s. We need to open your browser to authenticate you.", msg)
		text.Break(out)
		cont, err := text.AskYesNo(out, text.BoldYellow("Do you want to continue? [y/N]: "), in)
		text.Break(out)
		if err != nil {
			return err
		}
		if !cont {
			return fsterr.SkipExitError{
				Skip: true,
				Err:  fsterr.ErrDontContinue,
			}
		}
	}

	var serverErr error
	go func() {
		err := c.Globals.AuthServer.Start()
		if err != nil {
			serverErr = err
		}
	}()
	if serverErr != nil {
		return serverErr
	}

	text.Info(out, "Starting a local server to handle the authentication flow.")

	authorizationURL, err := c.Globals.AuthServer.AuthURL()
	if err != nil {
		return fsterr.RemediationError{
			Inner:       fmt.Errorf("failed to generate an authorization URL: %w", err),
			Remediation: auth.Remediation,
		}
	}

	text.Break(out)
	text.Description(out, "We're opening the following URL in your default web browser so you may authenticate with Fastly", authorizationURL)

	err = c.Globals.Opener(authorizationURL)
	if err != nil {
		return fmt.Errorf("failed to open your default browser: %w", err)
	}

	ar := <-c.Globals.AuthServer.GetResult()
	if ar.Err != nil || ar.SessionToken == "" {
		err := ar.Err
		if ar.Err == nil {
			err = errors.New("no session token")
		}
		return fsterr.RemediationError{
			Inner:       fmt.Errorf("failed to authorize: %w", err),
			Remediation: auth.Remediation,
		}
	}

	err = c.processProfiles(ar)
	if err != nil {
		c.Globals.ErrLog.Add(err)
		return fmt.Errorf("failed to process profile data: %w", err)
	}

	textFn := text.Success
	if c.InvokedFromProfileCreate || c.InvokedFromProfileUpdate {
		textFn = text.Info
	}
	textFn(out, "Session token (persisted to your local configuration): %s", ar.SessionToken)
	return nil
}

// ProfileFlow enumerates which profile flow to take.
type ProfileFlow uint8

const (
	// ProfileNone indicates we need to create a new 'default' profile as no
	// profiles currently exist.
	ProfileNone ProfileFlow = iota

	// ProfileCreate indicates we need to create a new profile using details
	// passed in either from the `sso` or `profile create` command.
	ProfileCreate

	// ProfileUpdate indicates we need to update a profile using details passed in
	// either from the `sso` or `profile update` command.
	ProfileUpdate
)

// identifyProfileAndFlow identifies the profile and the specific workflow.
func (c *RootCommand) identifyProfileAndFlow() (profileName string, flow ProfileFlow) {
	var profileOverride string
	switch {
	case c.Globals.Flags.Profile != "":
		profileOverride = c.Globals.Flags.Profile
	case c.Globals.Manifest.File.Profile != "":
		profileOverride = c.Globals.Manifest.File.Profile
	}

	currentDefaultProfile, _ := profile.Default(c.Globals.Config.Profiles)

	var newDefaultProfile string
	if currentDefaultProfile == "" && len(c.Globals.Config.Profiles) > 0 {
		newDefaultProfile, c.Globals.Config.Profiles = profile.SetADefault(c.Globals.Config.Profiles)
	}

	switch {
	case profileOverride != "":
		return profileOverride, ProfileUpdate
	case c.profile != "":
		return c.profile, ProfileUpdate
	case c.InvokedFromProfileCreate && c.ProfileCreateName != "":
		return c.ProfileCreateName, ProfileCreate
	case c.InvokedFromProfileUpdate && c.ProfileUpdateName != "":
		return c.ProfileUpdateName, ProfileUpdate
	case currentDefaultProfile != "":
		return currentDefaultProfile, ProfileUpdate
	case newDefaultProfile != "":
		return newDefaultProfile, ProfileUpdate
	default:
		return profile.DefaultName, ProfileCreate
	}
}

// processProfiles updates the relevant profile with the returned token data.
//
// First it checks the --profile flag and the `profile` fastly.toml field.
// Second it checks to see which profile is currently the default.
// Third it identifies which profile to be modified.
// Fourth it writes the updated in-memory data back to disk.
func (c *RootCommand) processProfiles(ar auth.AuthorizationResult) error {
	profileName, flow := c.identifyProfileAndFlow()

	switch flow {
	case ProfileCreate:
		c.processCreateProfile(ar, profileName)
	case ProfileUpdate:
		err := c.processUpdateProfile(ar, profileName)
		if err != nil {
			return fmt.Errorf("failed to update profile: %w", err)
		}
	}

	if err := c.Globals.Config.Write(c.Globals.ConfigPath); err != nil {
		return fmt.Errorf("failed to update config file: %w", err)
	}
	return nil
}

// processCreateProfile handles creating a new profile.
func (c *RootCommand) processCreateProfile(ar auth.AuthorizationResult, profileName string) {
	isDefault := true
	if c.InvokedFromProfileCreate {
		isDefault = c.ProfileDefault
	}

	c.Globals.Config.Profiles = createNewProfile(profileName, isDefault, c.Globals.Config.Profiles, ar)

	// If the user wants the newly created profile to be their new default, then
	// we'll call Set for its side effect of resetting all other profiles to have
	// their Default field set to false.
	if c.ProfileDefault { // this is set by the `profile create` command.
		if p, ok := profile.SetDefault(c.ProfileCreateName, c.Globals.Config.Profiles); ok {
			c.Globals.Config.Profiles = p
		}
	}
}

// processUpdateProfile handles updating a profile.
func (c *RootCommand) processUpdateProfile(ar auth.AuthorizationResult, profileName string) error {
	var isDefault bool
	if p := profile.Get(profileName, c.Globals.Config.Profiles); p != nil {
		isDefault = p.Default
	}
	if c.InvokedFromProfileUpdate {
		isDefault = c.ProfileDefault
	}

	ps, err := editProfile(profileName, isDefault, c.Globals.Config.Profiles, ar)
	if err != nil {
		return err
	}
	c.Globals.Config.Profiles = ps
	return nil
}

// IMPORTANT: Mutates the config.Profiles map type.
// We need to return the modified type so it can be safely reassigned.
func createNewProfile(profileName string, makeDefault bool, p config.Profiles, ar auth.AuthorizationResult) config.Profiles {
	now := time.Now().Unix()
	if p == nil {
		p = make(config.Profiles)
	}
	p[profileName] = &config.Profile{
		AccessToken:         ar.Jwt.AccessToken,
		AccessTokenCreated:  now,
		AccessTokenTTL:      ar.Jwt.ExpiresIn,
		Default:             makeDefault,
		Email:               ar.Email,
		RefreshToken:        ar.Jwt.RefreshToken,
		RefreshTokenCreated: now,
		RefreshTokenTTL:     ar.Jwt.RefreshExpiresIn,
		Token:               ar.SessionToken,
	}
	return p
}

// IMPORTANT: Mutates the config.Profiles map type.
// We need to return the modified type so it can be safely reassigned.
func editProfile(profileName string, makeDefault bool, p config.Profiles, ar auth.AuthorizationResult) (config.Profiles, error) {
	ps, ok := profile.Edit(profileName, p, func(p *config.Profile) {
		now := time.Now().Unix()
		p.Default = makeDefault
		p.AccessToken = ar.Jwt.AccessToken
		p.AccessTokenCreated = now
		p.AccessTokenTTL = ar.Jwt.ExpiresIn
		p.Email = ar.Email
		p.RefreshToken = ar.Jwt.RefreshToken
		p.RefreshTokenCreated = now
		p.RefreshTokenTTL = ar.Jwt.RefreshExpiresIn
		p.Token = ar.SessionToken
	})
	if !ok {
		return ps, fsterr.RemediationError{
			Inner:       fmt.Errorf("failed to update '%s' profile with new token data", profileName),
			Remediation: "Run `fastly sso` to retry.",
		}
	}
	return ps, nil
}
