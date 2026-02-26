package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/chuan/pkg/client"
	"github.com/chuan/pkg/config"
	"github.com/spf13/cobra"
)

func main() {
	var configFile string
	var serverAddr string
	var token string
	var tlsSkipVerify bool
	var tcpTunnels []string
	var udpTunnels []string
	var httpTunnels []string

	rootCmd := &cobra.Command{
		Use:   "chuan-client",
		Short: "Chuan tunnel client",
		RunE: func(cmd *cobra.Command, args []string) error {
			var cfg *config.ClientConfig
			if configFile != "" {
				var err error
				cfg, err = config.LoadClientConfig(configFile)
				if err != nil {
					return err
				}
			} else {
				cfg = &config.ClientConfig{
					ServerAddr:    serverAddr,
					Token:         token,
					TLSSkipVerify: tlsSkipVerify,
				}
			}

			// Override with flags
			if cmd.Flags().Changed("server") {
				cfg.ServerAddr = serverAddr
			}
			if cmd.Flags().Changed("token") {
				cfg.Token = token
			}
			if cmd.Flags().Changed("tls-skip-verify") {
				cfg.TLSSkipVerify = tlsSkipVerify
			}

			// Parse CLI tunnel flags
			for _, t := range tcpTunnels {
				tc, err := parseTunnelFlag("tcp", t)
				if err != nil {
					return err
				}
				cfg.Tunnels = append(cfg.Tunnels, tc)
			}
			for _, t := range udpTunnels {
				tc, err := parseTunnelFlag("udp", t)
				if err != nil {
					return err
				}
				cfg.Tunnels = append(cfg.Tunnels, tc)
			}
			for _, t := range httpTunnels {
				tc, err := parseHTTPTunnelFlag(t)
				if err != nil {
					return err
				}
				cfg.Tunnels = append(cfg.Tunnels, tc)
			}

			if cfg.ServerAddr == "" {
				return fmt.Errorf("server address is required")
			}
			if cfg.Token == "" {
				return fmt.Errorf("token is required")
			}
			if len(cfg.Tunnels) == 0 {
				return fmt.Errorf("at least one tunnel is required")
			}

			c := client.New(cfg)
			return c.Run()
		},
	}

	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "config file path")
	rootCmd.Flags().StringVarP(&serverAddr, "server", "s", "", "server address (host:port)")
	rootCmd.Flags().StringVarP(&token, "token", "t", "", "auth token")
	rootCmd.Flags().BoolVar(&tlsSkipVerify, "tls-skip-verify", false, "skip TLS verification")
	rootCmd.Flags().StringArrayVar(&tcpTunnels, "tcp", nil, "TCP tunnel (local_port:remote_port)")
	rootCmd.Flags().StringArrayVar(&udpTunnels, "udp", nil, "UDP tunnel (local_port:remote_port)")
	rootCmd.Flags().StringArrayVar(&httpTunnels, "http", nil, "HTTP tunnel (local_port:domain)")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// parseTunnelFlag parses "local_port:remote_port" format
func parseTunnelFlag(tunnelType, flag string) (config.TunnelConfig, error) {
	parts := strings.SplitN(flag, ":", 2)
	if len(parts) != 2 {
		return config.TunnelConfig{}, fmt.Errorf("invalid tunnel format: %s (expected local_port:remote_port)", flag)
	}
	local, err := strconv.Atoi(parts[0])
	if err != nil {
		return config.TunnelConfig{}, fmt.Errorf("invalid local port: %s", parts[0])
	}
	remote, err := strconv.Atoi(parts[1])
	if err != nil {
		return config.TunnelConfig{}, fmt.Errorf("invalid remote port: %s", parts[1])
	}
	return config.TunnelConfig{
		Name:       fmt.Sprintf("%s-%d", tunnelType, local),
		Type:       tunnelType,
		LocalPort:  local,
		RemotePort: remote,
	}, nil
}

// parseHTTPTunnelFlag parses "local_port:domain" format
func parseHTTPTunnelFlag(flag string) (config.TunnelConfig, error) {
	parts := strings.SplitN(flag, ":", 2)
	if len(parts) != 2 {
		return config.TunnelConfig{}, fmt.Errorf("invalid http tunnel format: %s (expected local_port:domain)", flag)
	}
	local, err := strconv.Atoi(parts[0])
	if err != nil {
		return config.TunnelConfig{}, fmt.Errorf("invalid local port: %s", parts[0])
	}
	return config.TunnelConfig{
		Name:      fmt.Sprintf("http-%s", parts[1]),
		Type:      "http",
		LocalPort: local,
		Domain:    parts[1],
	}, nil
}
