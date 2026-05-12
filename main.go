package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

type ServiceConfig struct {
	Name   string `yaml:"name"`
	Path   string `yaml:"path"`
	Target string `yaml:"target"`
}

type Config struct {
	Port     string          `yaml:"port"`
	Services []ServiceConfig `yaml:"services"`
}

func loadConfig() (*Config, error) {
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	return &config, err
}

func createProxy(target string) gin.HandlerFunc {
	targetURL, _ := url.Parse(target)
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Service unavailable: %v", err)
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`{"error": "Service unavailable"}`))
	}

	return func(c *gin.Context) {
		c.Request.URL.Path = c.Param("proxyPath")
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func main() {
	config, err := loadConfig()
	if err != nil {
		log.Fatal("Cannot load config:", err)
	}

	router := gin.Default()

	// Gateway info
	router.GET("/", func(c *gin.Context) {
		services := []string{}
		for _, svc := range config.Services {
			services = append(services, svc.Path)
		}
		c.JSON(http.StatusOK, gin.H{
			"gateway":  "API Gateway",
			"services": services,
		})
	})

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Auto register services từ config
	for _, svc := range config.Services {
		path := svc.Path + "/*proxyPath"
		router.Any(path, createProxy(svc.Target))
		log.Printf("✓ %s → %s", svc.Path, svc.Target)
	}

	log.Printf("🚀 Gateway started on :%s", config.Port)
	router.Run(":" + config.Port)
}
