package alfresco

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"text/template"

	"github.com/aborroy/alf-cli/internal/util"
	"github.com/aborroy/alf-cli/ui/selector"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Configuration holds all the Docker Compose configuration options
type Configuration struct {
	Version          string
	RAM              int64 // RAM in GB
	CPUs             int64
	HTTPS            bool
	Server           string
	AdminPassword    string
	Database         string
	DbPassword       string
	Port             string
	UseBinding       bool
	BindingIP        string
	UseFtp           bool
	FtpBindingIP     string
	IndexCrossLocale bool
	IndexContent     bool
	SolrComm         string
	Secret           string
	UseActiveMQ      bool
	AmqUser          string
	AmqPassword      string
	Addons           []string
	UseDockerVolume  bool
	Resources        map[string]util.Resource
}

var flags Configuration

// Available addon options
var availableAddons = []selector.Option{
	{Code: "ootbee-support-tools", Description: "Order of the Bee Support Tools 1.2.2.0"},
	{Code: "share-site-creators", Description: "Share Site Creators 0.0.8"},
	{Code: "share-site-space-templates", Description: "Share Site Space Templates 1.1.4-SNAPSHOT"},
	{Code: "alf-tengine-ocr", Description: "Alfresco OCR Transformer 1.1.0"},
	{Code: "esign-cert", Description: "ESign Cert 1.8.4"},
	{Code: "share-online-edition", Description: "Edit with LibreOffice in Alfresco Share 0.3."},
}

var dockerComposeCmd = &cobra.Command{
	Use:   "docker-compose",
	Short: "Docker Compose commands for Alfresco",
	RunE:  runDockerCompose,
}

func runDockerCompose(cmd *cobra.Command, args []string) error {

	config, err := buildConfiguration(cmd)
	if err != nil {
		return fmt.Errorf("failed to build configuration: %w", err)
	}

	if err := generateConfigFiles(config); err != nil {
		return fmt.Errorf("failed to generate config file: %w", err)
	}

	return nil
}

