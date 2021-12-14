package imagebundle

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mesosphere/dkp-cli/runtime/cli"
	"github.com/mesosphere/dkp-cli/runtime/cmd/log"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"

	"github.com/mesosphere/mindthegap/config"
	"github.com/mesosphere/mindthegap/docker/registry"
	"github.com/mesosphere/mindthegap/skopeo"
	"github.com/mesosphere/mindthegap/tar"
)

func NewCommand(ioStreams genericclioptions.IOStreams) *cobra.Command {
	var (
		configFile string
		platforms  []platform
		outputFile string
	)

	cmd := &cobra.Command{
		Use: "image-bundle",
		RunE: func(cmd *cobra.Command, args []string) error {
			klog.SetOutput(ioStreams.ErrOut)
			logger := log.NewLogger(ioStreams.ErrOut)
			statusLogger := cli.StatusForLogger(logger)

			statusLogger.Start("Parsing image bundle config")
			cfg, err := config.ParseFile(configFile)
			if err != nil {
				statusLogger.End(false)
				return err
			}
			klog.V(4).Infof("Images config: %+v", cfg)
			statusLogger.End(true)

			statusLogger.Start("Creating temporary directory")
			outputFileAbs, err := filepath.Abs(outputFile)
			if err != nil {
				statusLogger.End(false)
				return fmt.Errorf("failed to determine where to create temporary directory: %w", err)
			}

			tempDir, err := os.MkdirTemp(filepath.Dir(outputFileAbs), ".image-bundle-*")
			if err != nil {
				statusLogger.End(false)
				return fmt.Errorf("failed to create temporary directory: %w", err)
			}
			defer os.RemoveAll(tempDir)
			statusLogger.End(true)

			statusLogger.Start("Starting temporary Docker registry")
			reg, err := registry.NewRegistry(registry.Config{StorageDirectory: tempDir})
			if err != nil {
				statusLogger.End(false)
				return fmt.Errorf("failed to create local Docker registry: %w", err)
			}
			go func() {
				if err := reg.ListenAndServe(); err != nil {
					fmt.Fprintf(ioStreams.ErrOut, "error serving Docker registry: %v\n", err)
					os.Exit(2)
				}
			}()
			statusLogger.End(true)

			skopeoRunner, skopeoCleanup := skopeo.NewRunner()
			defer func() { _ = skopeoCleanup() }()

			for registryName, registryConfig := range cfg {
				var skopeoOpts []skopeo.SkopeoOption
				if registryConfig.TLSVerify != nil && !*registryConfig.TLSVerify {
					skopeoOpts = append(skopeoOpts, skopeo.DisableSrcTLSVerify())
				}
				if registryConfig.Credentials != nil && registryConfig.Credentials.Username != "" {
					skopeoOpts = append(
						skopeoOpts,
						skopeo.SrcCredentials(registryConfig.Credentials.Username, registryConfig.Credentials.Password),
					)
				} else {
					err = skopeoRunner.AttemptToLoginToRegistry(context.TODO(), registryName, klog.V(4).Enabled())
					if err != nil {
						return fmt.Errorf("error logging in to registry: %w", err)
					}
				}
				if klog.V(4).Enabled() {
					skopeoOpts = append(skopeoOpts, skopeo.Debug())
				}
				for imageName, imageTags := range registryConfig.Images {
					for _, imageTag := range imageTags {
						srcImageName := fmt.Sprintf("%s/%s:%s", registryName, imageName, imageTag)
						statusLogger.Start(
							fmt.Sprintf("Copying %s (platforms: %v)",
								srcImageName, platforms,
							),
						)
						for _, p := range platforms {
							skopeoOutput, err := skopeoRunner.Copy(context.TODO(),
								fmt.Sprintf("docker://%s", srcImageName),
								fmt.Sprintf("docker://%s/%s:%s", reg.Address(), imageName, imageTag),
								append(
									skopeoOpts,
									skopeo.DisableDestTLSVerify(), skopeo.OS(p.os), skopeo.Arch(p.arch), skopeo.Variant(p.variant),
								)...,
							)
							if err != nil {
								klog.Info(string(skopeoOutput))
								statusLogger.End(false)
								return err
							}
							klog.V(4).Info(string(skopeoOutput))
							statusLogger.End(true)
						}
					}
				}
			}

			if err := config.WriteSanitizedConfig(cfg, filepath.Join(tempDir, "images.yaml")); err != nil {
				return err
			}

			if err = tar.Tar(tempDir, outputFile); err != nil {
				statusLogger.End(false)
				return fmt.Errorf("failed to create image bundle tarball: %w", err)
			}
			statusLogger.End(true)

			return nil
		},
	}

	cmd.Flags().StringVar(&configFile, "images-file", "", "YAML file containing list of images to create bundle from")
	_ = cmd.MarkFlagRequired("images-file")
	cmd.Flags().Var(newPlatformSlicesValue([]platform{{os: "linux", arch: "amd64"}}, &platforms), "platform",
		"platforms to download images (required format: <os>/<arch>[/<variant>])")
	cmd.Flags().StringVar(&outputFile, "output-file", "images.tar", "Output file to write image bundle to")

	return cmd
}
