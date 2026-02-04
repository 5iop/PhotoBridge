package config

import (
	"log"
	"net"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

type Config struct {
	AdminUsername      string
	AdminPassword      string
	APIKey             string
	JWTSecret          string
	Port               string
	UploadDir          string
	DatabasePath       string
	CNCDNURL           string              // China CDN URL (e.g., https://cdn.pb.jangit.me)
	cdnIPSet           map[string]bool     // CDN server IPs (set for O(1) lookup, only grows)
	cdnIPMutex         sync.RWMutex        // Protects cdnIPSet
	TurnstileSiteKey   string              // Cloudflare Turnstile site key (public)
	TurnstileSecretKey string              // Cloudflare Turnstile secret key (private)
}

var AppConfig *Config

const shortname = "[Config]"

func Load() {
	log.Printf("%s Loading configuration", shortname)

	cdnURL := getEnv("CNCDN_URL", "")

	AppConfig = &Config{
		AdminUsername:      getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword:      getEnv("ADMIN_PASSWORD", "admin123"),
		APIKey:             getEnv("API_KEY", "photobridge-api-key"),
		JWTSecret:          getEnv("JWT_SECRET", "photobridge-jwt-secret"),
		Port:               getEnv("PORT", "8060"),
		UploadDir:          getEnv("UPLOAD_DIR", "./uploads"),
		DatabasePath:       getEnv("DATABASE_PATH", "./data/photobridge.db"),
		CNCDNURL:           cdnURL,                           // Optional China CDN URL
		cdnIPSet:           make(map[string]bool),            // Initialize CDN IP set
		TurnstileSiteKey:   getEnv("TURNSTILE_SITE_KEY", ""), // Optional Turnstile site key
		TurnstileSecretKey: getEnv("TURNSTILE_SECRET_KEY", ""), // Optional Turnstile secret key
	}
	log.Printf("%s Configuration loaded - Port: %s, UploadDir: %s, DatabasePath: %s",
		shortname, AppConfig.Port, AppConfig.UploadDir, AppConfig.DatabasePath)

	// Initial CDN IP resolution
	if cdnURL != "" {
		initialIPs := AppConfig.refreshCDNIPs()
		if len(initialIPs) > 0 {
			log.Printf("%s CDN IP whitelist initialized: %v", shortname, initialIPs)
		}

		// Start background goroutine to refresh CDN IPs every 5 seconds
		go AppConfig.startCDNIPRefresher()
	}

	// Ensure upload directory exists
	log.Printf("%s Creating upload directory: %s", shortname, AppConfig.UploadDir)
	if err := os.MkdirAll(AppConfig.UploadDir, 0755); err != nil {
		log.Fatalf("%s Failed to create upload directory %s: %v", shortname, AppConfig.UploadDir, err)
	}
	log.Printf("%s Upload directory created/verified: %s", shortname, AppConfig.UploadDir)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// refreshCDNIPs resolves CDN IPs and adds them to the set (never removes)
// Returns the list of newly added IPs
func (c *Config) refreshCDNIPs() []string {
	if c.CNCDNURL == "" {
		return nil
	}

	// Parse URL to extract hostname
	parsedURL, err := url.Parse(c.CNCDNURL)
	if err != nil {
		log.Printf("%s Failed to parse CNCDN_URL: %v", shortname, err)
		return nil
	}

	hostname := parsedURL.Hostname()
	if hostname == "" {
		log.Printf("%s No hostname found in CNCDN_URL", shortname)
		return nil
	}

	// Resolve hostname to IP addresses
	ips, err := net.LookupIP(hostname)
	if err != nil {
		log.Printf("%s Failed to resolve CDN hostname %s: %v", shortname, hostname, err)
		return nil
	}

	// Add new IPs to the set
	c.cdnIPMutex.Lock()
	defer c.cdnIPMutex.Unlock()

	var newIPs []string
	for _, ip := range ips {
		ipStr := ip.String()
		if !c.cdnIPSet[ipStr] {
			c.cdnIPSet[ipStr] = true
			newIPs = append(newIPs, ipStr)
		}
	}

	return newIPs
}

// startCDNIPRefresher starts a background goroutine to refresh CDN IPs every 5 seconds
func (c *Config) startCDNIPRefresher() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		newIPs := c.refreshCDNIPs()
		if len(newIPs) > 0 {
			log.Printf("%s New CDN IPs discovered: %v", shortname, newIPs)
		}
	}
}

// IsCDNIP checks if the given IP is in the CDN whitelist
func (c *Config) IsCDNIP(ip string) bool {
	// Remove port if present (e.g., "192.168.1.1:12345" -> "192.168.1.1")
	if colonIndex := strings.LastIndex(ip, ":"); colonIndex != -1 {
		// Check if it's IPv6 or IPv4 with port
		if strings.Count(ip, ":") == 1 {
			// IPv4 with port
			ip = ip[:colonIndex]
		}
	}

	c.cdnIPMutex.RLock()
	defer c.cdnIPMutex.RUnlock()

	return c.cdnIPSet[ip]
}

// AddCDNIP manually adds an IP to the CDN whitelist (useful for testing)
func (c *Config) AddCDNIP(ip string) {
	c.cdnIPMutex.Lock()
	defer c.cdnIPMutex.Unlock()

	c.cdnIPSet[ip] = true
}

// InitCDNIPSet initializes the CDN IP set (useful for testing)
func (c *Config) InitCDNIPSet() {
	c.cdnIPMutex.Lock()
	defer c.cdnIPMutex.Unlock()

	if c.cdnIPSet == nil {
		c.cdnIPSet = make(map[string]bool)
	}
}