func buildConfiguration(cmd *cobra.Command) (*Configuration, error) {
	cmdFlags := cmd.Flags()
	config := &Configuration{}

	// Detect system resources allocated for Docker
	detector := util.NewDockerResourceDetector()
	sysInfo, _ := detector.GetSystemInfo()
	fmt.Printf("Detected resources available for Docker: CPUs=%d, RAM (in GB)=%d\n", sysInfo.CPUCount, sysInfo.RAMGB)
	config.CPUs = sysInfo.CPUCount
	config.RAM = sysInfo.RAMGB
	if config.RAM < 8 {
		return nil, fmt.Errorf("insufficient RAM: %d GB detected, at least 8 GB is recommended", config.RAM)
	}

	// Calculate resources allocation for each service
	totalMiB := int64(config.RAM * 1024)
	scaled, err := util.Scale(totalMiB, float64(config.CPUs))
	if err != nil {
		return nil, err
	}
	config.Resources = scaled

	// Build configuration step by step
	if err := setVersion(config, cmdFlags); err != nil {
		return nil, err
	}
	if err := setHTTPS(config, cmdFlags); err != nil {
		return nil, err
	}
	if err := setServer(config, cmdFlags); err != nil {
		return nil, err
	}
	if err := setPassword(config, cmdFlags); err != nil {
		return nil, err
	}
	if err := setPort(config, cmdFlags); err != nil {
		return nil, err
	}
	if err := setBinding(config, cmdFlags); err != nil {
		return nil, err
	}
	if err := setFTP(config, cmdFlags); err != nil {
		return nil, err
	}
	if err := setDatabase(config, cmdFlags); err != nil {
		return nil, err
	}
	if err := setIndexing(config, cmdFlags); err != nil {
		return nil, err
	}
	if err := setSolr(config, cmdFlags); err != nil {
		return nil, err
	}
	if err := setActiveMQ(config, cmdFlags); err != nil {
		return nil, err
	}
	if err := setAddons(config, cmdFlags); err != nil {
		return nil, err
	}
	if err := setDockerVolume(config, cmdFlags); err != nil {
		return nil, err
	}

	return config, nil
}
func setVersion(config *Configuration, cmdFlags *pflag.FlagSet) error {
	if cmdFlags.Changed("version") {
		config.Version = flags.Version
		return nil
	}

	version, err := selector.RunSelector(
		"Which ACS version do you want to use?",
		[]string{"25.2", "25.1"},
	)
	if err != nil {
		return err
	}
	config.Version = version
	return nil
}
func setHTTPS(config *Configuration, cmdFlags *pflag.FlagSet) error {
	if cmdFlags.Changed("https") {
		config.HTTPS = flags.HTTPS
		return nil
	}

	https, err := selector.RunYesNoSelector("Do you want to enable HTTPS?", false)
	if err != nil {
		return err
	}
	config.HTTPS = https
	return nil
}
func setServer(config *Configuration, cmdFlags *pflag.FlagSet) error {
	if cmdFlags.Changed("server") {
		config.Server = flags.Server
		return nil
	}

	server, err := selector.RunTextInput("What is the name of your server?", "localhost")
	if err != nil {
		return err
	}
	config.Server = server
	return nil
}
func setPassword(config *Configuration, cmdFlags *pflag.FlagSet) error {
	var password string
	var err error

	if cmdFlags.Changed("password") {
		password = flags.AdminPassword
	} else {
		password, err = selector.RunPasswordInput("Choose the password for your 'admin' user", "admin")
		if err != nil {
			return err
		}
	}

	config.AdminPassword = util.ComputeHashPassword(password)
	return nil
}
func setPort(config *Configuration, cmdFlags *pflag.FlagSet) error {
	if cmdFlags.Changed("port") {
		config.Port = flags.Port
		return nil
	}

	port, err := selector.RunTextInput("What HTTP port do you want to use (all the services are using the same port)?", "8080")
	if err != nil {
		return err
	}
	config.Port = port
	return nil
}
func setBinding(config *Configuration, cmdFlags *pflag.FlagSet) error {
	if cmdFlags.Changed("use-binding") {
		config.UseBinding = flags.UseBinding
	} else {
		useBinding, err := selector.RunYesNoSelector("Do you want to specify a custom binding IP for HTTP?", false)
		if err != nil {
			return err
		}
		config.UseBinding = useBinding
	}

	if config.UseBinding {
		if cmdFlags.Changed("binding-ip") {
			config.BindingIP = flags.BindingIP
		} else {
			bindingIP, err := selector.RunTextInput("What is the binding IP for HTTP?", "0.0.0.0")
			if err != nil {
				return err
			}
			config.BindingIP = bindingIP
		}
	} else {
		config.BindingIP = "0.0.0.0"
	}

	return nil
}
func setFTP(cfg *Configuration, cmdFlags *pflag.FlagSet) error {
	if cmdFlags.Changed("ftp") {
		cfg.UseFtp = flags.UseFtp
	} else {
		useFtp, err := selector.RunYesNoSelector("Do you want to use FTP (default port is 2121)?", false)
		if err != nil {
			return err
		}
		cfg.UseFtp = useFtp
	}

	// Always set a default FTP binding IP, even if FTP is not used
	cfg.FtpBindingIP = "0.0.0.0"

	if !cfg.UseFtp {
		return nil
	}

	// Handle FTP binding IP only if FTP is enabled
	customBindingFtp := cmdFlags.Changed("ftp-binding-ip")
	if !customBindingFtp {
		var err error
		customBindingFtp, err = selector.RunYesNoSelector("Do you want to specify a custom binding IP for FTP?", false)
		if err != nil {
			return err
		}
	}

	if customBindingFtp {
		if cmdFlags.Changed("ftp-binding-ip") {
			cfg.FtpBindingIP = flags.FtpBindingIP
		} else {
			ftpBindingIP, err := selector.RunTextInput("Enter the IP address to bind the FTP service:", "0.0.0.0")
			if err != nil {
				return err
			}
			cfg.FtpBindingIP = ftpBindingIP
		}
	}

	return nil
}
func setDatabase(config *Configuration, cmdFlags *pflag.FlagSet) error {
	config.DbPassword = "alfresco"

	if cmdFlags.Changed("database") {
		config.Database = flags.Database
		return nil
	}

	database, err := selector.RunSelector(
		"Which Database Engine do you want to use?",
		[]string{"postgres", "mariadb"},
	)
	if err != nil {
		return err
	}
	config.Database = database
	return nil
}
func setIndexing(config *Configuration, cmdFlags *pflag.FlagSet) error {
	// Index cross locale
	if cmdFlags.Changed("index-cross-locale") {
		config.IndexCrossLocale = flags.IndexCrossLocale
	} else {
		indexCrossLocale, err := selector.RunYesNoSelector("Are you using content in different languages (this is the most common scenario)?", true)
		if err != nil {
			return err
		}
		config.IndexCrossLocale = indexCrossLocale
	}

	// Index content
	if cmdFlags.Changed("index-content") {
		config.IndexContent = flags.IndexContent
	} else {
		indexContent, err := selector.RunYesNoSelector("Do you want to search in the content of the documents?", true)
		if err != nil {
			return err
		}
		config.IndexContent = indexContent
	}

	return nil
}
func setSolr(config *Configuration, cmdFlags *pflag.FlagSet) error {
	if cmdFlags.Changed("solr-comm") {
		config.SolrComm = flags.SolrComm
		return nil
	}

	solrComm, err := selector.RunSelector(
		"Which Solr communication method do you want to use?",
		[]string{"secret", "https"},
	)
	if err != nil {
		return err
	}
	config.SolrComm = solrComm
	if solrComm == "secret" {
		config.Secret = util.GenerateRandomString(32)
	}
	return nil
}
func setActiveMQ(config *Configuration, cmdFlags *pflag.FlagSet) error {
	if cmdFlags.Changed("activemq") {
		config.UseActiveMQ = flags.UseActiveMQ
	} else {
		useActiveMQ, err := selector.RunYesNoSelector("Do you want to use the Events service (ActiveMQ)?", false)
		if err != nil {
			return err
		}
		config.UseActiveMQ = useActiveMQ
	}

	if !config.UseActiveMQ {
		return nil
	}

	// Handle ActiveMQ credentials
	amqCredentials := cmdFlags.Changed("amq-user") && cmdFlags.Changed("amq-password")
	if !amqCredentials {
		var err error
		amqCredentials, err = selector.RunYesNoSelector("Do you want to use credentials for Events service (ActiveMQ)?", false)
		if err != nil {
			return err
		}
	}

	if amqCredentials {
		if cmdFlags.Changed("amq-user") {
			config.AmqUser = flags.AmqUser
		} else {
			amqUser, err := selector.RunTextInput("Enter the username for ActiveMQ", "admin")
			if err != nil {
				return err
			}
			config.AmqUser = amqUser
		}

		if cmdFlags.Changed("amq-password") {
			config.AmqPassword = flags.AmqPassword
		} else {
			amqPassword, err := selector.RunPasswordInput("Enter the password for ActiveMQ", "admin")
			if err != nil {
				return err
			}
			config.AmqPassword = amqPassword
		}
	}

	return nil
}
func setAddons(config *Configuration, cmdFlags *pflag.FlagSet) error {
	if len(flags.Addons) > 0 {
		config.Addons = flags.Addons
		return nil
	}

	selectedAddons, err := selector.RunSelectorWithOptions(
		"Select the addons to be installed",
		availableAddons,
		true,
	)
	if err != nil {
		return err
	}

	var addonCodes []string
	for _, addon := range selectedAddons {
		addonCodes = append(addonCodes, addon.Code)
	}
	config.Addons = addonCodes
	return nil
}
func setDockerVolume(config *Configuration, cmdFlags *pflag.FlagSet) error {
	// Hard rule for Windows ─ always Docker volumes
	if util.IsWindows() {
		fmt.Println("Host volumes are not recommended on Windows. Docker volumes will be used instead.")
		config.UseDockerVolume = true
		return nil
	}

	if cmdFlags.Changed("docker-volume") {
		config.UseDockerVolume = flags.UseDockerVolume
	} else {
		useDockerVolume, err := selector.RunYesNoSelector(
			"Do you want Docker to manage volume storage (recommended when dealing with permission issues)?",
			true,
		)
		if err != nil {
			return err
		}
		config.UseDockerVolume = useDockerVolume
	}

	// Warning for Linux host volumes
	if !config.UseDockerVolume && util.IsLinux() {
		fmt.Println("Warning: using host volumes on Linux may lead to permission issues. Consider Docker-managed volumes instead.")
	}

	return nil
}

