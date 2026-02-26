package main

import (
	"fmt"
	"log"
	"os"

	"github.com/chuan/pkg/config"
	"github.com/chuan/pkg/server"
	ctls "github.com/chuan/pkg/tls"
	"github.com/spf13/cobra"
)

func main() {
	var configFile string
	var bindPort int
	var token string
	var certFile string
	var keyFile string

	rootCmd := &cobra.Command{
		Use:   "chuan-server",
		Short: "Chuan tunnel server",
		RunE: func(cmd *cobra.Command, args []string) error {
			var cfg *config.ServerConfig
			if configFile != "" {
				var err error
				cfg, err = config.LoadServerConfig(configFile)
				if err != nil {
					return err
				}
			} else {
				cfg = &config.ServerConfig{
					BindPort:       bindPort,
					Token:          token,
					MaxConnections: 100,
					HTTPPort:       80,
				}
				cfg.TLS.Cert = certFile
				cfg.TLS.Key = keyFile
			}

			// Override with flags if set
			if cmd.Flags().Changed("port") {
				cfg.BindPort = bindPort
			}
			if cmd.Flags().Changed("token") {
				cfg.Token = token
			}

			if cfg.Token == "" {
				return fmt.Errorf("token is required")
			}

			// Auto-generate cert if not exists
			if cfg.TLS.Cert == "" {
				cfg.TLS.Cert = "chuan-server.crt"
				cfg.TLS.Key = "chuan-server.key"
			}
			if _, err := os.Stat(cfg.TLS.Cert); os.IsNotExist(err) {
				log.Println("generating self-signed certificate...")
				if err := ctls.GenerateSelfSignedCert(cfg.TLS.Cert, cfg.TLS.Key); err != nil {
					return fmt.Errorf("generate cert: %w", err)
				}
			}

			s := server.New(cfg)
			return s.Run()
		},
	}

	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "config file path")
	rootCmd.Flags().IntVarP(&bindPort, "port", "p", 7000, "bind port")
	rootCmd.Flags().StringVarP(&token, "token", "t", "", "auth token")
	rootCmd.Flags().StringVar(&certFile, "cert", "", "TLS cert file")
	rootCmd.Flags().StringVar(&keyFile, "key", "", "TLS key file")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
