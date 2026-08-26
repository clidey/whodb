package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/clidey/whodb/cli/internal/config"
	platformapi "github.com/clidey/whodb/cli/internal/platform"
	"github.com/clidey/whodb/cli/internal/sdkgen"
)

// sdkCmd groups SDK-related developer tooling.
var sdkCmd = &cobra.Command{
	Use:   "sdk",
	Short: "SDK developer tooling for the hosted WhoDB platform",
}

var (
	sdkGenerateLanguage string
	sdkGenerateOut      string
	sdkGenerateHost     string
	sdkGenerateProject  string
)

var sdkGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a typed SDK client from your project's ontology",
	Long: `Generate typed entity classes and a typed client root from the current
project's ontology, layered over the dynamic WhoDB SDK (@clidey/whodb for
TypeScript, whodb for Python).

The generated file embeds a hash of the ontology it was built from; the SDK
warns at runtime when the live ontology has drifted. Re-run this command after
ontology changes.`,
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: false,
	RunE: func(cmd *cobra.Command, args []string) error {
		language, err := sdkgen.ParseLanguage(sdkGenerateLanguage)
		if err != nil {
			return err
		}
		if strings.TrimSpace(sdkGenerateOut) == "" {
			return fmt.Errorf("--out is required (directory for the generated file)")
		}

		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("cannot load hosted WhoDB config: %w", err)
		}
		hostURL := strings.TrimSpace(sdkGenerateHost)
		if hostURL == "" {
			hostURL = strings.TrimSpace(cfg.Platform.DefaultHost)
		}
		if hostURL == "" {
			hostURL = platformapi.DefaultHost
		}
		hostURL, err = platformapi.NormalizeHost(hostURL)
		if err != nil {
			return err
		}
		host, ok := cfg.GetPlatformHost(hostURL)
		if !ok || strings.TrimSpace(host.AccountID) == "" {
			return fmt.Errorf("not logged in to %s — run: whodb login --host %s", hostURL, hostURL)
		}
		projectID := strings.TrimSpace(sdkGenerateProject)
		if projectID == "" {
			projectID = host.DefaultProjectID
		}
		if projectID == "" {
			return fmt.Errorf("no project selected — run: whodb use, or pass --project")
		}

		tokenSource := platformapi.NewOIDCTokenSource(hostURL, host.AccountID, cfg)
		client, err := platformapi.NewAuthenticatedClient(hostURL, tokenSource)
		if err != nil {
			return err
		}
		client.SetWorkspaceContext(host.DefaultOrgID, projectID)
		manifest, err := client.PlatformManifest(cmd.Context())
		if err != nil {
			return fmt.Errorf("cannot fetch platform manifest: %w", err)
		}
		client.SetPlatformManifest(manifest)

		ontologies, err := client.Ontologies(cmd.Context(), projectID)
		if err != nil {
			return fmt.Errorf("cannot list ontology entities: %w", err)
		}
		model := sdkgen.BuildModel(ontologies)
		written, err := sdkgen.Generate(model, language, sdkGenerateOut)
		if err != nil {
			return err
		}
		for _, path := range written {
			fmt.Fprintf(cmd.OutOrStdout(), "wrote %s (%d entities, ontology hash %s)\n", path, len(model.Entities), model.Hash()[:12])
		}
		return nil
	},
}

func init() {
	sdkGenerateCmd.Flags().StringVar(&sdkGenerateLanguage, "language", "", "output language: ts, python (required)")
	sdkGenerateCmd.Flags().StringVar(&sdkGenerateOut, "out", "", "output directory (required)")
	sdkGenerateCmd.Flags().StringVar(&sdkGenerateHost, "host", "", "hosted WhoDB URL (default app.whodb.com)")
	sdkGenerateCmd.Flags().StringVar(&sdkGenerateProject, "project", "", "project ID (default: workspace from `whodb use`)")
	_ = sdkGenerateCmd.MarkFlagRequired("language")
	_ = sdkGenerateCmd.MarkFlagRequired("out")
	sdkCmd.AddCommand(sdkGenerateCmd)
	rootCmd.AddCommand(sdkCmd)
}