// generateConfigFiles renders every *.tmpl in TemplateFS to an output file
// whose path is the same as the template path minus the "templates/" prefix
// and the ".tmpl" suffix.
func generateConfigFiles(cfg *Configuration) error {
	// 1 - collect all *.tmpl files inside the embedded FS
	var paths []string
	if err := fs.WalkDir(TemplateFS, "templates", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".tmpl") {
			paths = append(paths, p)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("walk embedded templates: %w", err)
	}

	// 2 - create a template root and register every file under its unique path
	root := template.New("root").Funcs(template.FuncMap{
		"formatMem": util.FormatMem,
		"hasAddon":  func(code string) bool { return slices.Contains(cfg.Addons, code) },
	})

	for _, src := range paths {
		data, err := fs.ReadFile(TemplateFS, src)
		if err != nil {
			return fmt.Errorf("read %s: %w", src, err)
		}
		name := strings.TrimPrefix(src, "templates/") // e.g. "alfresco/Dockerfile.tmpl"
		if _, err := root.New(name).Parse(string(data)); err != nil {
			return fmt.Errorf("parse %s: %w", src, err)
		}
	}

	// 3 - render each template to its output file
	for _, src := range paths {
		rel := strings.TrimPrefix(src, "templates/") // "alfresco/Dockerfile.tmpl"

		if filepath.Base(rel) == "create_volumes.sh.tmpl" {
			if runtime.GOOS == "linux" {
				fmt.Printf("\x1b[33;1mWARNING: Before starting Alfresco for the first time, run 'sudo ./create-volumes.sh'\x1b[0m\n")
			} else {
				continue
			}
		}

		outPath := strings.TrimSuffix(rel, ".tmpl") // "alfresco/Dockerfile"

		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", filepath.Dir(outPath), err)
		}
		out, err := os.Create(outPath)
		if err != nil {
			return fmt.Errorf("create %s: %w", outPath, err)
		}

		if err := root.Lookup(rel).Execute(out, cfg); err != nil {
			out.Close()
			return fmt.Errorf("execute %s: %w", src, err)
		}
		out.Close()
	}

	// 4 - handle binary files and addons
	if cfg.Database == "mariadb" {
		if err := copyBinary("templates/libs/mariadb-java-client-2.7.4.jar",
			"libs/mariadb-java-client-2.7.4.jar"); err != nil {
			return fmt.Errorf("copy mariadb driver: %w", err)
		}
	}
	if !cfg.UseActiveMQ {
		if err := copyBinary("templates/libs/activemq-broker-5.18.3.jar",
			"libs/activemq-broker-5.18.3.jar"); err != nil {
			return fmt.Errorf("copy ActiveMQ local library: %w", err)
		}
	}
	if cfg.SolrComm == "https" {
		if err := copyFolder("templates/keystores", "keystores", TemplateFS); err != nil {
			return fmt.Errorf("copy mTLS keystores: %w", err)
		}
	}

	// 5 - copy addons
	if slices.Contains(cfg.Addons, "alf-tengine-ocr") {
		if err := copyBinary("templates/addons/jars/embed-metadata-action-1.0.0.jar",
			"alfresco/modules/jars/tengine-ocr-1.1.0.jar"); err != nil {
			return fmt.Errorf("copy TEngine OCR repository addon: %w", err)
		}
	}
	if slices.Contains(cfg.Addons, "ootbee-support-tools") {
		if err := copyBinary("templates/addons/amps/support-tools-repo-1.2.3.0-SNAPSHOT-amp.amp",
			"alfresco/modules/amps/support-tools-repo-1.2.3.0-SNAPSHOT-amp.amp"); err != nil {
			return fmt.Errorf("copy OOTB Tools repository addon: %w", err)
		}
		if err := copyBinary("templates/addons/amps_share/support-tools-share-1.2.3.0-SNAPSHOT-amp.amp",
			"share/modules/amps/support-tools-share-1.2.3.0-SNAPSHOT-amp.amp"); err != nil {
			return fmt.Errorf("copy OOTB Tools share addon: %w", err)
		}
	}
	if slices.Contains(cfg.Addons, "share-site-creators") {
		if err := copyBinary("templates/addons/amps/share-site-creators-repo-0.0.8.amp",
			"alfresco/modules/amps/share-site-creators-repo-0.0.8.amp"); err != nil {
			return fmt.Errorf("copy Share Site Creators repository addon: %w", err)
		}
		if err := copyBinary("templates/addons/amps_share/share-site-creators-share-0.0.8.amp",
			"share/modules/amps/share-site-creators-share-0.0.8.amp"); err != nil {
			return fmt.Errorf("copy Share Site Creators share addon: %w", err)
		}
	}
	if slices.Contains(cfg.Addons, "share-site-space-templates") {
		if err := copyBinary("templates/addons/amps/share-site-space-templates-repo-1.1.4-SNAPSHOT.amp",
			"alfresco/modules/amps/share-site-space-templates-repo-1.1.4-SNAPSHOT.amp"); err != nil {
			return fmt.Errorf("copy Share Site Space Templates repository addon: %w", err)
		}
	}
	if slices.Contains(cfg.Addons, "esign-cert") {
		if err := copyBinary("templates/addons/amps/esign-cert-repo-1.8.4.amp",
			"alfresco/modules/amps/esign-cert-repo-1.8.4.amp"); err != nil {
			return fmt.Errorf("copy eSign Cert repository addon: %w", err)
		}
		if err := copyBinary("templates/addons/amps_share/esign-cert-share-1.8.4.amp",
			"share/modules/amps/esign-cert-share-1.8.4.amp"); err != nil {
			return fmt.Errorf("copy eSign Cert share addon: %w", err)
		}
	}
	if slices.Contains(cfg.Addons, "share-online-edition") {
		if err := copyBinary("templates/addons/amps_share/zk-libreoffice-addon-share.amp",
			"share/modules/amps/zk-libreoffice-addon-share.amp"); err != nil {
			return fmt.Errorf("copy Share Online Edition share addon: %w", err)
		}
	}

	return nil
}

