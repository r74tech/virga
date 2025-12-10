package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/r74tech/virga/internal/shared/logger"
	"gopkg.in/yaml.v3"
)

type NetworkInterface struct {
	Name string
	IP   string
}

func main() {
	updateListeners := flag.Bool("update-listeners", false, "also update listener bind_address values")
	flag.Parse()
	// Get network interfaces
	interfaces := getNetworkInterfaces()
	if len(interfaces) == 0 {
		logger.Error("No network interfaces found")
		os.Exit(1)
	}

	// Add localhost option
	interfaces = append([]NetworkInterface{{Name: "localhost", IP: "127.0.0.1"}}, interfaces...)

	// Show interfaces and get selection
	logger.Info("Available network interfaces:")
	for i, iface := range interfaces {
		logger.Info("%d) %s (%s)", i+1, iface.IP, iface.Name)
	}

	fmt.Print("\nSelect interface number [1]: ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	// Default to first option
	if input == "" {
		input = "1"
	}

	// Parse selection
	selection, err := strconv.Atoi(input)
	if err != nil || selection < 1 || selection > len(interfaces) {
		logger.Error("Invalid selection")
		os.Exit(1)
	}

	selectedIP := interfaces[selection-1].IP
	logger.Info("\nSelected IP: %s", selectedIP)

	// Update configuration files
	if err := updateConfigs(selectedIP, *updateListeners); err != nil {
		logger.Error("Error updating configs: %v", err)
		os.Exit(1)
	}

	logger.Info("\nConfiguration files updated successfully!")
}

func getNetworkInterfaces() []NetworkInterface {
	var interfaces []NetworkInterface

	ifaces, err := net.Interfaces()
	if err != nil {
		return interfaces
	}

	for _, iface := range ifaces {
		// Skip down interfaces
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		// Skip loopback
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// Skip if not IPv4
			if ip == nil || ip.To4() == nil {
				continue
			}

			// Skip link-local addresses
			if ip.IsLinkLocalUnicast() {
				continue
			}

			interfaces = append(interfaces, NetworkInterface{
				Name: iface.Name,
				IP:   ip.String(),
			})
		}
	}

	return interfaces
}

func updateConfigs(ip string, updateListeners bool) error {
	configDir := "configs"

	// Update server.yaml (including listeners section)
	serverConfig := filepath.Join(configDir, "server.yaml")
	if err := updateServerConfig(serverConfig, ip); err != nil {
		return fmt.Errorf("server.yaml: %w", err)
	}

	if updateListeners {
		if err := updateListenerConfig(serverConfig, ip); err != nil {
			logger.Warn("failed to update listeners in server.yaml: %v", err)
		}
	} else {
		logger.Info("Skipping listener updates in server.yaml (use --update-listeners to enable)")
	}

	// Update beacon config files
	beaconFiles, err := filepath.Glob(filepath.Join(configDir, "beacon*.yaml"))
	if err != nil {
		return fmt.Errorf("finding beacon configs: %w", err)
	}

	for _, file := range beaconFiles {
		if err := updateBeaconConfig(file, ip); err != nil {
			logger.Warn("failed to update %s: %v", file, err)
		}
	}

	// Update separate listener config if exists
	listenerConfig := filepath.Join(configDir, "listeners.yaml")
	if _, err := os.Stat(listenerConfig); err == nil {
		if updateListeners {
			if err := updateListenerConfig(listenerConfig, ip); err != nil {
				logger.Warn("failed to update listeners.yaml: %v", err)
			}
		} else {
			logger.Info("Skipping listeners.yaml update (use --update-listeners to enable)")
		}
	}

	return nil
}

func updateServerConfig(path string, ip string) error {
	// Read the YAML file
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Parse YAML into a generic map to preserve all fields
	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return err
	}

	// Update server.host
	if server, ok := config["server"].(map[string]interface{}); ok {
		server["host"] = ip
	}

	// Marshal back to YAML
	updatedData, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(path, updatedData, 0o644)
}

func updateBeaconConfig(path string, ip string) error {
	// Read the YAML file
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Parse YAML into a generic map
	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return err
	}

	// Update c2_url if it exists
	if c2URL, ok := config["c2_url"].(string); ok {
		// Parse the URL and update the host
		u, err := url.Parse(c2URL)
		if err == nil {
			u.Host = fmt.Sprintf("%s:%s", ip, u.Port())
			if u.Port() == "" {
				// If no port specified, use default based on scheme
				if u.Scheme == "https" {
					u.Host = fmt.Sprintf("%s:8443", ip)
				} else {
					u.Host = fmt.Sprintf("%s:8080", ip)
				}
			}
			config["c2_url"] = u.String()
		}
	}

	// Update beacon.c2.host if it exists (nested structure)
	if beacon, ok := config["beacon"].(map[string]interface{}); ok {
		if c2, ok := beacon["c2"].(map[string]interface{}); ok {
			c2["host"] = ip
		}
	}

	// Update c2.host if it exists (nested structure)
	if c2, ok := config["c2"].(map[string]interface{}); ok {
		c2["host"] = ip
	}

	// Marshal back to YAML
	updatedData, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(path, updatedData, 0o644)
}

func updateListenerConfig(path string, ip string) error {
	// Read the YAML file
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Parse YAML into a generic map
	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return err
	}

	// Update bind_address in listeners
	if listeners, ok := config["listeners"].([]interface{}); ok {
		for _, listener := range listeners {
			if l, ok := listener.(map[string]interface{}); ok {
				l["bind_address"] = ip
			}
		}
	}

	// Marshal back to YAML
	updatedData, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(path, updatedData, 0o644)
}
