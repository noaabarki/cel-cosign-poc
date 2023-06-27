package provider

import (
	"context"
	"crypto/x509"
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/sigstore/cosign/cmd/cosign/cli/fulcio"
	"github.com/sigstore/cosign/cmd/cosign/cli/options"
	"github.com/sigstore/cosign/cmd/cosign/cli/rekor"
	"github.com/sigstore/cosign/pkg/cosign"
	"github.com/sigstore/rekor/pkg/generated/client"
)

const (
	apiVersion      = "externaldata.gatekeeper.sh/v1alpha1"
	defaultRekorURL = "https://rekor.sigstore.dev"
)

var (
	rekorClient         *client.Rekor
	fulcioRoots         *x509.CertPool
	fulcioIntermediates *x509.CertPool
	rekorURL            = defaultRekorURL
	registryOptions     = options.RegistryOptions{}
)

func init() {
	// init the Fulcio root
	roots, err := fulcio.GetRoots()
	if err != nil {
		panic(fmt.Sprintf("getting Fulcio root certs: %v", err))
	}
	fulcioRoots = roots

	// init the Fulcio intermediate certs
	intermediates, err := fulcio.GetIntermediates()
	if err != nil {
		panic(fmt.Sprintf("getting Fulcio intermediates certs: %v", err))
	}
	fulcioIntermediates = intermediates

	// init the Rekor client
	rc, err := rekor.NewClient(rekorURL)
	if err != nil {
		panic(fmt.Sprintf("creating Rekor client: %v", err))
	}
	rekorClient = rc
}

// VerifyImages verifies the images using the Rekor client
func VerifyImages(images []string, ctx context.Context) (bool, error) {
	registryClientOpts, err := registryOptions.ClientOpts(ctx)
	if err != nil {
		return false, err
	}

	for _, image := range images {
		ref, err := name.ParseReference(image)
		if err != nil {
			return false, err
		}

		_, bundleVerified, err := cosign.VerifyImageSignatures(ctx, ref, &cosign.CheckOpts{
			RekorClient:        rekorClient,
			RegistryClientOpts: registryClientOpts,
			RootCerts:          fulcioRoots,
			IntermediateCerts:  fulcioIntermediates,
			ClaimVerifier:      cosign.SimpleClaimVerifier,
		})
		if err != nil {
			return false, err
		}

		if !bundleVerified {
			return false, nil
		}
	}

	return true, nil
}