func copyBinary(srcPath string, outPath string) error {
	in, err := TemplateFS.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open embedded binary %s: %w", srcPath, err)
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(outPath), err)
	}

	out, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create %s: %w", outPath, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy %s: %w", outPath, err)
	}

	return nil
}

// CopyFolder copies all files and directories from srcDir to dstDir
func copyFolder(srcDir, dstDir string, sourceFS fs.FS) error {
	return fs.WalkDir(sourceFS, srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath := strings.TrimPrefix(path, srcDir)
		targetPath := filepath.Join(dstDir, relPath)

		if d.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}

		in, err := sourceFS.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()

		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}

		out, err := os.Create(targetPath)
		if err != nil {
			return err
		}
		defer out.Close()

		_, err = io.Copy(out, in)
		return err
	})
}

/*
	 Example usage
	   To run the Docker Compose command with specific flags, you can use:
		alf-cli docker-compose \
		  --version=25.2 \
		  --https=false \
		  --server=localhost \
		  --password=admin \
		  --port=8080 \
		  --use-binding=false \
		  --ftp=false \
		  --database=postgres \
		  --index-content=true \
		  --index-cross-locale=true \
		  --solr-comm=secret \
		  --activemq=false \
		  --addons=js-console \
		  --docker-volume=false
*/
func init() {
	// Basic configuration flags
	dockerComposeCmd.Flags().StringVar(&flags.Version, "version", "", "ACS version (25.2, 25.1)")
	dockerComposeCmd.Flags().BoolVar(&flags.HTTPS, "https", false, "Enable HTTPS")
	dockerComposeCmd.Flags().StringVar(&flags.Server, "server", "", "Server name")
	dockerComposeCmd.Flags().StringVar(&flags.AdminPassword, "password", "", "Admin password")
	dockerComposeCmd.Flags().StringVar(&flags.Port, "port", "", "HTTP port")

	// Network binding flags
	dockerComposeCmd.Flags().BoolVar(&flags.UseBinding, "use-binding", false, "Use custom HTTP binding IP")
	dockerComposeCmd.Flags().StringVar(&flags.BindingIP, "binding-ip", "0.0.0.0", "HTTP binding IP")

	// FTP configuration flags
	dockerComposeCmd.Flags().BoolVar(&flags.UseFtp, "ftp", false, "Enable FTP")
	dockerComposeCmd.Flags().StringVar(&flags.FtpBindingIP, "ftp-binding-ip", "0.0.0.0", "FTP binding IP")

	// Database and indexing flags
	dockerComposeCmd.Flags().StringVar(&flags.Database, "database", "postgres", "Database Engine (postgres, mariadb)")
	dockerComposeCmd.Flags().BoolVar(&flags.IndexCrossLocale, "index-cross-locale", true, "Enable cross-locale indexing")
	dockerComposeCmd.Flags().BoolVar(&flags.IndexContent, "index-content", true, "Enable full-text indexing")
	dockerComposeCmd.Flags().StringVar(&flags.SolrComm, "solr-comm", "", "Solr communication method (secret|https)")

	// ActiveMQ configuration flags
	dockerComposeCmd.Flags().BoolVar(&flags.UseActiveMQ, "activemq", false, "Enable ActiveMQ")
	dockerComposeCmd.Flags().StringVar(&flags.AmqUser, "amq-user", "admin", "ActiveMQ username")
	dockerComposeCmd.Flags().StringVar(&flags.AmqPassword, "amq-password", "admin", "ActiveMQ password")

	// Addon and volume flags
	dockerComposeCmd.Flags().StringSliceVarP(&flags.Addons, "addons", "a", nil, "Comma-separated list of addon codes")
	dockerComposeCmd.Flags().BoolVar(&flags.UseDockerVolume, "docker-volume", true, "Use Docker-managed volumes")

	rootCmd.AddCommand(dockerComposeCmd)
}
