/**
 * @author Dmitry Vovk <dmitry.vovk@gmail.com>
 * @copyright 2014
 */
package conf

import (
	"log"
	"io/ioutil"
	"encoding/json"
	"errors"
	"regexp"
	"net"
	"strconv"
	netUrl "net/url"
)

type Url struct {
	Source    string `json:"source"`    // IP:port of stream source
	Interface string `json:"interface"` // NIC name
	Set       uint `json:"set"`         // Set id
	Networks  []net.IPNet               // Allowed networks (based on set values in networks config)
}

type UrlConfig map[string]Url

var MaxMTU int = 1500

const (
	VALID_PATH = `^/[a-z0-9_-]+$`
)

// Read and parse JSON sources config
func ReadUrls(fileName *string) (UrlConfig, error) {
	log.Print("Reading config")
	file, err := ioutil.ReadFile(*fileName)
	if err != nil {
		log.Printf("Could not read config: %s", err)
		return UrlConfig{}, errors.New("Could not read config")
	}
	var config UrlConfig
	err = json.Unmarshal(file, &config)
	if err != nil {
		log.Printf("Could not parse config: %s", err)
		return UrlConfig{}, errors.New("Could not parse config")
	}
	log.Printf("Read %d records", len(config))
	if configValid(config) {
		return config, nil
	} else {
		return UrlConfig{}, errors.New("Config is not valid")
	}
}

// Check config for validity
func configValid(config UrlConfig) bool {
	// Get list of network interfaces
	var ifaceNames map[string]bool
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Printf("Could not get interfaces list: %s", err)
		return false
	}
	ifaceNames = make(map[string]bool, len(interfaces))
	for _, iface := range interfaces {
		ifaceNames[iface.Name] = true
		// Find out biggest MTU
		if iface.MTU > MaxMTU {
			MaxMTU = iface.MTU
		}
	}
	var validPath = regexp.MustCompile(VALID_PATH)
	for path, url := range config {
		if !validPath.MatchString(path) {
			log.Printf("Invalid path found: %s", path)
			return false
		}
		urlParts, err := netUrl.Parse(url.Source)
		if err != nil {
			log.Printf("Could not parse source url %s", url.Source)
			return false
		}
		if urlParts.Scheme == "udp" {
			// url.Source was like udp://123.4.5.6:123
			host, port, err := net.SplitHostPort(urlParts.Host)
			if err != nil {
				log.Printf("Could not parse host:port source pair of path %s: %s", path, urlParts.Host)
				return false
			}
			ipAddr := net.ParseIP(host)
			if ipAddr == nil {
				log.Printf("Invalid ip address in source %s: %s", path, host)
				return false
			}
			dPort, err := strconv.Atoi(port)
			if dPort == 0 || err != nil {
				log.Printf("Invalid port in source %s: %s", path, port)
				return false
			}
			if _, ok := ifaceNames[url.Interface]; !ok {
				log.Printf("Interface for source %s not found: %s", path, url.Interface)
				return false
			}
		} else if urlParts.Scheme == "http" {
			// Anything will work if URL was parsed with http scheme
		} else {
			log.Printf("Invalid address in source %s: %s", path, url.Source)
			return false
		}
	}
	return true
}

func GetScheme(url string) string {
	parsedUrl, err := netUrl.Parse(url)
	if err == nil {
		return parsedUrl.Scheme
	}
	return ""
}

func GetHost(url string) string {
	parsedUrl, err := netUrl.Parse(url)
	if err == nil {
		return parsedUrl.Host
	}
	return ""
}
