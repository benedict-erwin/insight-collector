package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/benedict-erwin/insight-collector/pkg/maxmind"
	"github.com/benedict-erwin/insight-collector/pkg/utils"
	"github.com/spf13/cobra"
)

// 📋 Available Public Functions

// // From pkg/maxmind/maxmind.go

// // Manual database update check
// func CheckForUpdates() error

// // Force download specific database ("city" atau "asn")
// func ForceDownload(dbType string) error

// // Get current download status
// func GetDownloadStatus() map[string]interface{}

// // Regular service functions (sudah ada)
// func GetDatabaseInfo() *DatabaseInfo
// func LookupCity(ip net.IP) *GeoLocation
// func LookupASN(ip net.IP) *ASNInfo
// func Health() error

// # Check for updates (all databases)
// ./insight-collector maxmind check-updates

// # Check specific database
// ./insight-collector maxmind check-updates --database city

// # Force download
// ./insight-collector maxmind download --database city --force

// # Show status with auth info (masked)
// ./insight-collector maxmind status

// Check for updates
var maxmindCheckCmd = &cobra.Command{
	Use:   "check-updates",
	Short: "Check for MaxMind database updates",
	Long:  "Check if new database version is available for download",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Init maxmind (will init config internally)
		if err := maxmind.InitMinimalForCLI(); err != nil {
			return fmt.Errorf("failed to init MaxMind: %w", err)
		}
		defer maxmind.Close()

		// Check for updates
		utils.ClearScreen()
		fmt.Println("Checking for MaxMind database updates...")
		if err := maxmind.CheckForUpdates(); err != nil {
			return fmt.Errorf("update check failed: %w", err)
		}

		//  Check completed
		fmt.Println("Database update check completed")
		return nil
	},
}

// Download databases
var maxmindDownloadCmd = &cobra.Command{
	Use:   "download [city|asn|all]",
	Short: "Force download MaxMind database",
	Long:  "Force download specific MaxMind database or all databases",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("requires exactly 1 arg city|asn|all")
		}
		validArgs := []string{"city", "asn", "all"}
		for _, valid := range validArgs {
			if args[0] == valid {
				return nil
			}
		}
		return fmt.Errorf("invalid argument %s, must be one of: %v", args[0], validArgs)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Init maxmind (will init config internally)
		if err := maxmind.InitMinimalForCLI(); err != nil {
			return fmt.Errorf("failed to init MaxMind: %w", err)
		}
		defer maxmind.Close()

		utils.ClearScreen()
		dbType := strings.ToLower(args[0])
		if dbType == "all" {
			// Download both databases
			fmt.Println("Force downloading all databases...")

			// Download City database
			fmt.Println("Downloading City database...")
			if err := maxmind.ForceDownload("city"); err != nil {
				fmt.Printf("Error downloading city database: %v\n", err)
			} else {
				fmt.Println("City database downloaded successfully")
			}

			// Download ASN database
			fmt.Println("Downloading ASN database...")
			if err := maxmind.ForceDownload("asn"); err != nil {
				fmt.Printf("Error downloading ASN database: %v\n", err)
			} else {
				fmt.Println("ASN database downloaded successfully")
			}

			fmt.Println("All database downloads completed")
			return nil
		} else {
			// Download City/ASN database
			fmt.Printf("Downloading %s database...\n", utils.UcFirst(dbType))
			if err := maxmind.ForceDownload(dbType); err != nil {
				fmt.Printf("Error downloading %s database: %v\n", utils.UcFirst(dbType), err)
			}
			fmt.Printf("%s database downloaded successfully", utils.UcFirst(dbType))
			return nil
		}
	},
}

var maxmindStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show MaxMind service status",
	Long:  "Display current status of MaxMind service and databases",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Init maxmind (will init config internally)
		if err := maxmind.InitMinimalForCLI(); err != nil {
			return fmt.Errorf("failed to init MaxMind: %w", err)
		}
		defer maxmind.Close()

		// Get Status
		status := maxmind.GetDownloadStatusCLI()

		// If using JSON Output
		if jsonOutput, _ := cmd.Flags().GetBool("json"); jsonOutput {
			utils.ClearScreen()
			output, _ := json.MarshalIndent(status, "", "  ")
			fmt.Println(string(output))
			return nil
		}

		// Display service status
		utils.ClearScreen()
		fmt.Printf("MaxMind Service Status\n")
		fmt.Printf("=====================\n")
		fmt.Printf("Enabled: %v\n", status["enabled"])

		if downloaderEnabled, ok := status["enabled"].(bool); ok && downloaderEnabled {
			fmt.Printf("Storage Path: %s\n", status["storage_path"])
			if client, ok := status["client"].(string); ok {
				fmt.Printf("Client: %s\n", client)
			}
		}

		// Display database info
		fmt.Printf("\nDatabase Information\n")
		fmt.Printf("===================\n")

		table := tablewriter.NewWriter(os.Stdout)
		table.Header([]string{"Database", "Exists", "Size", "Modified", "Status"})

		if databases, ok := status["databases"].(map[string]interface{}); ok {
			for dbType, info := range databases {
				if dbInfo, ok := info.(map[string]interface{}); ok {
					name := dbInfo["name"].(string)

					if dbStatus, ok := dbInfo["status"].(map[string]interface{}); ok {
						exists := "❌"
						size := "N/A"
						modified := "N/A"
						statusText := "Missing"

						if dbStatus["exists"].(bool) {
							exists = "✅"
							statusText = "Available"

							if sizeVal, ok := dbStatus["size"].(int64); ok {
								size = fmt.Sprintf("%.2f MB", float64(sizeVal)/(1024*1024))
							}

							if modTime, ok := dbStatus["mod_time"].(time.Time); ok {
								modified = modTime.Format("2006-01-02 15:04:05")
							}
						}

						table.Append([]string{
							fmt.Sprintf("%s (%s)", strings.ToUpper(dbType), name),
							exists,
							size,
							modified,
							statusText,
						})
					}
				}
			}
		}

		// Render dbinfo table
		table.Render()

		// Show health status
		fmt.Printf("\nHealth Status\n")
		fmt.Printf("============\n")
		if err := maxmind.Health(); err != nil {
			fmt.Printf("❌ Health: %v\n", err)
		} else {
			fmt.Printf("✅ Health: OK\n")
		}
		return nil
	},
}

var maxmindInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show detailed database information",
	Long:  "Display detailed information about loaded MaxMind databases",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Initialize MaxMind (will init config internally)
		if err := maxmind.InitMinimalForCLI(); err != nil {
			return fmt.Errorf("failed to initialize MaxMind: %w", err)
		}
		defer maxmind.Close()

		// Get database info
		dbInfo, err := maxmind.GetDatabaseInfoCLI()
		if err != nil {
			return err
		}

		// If using JSON Output
		if jsonOutput, _ := cmd.Flags().GetBool("json"); jsonOutput {
			utils.ClearScreen()
			output, _ := json.MarshalIndent(dbInfo, "", "  ")
			fmt.Println(string(output))
			return nil
		}

		// Pretty print database info
		utils.ClearScreen()
		fmt.Printf("MaxMind Database Information\n")
		fmt.Printf("===========================\n")
		fmt.Printf("Enabled: %v\n", dbInfo.Enabled)
		fmt.Printf("Loaded At: %s\n", dbInfo.LoadedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Reload Count: %d\n", dbInfo.ReloadCount)

		if dbInfo.Enabled {
			fmt.Printf("\nCity Database:\n")
			fmt.Printf("  Path: %s\n", dbInfo.CityDBPath)
			fmt.Printf("  Size: %.2f MB\n", float64(dbInfo.CityDBSize)/(1024*1024))
			fmt.Printf("  Modified: %s\n", dbInfo.CityDBModTime.Format("2006-01-02 15:04:05"))

			fmt.Printf("\nASN Database:\n")
			fmt.Printf("  Path: %s\n", dbInfo.ASNDBPath)
			fmt.Printf("  Size: %.2f MB\n", float64(dbInfo.ASNDBSize)/(1024*1024))
			fmt.Printf("  Modified: %s\n", dbInfo.ASNDBModTime.Format("2006-01-02 15:04:05"))
			fmt.Printf("\n")
		}

		return nil
	},
}

var maxmindLookupCmd = &cobra.Command{
	Use:   "lookup [ip]",
	Short: "Test IP address lookup",
	Long:  "Perform GeoIP lookup for testing MaxMind databases",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ipAddr := args[0]

		// Initialize MaxMind (will init config internally)
		if err := maxmind.InitMinimalForCLI(); err != nil {
			return fmt.Errorf("failed to initialize MaxMind: %w", err)
		}
		defer maxmind.Close()

		// Perform lookups
		geoLocation := maxmind.LookupCityFromString(ipAddr)
		asnInfo := maxmind.LookupASNFromString(ipAddr)

		// If using JSON Output
		if jsonOutput, _ := cmd.Flags().GetBool("json"); jsonOutput {
			result := map[string]interface{}{
				"ip":  ipAddr,
				"geo": geoLocation,
				"asn": asnInfo,
			}
			utils.ClearScreen()
			output, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(output))
			return nil
		}

		// Pretty print results
		utils.ClearScreen()
		fmt.Printf("GeoIP Lookup Results for %s\n", ipAddr)
		fmt.Printf("=============================\n")

		fmt.Printf("Geographic Information:\n")
		fmt.Printf("  Country: %s (%s)\n", geoLocation.Country, geoLocation.CountryCode)
		fmt.Printf("  City: %s\n", geoLocation.City)
		fmt.Printf("  Region: %s\n", geoLocation.Region)
		fmt.Printf("  Coordinates: %.4f, %.4f\n", geoLocation.Latitude, geoLocation.Longitude)
		fmt.Printf("  Timezone: %s\n", geoLocation.Timezone)

		fmt.Printf("\nASN Information:\n")
		fmt.Printf("  ASN: %d\n", asnInfo.ASN)
		fmt.Printf("  Organization: %s\n", asnInfo.Organization)

		return nil
	},
}

var maxmindCmd = &cobra.Command{
	Use:   "maxmind",
	Short: "MaxMind GeoIP database management",
	Long:  "Commands for managing MaxMind GeoIP databases and performing lookups",
}

func init() {
	// Add subcommands
	maxmindCmd.AddCommand(maxmindCheckCmd)
	maxmindCmd.AddCommand(maxmindDownloadCmd)
	maxmindCmd.AddCommand(maxmindStatusCmd)
	maxmindCmd.AddCommand(maxmindInfoCmd)
	maxmindCmd.AddCommand(maxmindLookupCmd)

	// Command flag
	maxmindStatusCmd.Flags().BoolP("json", "j", false, "Output info in JSON format")
	maxmindInfoCmd.Flags().BoolP("json", "j", false, "Output info in JSON format")
	maxmindLookupCmd.Flags().BoolP("json", "j", false, "Output info in JSON format")

	// Add root command
	rootCmd.AddCommand(maxmindCmd)
}
