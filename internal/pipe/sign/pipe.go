package sign

import (
	"fmt"

	"github.com/macreleaser/macreleaser/pkg/context"
	"github.com/macreleaser/macreleaser/pkg/sign"
)

// Pipe executes code signing on the built .app bundle.
type Pipe struct{}

func (Pipe) String() string { return "signing application" }

func (Pipe) Run(ctx *context.Context) error {
	if ctx.Artifacts.AppPath == "" {
		return fmt.Errorf("no .app found to sign — ensure the build step completed successfully")
	}

	identity := ctx.Config.Sign.Identity

	// Validate that the configured identity exists in the keychain
	ctx.Logger.Infof("Validating signing identity: %s", identity)
	if err := sign.CheckIdentityInKeychain(identity); err != nil {
		return fmt.Errorf("identity validation failed: %w", err)
	}

	// Enable Hardened Runtime when notarization is configured and not skipped (Apple requires it)
	hardenedRuntime := !ctx.SkipNotarize && ctx.Config.Notarize.AppleID != ""
	if hardenedRuntime {
		ctx.Logger.Info("Hardened Runtime enabled (required for notarization)")
	}

	// Sign the .app bundle in-place
	ctx.Logger.Infof("Signing %s", ctx.Artifacts.AppPath)
	output, err := sign.RunCodesign(identity, ctx.Artifacts.AppPath, hardenedRuntime)
	if err != nil {
		ctx.Logger.Debug(output)
		return fmt.Errorf("signing failed: %w", err)
	}
	ctx.Logger.Debug(output)

	// Verify the signature
	ctx.Logger.Info("Verifying signature")
	output, err = sign.RunVerify(ctx.Artifacts.AppPath)
	if err != nil {
		ctx.Logger.Debug(output)
		return fmt.Errorf("signature verification failed: %w", err)
	}
	ctx.Logger.Debug(output)

	ctx.Logger.Infof("Signed and verified: %s", ctx.Artifacts.AppPath)
	return nil
}
